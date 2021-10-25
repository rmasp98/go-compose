package compose_test

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/rmasp98/go-compose/compose"
)

func TestInvalidNetworkConfig(t *testing.T) {
	if _, err := compose.NewNetwork("invalid config"); err == nil {
		t.Errorf("Should return error but returned nothing")
	}
}

func TestReturnsDefaultForEmptyNetwork(t *testing.T) {
	network, _ := compose.NewNetwork(nil)
	createConfig := network.GetCreateConfig()
	if createConfig.Driver != "bridge" {
		t.Errorf("Should have returned bridge but got \"%s\"", createConfig.Driver)
	}
}

func TestCustomNetworkConfigurable(t *testing.T) {
	for _, mapping := range getNetworkMapping() {
		network, err := compose.NewNetwork(map[string]interface{}{mapping.name: mapping.source})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if err := verifyNetworkConfig(mapping.name, mapping.expected, network.GetCreateConfig()); err != nil {
			t.Errorf(err.Error())
		}
	}
}

func TestReturnsErrorIfNotValidTypeInNetwork(t *testing.T) {
	options := []string{
		"driver", "driver_opts", "attachable", "enable_ipv6", "internal", "external", "labels", "ipam",
	}
	for _, option := range options {
		if _, err := compose.NewNetwork(map[string]interface{}{option: 0}); err == nil {
			t.Errorf("Should have returned an error for \"%s\" but returned nothing", option)
		}
	}
}

func TestRetunsErrorIfIPAMDriverInvalid(t *testing.T) {
	if _, err := compose.NewNetwork(map[string]interface{}{
		"ipam": map[string]interface{}{"driver": 0}}); err == nil {
		t.Errorf("Should have returned an error for IPAM driver but returned nothing")
	}
}

func TestRetunsErrorIfIPAMConfigInvalid(t *testing.T) {
	if _, err := compose.NewNetwork(map[string]interface{}{
		"ipam": map[string]interface{}{"config": 0}}); err == nil {
		t.Errorf("Should have returned an error for IPAM config but returned nothing")
	}
}

func TestReturnsNameForExternalNetwork(t *testing.T) {
	network, _ := compose.NewNetwork(map[string]interface{}{"external": true, "name": "Test"})
	name, external := network.GetExternalName()
	if name != "Test" || !external {
		t.Errorf("Name was not correct: \"%s\"", name)
	}
}

func TestReturnsErrorForInvalidData(t *testing.T) {
	for _, mapping := range getInvalidNetworkConfig() {
		if _, err := compose.NewNetwork(map[string]interface{}{mapping.name: mapping.data}); err == nil {
			t.Errorf("%s should have returned error but did not", mapping.name)
			return
		}
	}
}

// Test data and helper functions
var (
	sourceIPAM   = map[string]interface{}{"driver": "test", "config": []interface{}{map[string]interface{}{"subnet": "172.28.0.0/16"}}}
	expectedIPAM = network.IPAM{Driver: "test", Config: []network.IPAMConfig{{Subnet: "172.28.0.0/16"}}}
)

func getNetworkMapping() []verifyMapping {
	return []verifyMapping{
		{"driver", "host", "host"},
		{"driver_opts", map[string]interface{}{"Test": "Me"}, map[string]string{"Test": "Me"}},
		{"attachable", true, true},
		{"enable_ipv6", true, true},
		{"ipam", sourceIPAM, expectedIPAM},
		{"internal", true, true},
		{"labels", map[string]interface{}{"Test": "Me"}, map[string]string{"Test": "Me"}},
		{"external", true, true},
		{"name", "Test", "Test"},
	}
}

func verifyNetworkConfig(name string, expected interface{}, config types.NetworkCreate) error {
	var err error
	switch name {
	case "driver":
		err = verifyValue(expected, config.Driver)
	case "driver_opts":
		err = verifyValue(expected, config.Options)
	case "attachable":
		err = verifyValue(expected, config.Attachable)
	case "enable_ipv6":
		err = verifyValue(expected, config.EnableIPv6)
	case "ipam":
		err = verifyValue(expected, *config.IPAM)
	case "internal":
		err = verifyValue(expected, config.Internal)
	case "labels":
		err = verifyValue(expected, config.Labels)
	}
	if err != nil {
		return fmt.Errorf("%s: %s", name, err.Error())
	}
	return nil
}

type invalidMapping struct {
	name string
	data interface{}
}

func getInvalidNetworkConfig() []invalidMapping {
	return []invalidMapping{
		{"driver", "invalid"},
		{"ipam", map[string]interface{}{"config": []interface{}{map[string]interface{}{"invalid": "test"}}}},
		{"ipam", map[string]interface{}{"config": []interface{}{map[string]interface{}{"subnet": "invalid"}}}},
	}
}
