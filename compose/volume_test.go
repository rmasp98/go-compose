package compose_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/volume"
	"github.com/rmasp98/go-compose/compose"
)

func TestConfigNotMapReturnsError(t *testing.T) {
	if _, err := compose.NewVolume("invalid config"); err == nil {
		t.Errorf("Should return error but returned nothing")
	}
}

func TestCustomVolumeConfigurable(t *testing.T) {
	for _, mapping := range getVolumeMapping() {
		vol, err := compose.NewVolume(map[string]interface{}{mapping.name: mapping.source})
		if err != nil {
			t.Error(err)
			return
		}
		if err := verifyVolumeConfig(mapping.name, mapping.expected, vol.GetCreateConfig()); err != nil {
			t.Error(err)
		}
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

func TestCanParseEmptyVolume(t *testing.T) {
	if _, err := compose.NewVolume(nil); err != nil {
		t.Error(err)
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

func getVolumeMapping() []verifyMapping {
	return []verifyMapping{
		{"driver", "local", "local"},
		{"driver_opts", map[string]interface{}{"Test": "Me"}, map[string]string{"Test": "Me"}},
		{"labels", map[string]interface{}{"Test": "Me"}, map[string]string{"Test": "Me"}},
		{"name", "Test", "Test"},
	}
}

func verifyVolumeConfig(name string, expected interface{}, config volume.VolumeCreateBody) error {
	var err error
	switch name {
	case "driver":
		err = verifyValue(expected, config.Driver)
	case "driver_opts":
		err = verifyValue(expected, config.DriverOpts)
	case "labels":
		err = verifyValue(expected, config.Labels)
	case "name":
		err = verifyValue(expected, config.Name)
	}
	if err != nil {
		return fmt.Errorf("%s: %s", name, err.Error())
	}
	return nil
}
