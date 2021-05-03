package compose

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
)

type setValueMapping struct {
	name    string
	target  interface{}
	convert func(interface{}) (interface{}, error)
}

func setValues(values []setValueMapping, config map[string]interface{}) error {
	for _, mapping := range values {
		if err := setValue(mapping.target, mapping.name, config, mapping.convert); err != nil {
			return err
		}
	}
	return nil
}

func setValue(target interface{}, name string, config map[string]interface{}, convert func(interface{}) (interface{}, error)) error {
	if iface, isSet := config[name]; isSet {
		if convert != nil {
			var err error
			if iface, err = convert(iface); err != nil {
				return err
			}
		}

		// Use switch if this set function does not work
		if !set(target, iface) {
			// TODO: Must be a better way to do this
			t := reflect.TypeOf(target).String()
			return fmt.Errorf("%s should be type %s", name, t[1:])
		}
	}
	return nil
}

func set(target interface{}, value interface{}) bool {
	if target == nil || value == nil {
		return false
	}
	t := reflect.ValueOf(target).Elem()
	v := reflect.ValueOf(value)
	if !v.Type().AssignableTo(t.Type()) {
		return false
	}
	t.Set(v)
	return true
}

func convertDuration(input interface{}) (interface{}, error) {
	if durationString, isStr := input.(string); isStr {
		duration, err := time.ParseDuration(durationString)
		if err != nil {
			return nil, err
		}
		return duration, nil
	}
	return nil, fmt.Errorf("duration should be a string")
}

// TODO: parse the different volume types and filter out the volType
func parseVolumes(input interface{}) ([]types.ServiceVolumeConfig, error) {
	var volumes []types.ServiceVolumeConfig
	switch input := input.(type) {
	case []string:
		for _, config := range input {
			vol, err := loader.ParseVolume(config)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, vol)
		}
	case []map[string]interface{}:
		for _, config := range input {
			vol, err := parseVolumeMap(config)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, vol)
		}
	default:
		return volumes, fmt.Errorf("volumes must be a string list or a map")
	}
	return volumes, nil
}

func parseVolumeMap(config map[string]interface{}) (types.ServiceVolumeConfig, error) {
	volume := types.ServiceVolumeConfig{
		Bind:   new(types.ServiceVolumeBind),
		Volume: new(types.ServiceVolumeVolume),
		Tmpfs:  new(types.ServiceVolumeTmpfs),
	}
	mapping := []setValueMapping{
		{"type", &volume.Type, nil},
		{"source", &volume.Source, nil},
		{"target", &volume.Target, nil},
		{"read_only", &volume.ReadOnly, nil},
		{"bind", &volume.Bind.Propagation, convertPropagation},
		{"volume", &volume.Volume.NoCopy, convertVolNoCopy},
		{"tmpfs", &volume.Tmpfs.Size, nil},
	}
	if err := setValues(mapping, config); err != nil {
		return volume, err
	}
	return volume, nil
}

func convertPropagation(input interface{}) (interface{}, error) {
	if config, isMap := input.(map[string]bool); isMap {
		propagation, isSet := config["propagation"]
		if !isSet {
			return nil, fmt.Errorf("volume bind missing proporation")
		}
		return propagation, nil
	}
	return nil, nil
}

func convertVolNoCopy(input interface{}) (interface{}, error) {
	if config, isMap := input.(map[string]bool); isMap {
		noCopy, isSet := config["nocopy"]
		if !isSet {
			return nil, fmt.Errorf("nocopy was not set in the volume definition")
		}
		return noCopy, nil
	}
	return nil, fmt.Errorf("volume should be a map[string]bool")
}

func getVolumeString(config types.ServiceVolumeConfig) string {
	var volume string
	if config.Source != "" {
		volume += config.Source + ":"
	}
	if config.Target != "" {
		volume += config.Target + ":"
	}
	if config.ReadOnly {
		volume += "ro,"
	} else {
		volume += "rw,"
	}
	if config.Type == "volume" && config.Volume != nil && config.Volume.NoCopy {
		volume += "nocopy,"
	}
	return strings.TrimRight(volume, ",")
}
