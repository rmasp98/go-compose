package compose

import (
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

type Volume struct {
	driver     string
	driverOpts map[string]string
	labels     map[string]string
	external   bool
	name       string
}

func NewVolume(config interface{}) (Volume, error) {
	volume := Volume{}

	if volumeConfig, isMap := config.(map[string]interface{}); isMap {
		if err := setValue(&volume.driver, "driver", volumeConfig, nil); err != nil {
			return volume, err
		}

		if err := setValue(&volume.driverOpts, "driver_opts", volumeConfig, nil); err != nil {
			return volume, err
		}

		if err := setValue(&volume.labels, "labels", volumeConfig, nil); err != nil {
			return volume, err
		}

		if err := setValue(&volume.name, "name", volumeConfig, nil); err != nil {
			return volume, err
		}

		if err := setValue(&volume.external, "external", volumeConfig, nil); err != nil {
			return volume, err
		}
	} else {
		return volume, fmt.Errorf("volume should be a map")
	}

	return volume, nil
}

func (v Volume) GetCreateConfig() volume.VolumeCreateBody {
	if v.external {
		return volume.VolumeCreateBody{}
	}
	return volume.VolumeCreateBody{
		Driver:     v.driver,
		DriverOpts: v.driverOpts,
		Labels:     v.labels,
		Name:       v.name,
	}
}

func (v Volume) GetExternalName() (string, bool) {
	if v.external {
		return v.name, v.external
	}
	return "", v.external
}
