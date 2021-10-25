package compose

import (
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
)

// TODO: Services, configs

type Stack struct {
	services map[string]Service
	networks map[string]Network
	volumes  map[string]Volume
}

func NewStack(composeData interface{}) (Stack, error) {
	config := composeData.(map[string]interface{})
	if err := verifyVersion(config); err != nil {
		return Stack{}, err
	}

	services := make(map[string]Service)
	if err := parseConfig("services", config, func(name string, config interface{}) error {
		var err error
		services[name], err = NewService(config)
		return err
	}); err != nil {
		return Stack{}, err
	}

	networks := make(map[string]Network)
	if err := parseConfig("networks", config, func(name string, config interface{}) error {
		var err error
		networks[name], err = NewNetwork(config)
		return err
	}); err != nil {
		return Stack{}, err
	}

	volumes := make(map[string]Volume)
	if err := parseConfig("volumes", config, func(name string, config interface{}) error {
		var err error
		volumes[name], err = NewVolume(config)
		return err
	}); err != nil {
		return Stack{}, err
	}

	return Stack{services, networks, volumes}, nil
}

// TODO: probably need a list for each of below

func (s Stack) GetNetworkCreate(name string) types.NetworkCreate {
	return s.networks[name].GetCreateConfig()
}

func (s Stack) GetVolumeCreate(name string) volume.VolumeCreateBody {
	return s.volumes[name].GetCreateConfig()
}

func (s Stack) GetServiceContainerCreate(name string) container.Config {
	return s.services[name].GetContainerConfig()
}

func verifyVersion(config map[string]interface{}) error {
	if version, hasVersion := config["version"]; hasVersion {
		versionNum, err := strconv.ParseFloat(version.(string), 32)
		if err != nil {
			return fmt.Errorf("version value not valid")
		}
		if versionNum < 3.0 || versionNum > 3.8 {
			return fmt.Errorf("Incorrect version")
		}
	}
	return nil
}

func parseConfig(thing string, mainConfig map[string]interface{}, parser func(string, interface{}) error) error {
	if iface, exists := mainConfig[thing]; exists {
		if config, isMap := iface.(map[string]interface{}); isMap {
			for name, data := range config {
				if err := parser(name, data); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("%s should be a map", thing)
		}
	}
	return nil
}
