package compose

import (
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
)

// TODO: Services, configs

type Stack struct {
	networks map[string]Network
	volumes  map[string]Volume
}

func NewStack(composeData interface{}) (Stack, error) {
	config := composeData.(map[string]interface{})
	if err := verifyVersion(config); err != nil {
		return Stack{}, err
	}

	//TODO: should be able to put this functionality into generic function
	networks := make(map[string]Network)
	if networkInterface, exists := config["networks"]; exists {
		if networkConfig, isMap := networkInterface.(map[string]interface{}); isMap {
			for name, network := range networkConfig {
				var err error
				if networks[name], err = NewNetwork(network); err != nil {
					return Stack{}, fmt.Errorf("%s: %s", name, err.Error())
				}
			}
		} else {
			return Stack{}, fmt.Errorf("networks should be a map")
		}
	}

	volumes := make(map[string]Volume)
	if volumeInterface, exists := config["volumes"]; exists {
		if volumeConfig, isMap := volumeInterface.(map[string]interface{}); isMap {
			for name, volume := range volumeConfig {
				var err error
				if volumes[name], err = NewVolume(volume); err != nil {
					return Stack{}, fmt.Errorf("%s: %s", name, err.Error())
				}
			}
		} else {
			return Stack{}, fmt.Errorf("volumes should be a map")
		}
	}

	return Stack{networks, volumes}, nil
}

func (s Stack) GetNetworkCreate(name string) types.NetworkCreate {
	return s.networks[name].GetCreateConfig()
}

func (s Stack) GetVolumeCreate(name string) volume.VolumeCreateBody {
	return s.volumes[name].GetCreateConfig()
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
