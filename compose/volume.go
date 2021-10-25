package compose

import (
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

type Volume struct {
	data     volume.VolumeCreateBody
	external bool
}

func NewVolume(config interface{}) (Volume, error) {
	volume := Volume{}

	if volumeConfig, isMap := config.(map[string]interface{}); isMap {
		// TODO: add validation functions
		mapping := []setValueMapping{
			{"driver", &volume.data.Driver, nil, nil},
			{"driver_opts", &volume.data.DriverOpts, convertToStringMap, nil},
			{"labels", &volume.data.Labels, convertToStringMap, nil},
			{"name", &volume.data.Name, nil, nil},
			{"external", &volume.external, nil, nil},
		}
		if err := setValues(mapping, volumeConfig); err != nil {
			return volume, err
		}
	} else if config != nil {
		return volume, fmt.Errorf("volume should be a map")
	}

	return volume, nil
}

func (v Volume) GetCreateConfig() volume.VolumeCreateBody {
	if !v.external {
		return v.data
	}
	return volume.VolumeCreateBody{}
}

func (v Volume) GetExternalName() (string, bool) {
	return v.data.Name, v.external
}
