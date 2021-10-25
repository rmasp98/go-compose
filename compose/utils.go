package compose

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
)

type setValueMapping struct {
	name     string
	target   interface{}
	convert  func(interface{}) (interface{}, error)
	validate func(interface{}) error
}

func setValues(values []setValueMapping, config map[string]interface{}) error {
	for _, mapping := range values {
		if err := setValue(mapping.target, mapping.name, config, mapping.convert, mapping.validate); err != nil {
			return err
		}
	}
	return nil
}

func setValue(target interface{}, name string, config map[string]interface{}, convert func(interface{}) (interface{}, error), validate func(interface{}) error) error {
	if iface, isSet := config[name]; isSet {
		if convert != nil {
			var err error
			if iface, err = convert(iface); err != nil {
				return fmt.Errorf("%s: %s", name, err.Error())
			}
		}

		if validate != nil {
			if err := validate(iface); err != nil {
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
	case []interface{}:
		list, err := convertToStringList(input)
		if err != nil {
			return nil, err
		}
		for _, config := range list.([]string) {
			vol, err := loader.ParseVolume(config)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, vol)
		}
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
	// TODO: add validation functions
	mapping := []setValueMapping{
		{"type", &volume.Type, nil, nil},
		{"source", &volume.Source, nil, nil},
		{"target", &volume.Target, nil, nil},
		{"read_only", &volume.ReadOnly, nil, nil},
		{"bind", &volume.Bind.Propagation, convertPropagation, nil},
		{"volume", &volume.Volume.NoCopy, convertVolNoCopy, nil},
		{"tmpfs", &volume.Tmpfs.Size, nil, nil},
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

func convertToStringMap(input interface{}) (interface{}, error) {
	return parseStringMap(input)
}

func parseStringMap(input interface{}) (map[string]string, error) {
	stringMap := make(map[string]string)
	switch input := input.(type) {
	case map[string]interface{}:
		for key, value := range input {
			value, err := getString(value)
			if err != nil {
				return nil, fmt.Errorf("%s - %s", key, err.Error())
			}
			stringMap[key] = value
		}
	case []interface{}:
		stringList, err := parseStringList(input)
		if err != nil {
			return nil, err
		}
		for _, mapping := range stringList {
			split := strings.Split(mapping, "=")
			switch len(split) {
			case 1:
				stringMap[split[0]] = ""
			case 2:
				stringMap[split[0]] = split[1]
			default:
				return nil, fmt.Errorf("Connot convert []string to map[string]string")
			}
		}
	default:
		return nil, fmt.Errorf("type %t not supported", input)
	}

	return stringMap, nil
}

func convertToStringList(input interface{}) (interface{}, error) {
	return parseStringList(input)
}

func parseStringList(input interface{}) ([]string, error) {
	switch input := input.(type) {
	case []interface{}:
		var output []string
		for _, value := range input {
			value, err := getString(value)
			if err != nil {
				return nil, err
			}
			output = append(output, value)
		}
		return output, nil
	case map[string]interface{}:
		var output []string
		for key, value := range input {
			value, err := getString(value)
			if err != nil {
				return nil, err
			}
			output = append(output, key+"="+value)
		}
		return output, nil
	case string:
		return strings.Split(input, " "), nil
	}

	return nil, fmt.Errorf("Cannot convert to []string")
}

func getString(source interface{}) (string, error) {
	switch source := source.(type) {
	case string:
		return source, nil
	case int:
		return strconv.Itoa(source), nil
	case bool:
		return strconv.FormatBool(source), nil
	case nil:
		return "", nil
	}
	return "", fmt.Errorf("Unsupported type")
}
