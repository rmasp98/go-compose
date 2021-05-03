package compose_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/volume"
	"github.com/rmasp98/go-compose/compose"
)

func TestInvalidVolumeConfig(t *testing.T) {
	if _, err := compose.NewVolume("invalid config"); err == nil {
		t.Errorf("Should return error but returned nothing")
	}
}

func TestCustomVolumeConfigurable(t *testing.T) {
	custom := customVolume()
	vol, _ := compose.NewVolume(custom)
	if err := verifyVolume(vol, custom); err != nil {
		t.Errorf(err.Error())
	}
}

func TestExternalVolumeReturnsEmptyVolumeCreate(t *testing.T) {
	vol, _ := compose.NewVolume(map[string]interface{}{"name": "test", "external": true})
	if !reflect.DeepEqual(vol.GetCreateConfig(), volume.VolumeCreateBody{}) {
		t.Errorf("Did not return empty VolumeCreateBody")
	}
}

func TestReturnsErrorIfNotValidTypeInVolume(t *testing.T) {
	options := []string{
		"driver", "driver_opts", "external", "labels", "name",
	}
	for _, option := range options {
		if _, err := compose.NewVolume(map[string]interface{}{option: 0}); err == nil {
			t.Errorf("Should have returned an error for \"%s\" but returned nothing", option)
		}
	}
}

func TestReturnsNameForExternalVolume(t *testing.T) {
	vol, _ := compose.NewVolume(map[string]interface{}{"external": true, "name": "Test"})
	name, external := vol.GetExternalName()
	if name != "Test" || !external {
		t.Errorf("Name was not correct: \"%s\"", name)
	}
}

func TestReturnsEmptyForNonExternalVolume(t *testing.T) {
	vol, _ := compose.NewVolume(map[string]interface{}{"external": false, "name": "Test"})
	name, external := vol.GetExternalName()
	if name != "" || external {
		t.Errorf("Name was not correct: \"%s\"", name)
	}
}

func customVolume() map[string]interface{} {
	return map[string]interface{}{
		"driver":      "local",
		"driver_opts": map[string]string{"Test": "Me"},
		"labels":      map[string]string{"Test": "Me"},
		"name":        "Test",
	}
}

func verifyVolume(vol compose.Volume, settings map[string]interface{}) error {
	config := vol.GetCreateConfig()

	if config.Driver != settings["driver"] {
		return fmt.Errorf("Driver was set to \"%s\" but should be \"%s\"", config.Driver, settings["driver"])
	}
	if !reflect.DeepEqual(config.DriverOpts, settings["driver_opts"]) {
		return fmt.Errorf("Driver options were set to %v but should be %v",
			config.DriverOpts, settings["driver_opts"])
	}
	if !reflect.DeepEqual(config.Labels, settings["labels"]) {
		return fmt.Errorf("Driver options were set to %v but should be %v",
			config.Labels, settings["labels"])
	}
	if config.Name != settings["name"] {
		return fmt.Errorf("Name was set to \"%s\" but should be \"%s\"", config.Name, settings["name"])
	}
	return nil
}
