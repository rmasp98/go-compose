package compose_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/rmasp98/go-compose/compose"
)

func TestInvalidNetworkConfig(t *testing.T) {
	if _, err := compose.NewNetwork("invalid config"); err == nil {
		t.Errorf("Should return error but returned nothing")
	}
}

func TestEmptyConfigSetsDefaults(t *testing.T) {
	network, _ := compose.NewNetwork(map[string]interface{}{})
	if err := verifyNetwork(network, defaultNetwork()); err != nil {
		t.Errorf(err.Error())
	}
}

func TestCustomNetworkConfigurable(t *testing.T) {
	custom := customNetwork()
	network, _ := compose.NewNetwork(custom)
	if err := verifyNetwork(network, custom); err != nil {
		t.Errorf(err.Error())
	}
}

func TestExternalNetworkReturnsEmptyNetworkCreate(t *testing.T) {
	network, _ := compose.NewNetwork(map[string]interface{}{"external": true})
	if !reflect.DeepEqual(network.GetCreateConfig(), types.NetworkCreate{}) {
		t.Errorf("Did not return empty NetowrkCreate")
	}
}

func TestReturnsErrorIfNetworkDriverInvalid(t *testing.T) {
	if _, err := compose.NewNetwork(map[string]interface{}{"driver": "invalid"}); err == nil {
		t.Errorf("Should have returned an error but returned nothing")
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

func TestReturnsEmptyForNonExternalNetwork(t *testing.T) {
	network, _ := compose.NewNetwork(map[string]interface{}{"name": "Test"})
	name, external := network.GetExternalName()
	if name != "" || external {
		t.Errorf("Name was not correct: \"%s\"", name)
	}
}

// Helper data and functions
type networkSettings struct {
	driver     string
	driverOpts map[string]string
	attachable bool
	enableIPv6 bool
	ipamDriver string
	ipamSubnet string
	internal   bool
	labels     map[string]string
	external   bool
	name       string
}

func defaultNetwork() map[string]interface{} {
	return map[string]interface{}{
		"driver":      "bridge",
		"driver_opts": map[string]string{},
		"attachable":  false,
		"enable_ipv6": false,
		"ipam":        map[string]interface{}{},
		"internal":    false,
		"labels":      map[string]string{},
		"external":    false,
		"name":        "",
	}
}

func customNetwork() map[string]interface{} {
	return map[string]interface{}{
		"driver":      "host",
		"driver_opts": map[string]string{"Test": "Me"},
		"attachable":  true,
		"enable_ipv6": true,
		"ipam": map[string]interface{}{"driver": "test",
			"config": []map[string]string{{"subnet": "172.28.0.0/16"}}},
		"internal": true,
		"labels":   map[string]string{"Test": "Me"},
		"name":     "Test",
	}
}

func verifyNetwork(network compose.Network, settings map[string]interface{}) error {
	config := network.GetCreateConfig()

	// TODO: external and name is not checked yet
	if config.Driver != settings["driver"] {
		return fmt.Errorf("Driver was set to \"%s\" but should be \"%s\"", config.Driver, settings["driver"])
	}
	if !reflect.DeepEqual(config.Options, settings["driver_opts"]) {
		return fmt.Errorf("Driver options were set to %v but should be %v",
			config.Options, settings["driver_opts"])
	}
	if config.Attachable != settings["attachable"] {
		return fmt.Errorf("Attachable was set to %t but should be %t",
			config.Attachable, settings["attachable"])
	}
	if config.EnableIPv6 != settings["enable_ipv6"] {
		return fmt.Errorf("IPv6 was set to %t but should be %t", config.EnableIPv6, settings["enable_ipv6"])
	}
	ipam := settings["ipam"].(map[string]interface{})
	if driver, isSet := ipam["driver"]; isSet && config.IPAM.Driver != driver.(string) {
		return fmt.Errorf("IPAM Driver was set to \"%s\" but should be \"%s\"",
			config.IPAM.Driver, driver)
	}
	if ipamConfig, isSet := ipam["config"]; isSet {
		subnets := ipamConfig.([]map[string]string)
		if len(subnets) != len(config.IPAM.Config) {
			return fmt.Errorf("There are not the same number of subnets configured")
		}
		for i := range subnets {
			if subnets[i]["subnet"] != config.IPAM.Config[i].Subnet {
				return fmt.Errorf("Subnet \"%s\" is not the same as \"%s\"",
					config.IPAM.Config[i].Subnet, subnets[i]["subnet"])
			}
		}
	}
	if config.Internal != settings["internal"] {
		return fmt.Errorf("Internal configured to %t but should be %t", config.Internal, settings["internal"])
	}
	if !reflect.DeepEqual(config.Labels, settings["labels"]) {
		return fmt.Errorf("Labels are %v when should be %v", config.Labels, settings["labels"])
	}
	return nil
}
