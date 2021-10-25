package compose

import (
	"fmt"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

// Network contains information from compose file to create the corresponding network
type Network struct {
	data     types.NetworkCreate
	external bool
	name     string
}

// NewNetwork creates new network config based on an element of the networks section of the compose file
func NewNetwork(config interface{}) (Network, error) {
	network := Network{getDefaultNetwork(), false, ""}
	if networkConfig, isMap := config.(map[string]interface{}); isMap {
		if err := network.parseConfig(networkConfig); err != nil {
			return network, err
		}
	} else if config != nil {
		return network, fmt.Errorf("network should be a map")
	}

	return network, nil
}

// GetCreateConfig returns the NetworkCreate struct required to create network with the docker API
func (n Network) GetCreateConfig() types.NetworkCreate {
	return n.data
}

// GetExternalName will return name of external network that has been created seperate to compose
// If network not external, it will return empty string and false.
func (n Network) GetExternalName() (string, bool) {
	return n.name, n.external
}

func (n *Network) parseConfig(config map[string]interface{}) error {
	// TODO: add validation functions
	mapping := []setValueMapping{
		{"driver", &n.data.Driver, nil, validateNetworkDriver},
		{"driver_opts", &n.data.Options, convertToStringMap, nil},
		{"attachable", &n.data.Attachable, nil, nil},
		{"enable_ipv6", &n.data.EnableIPv6, nil, nil},
		{"internal", &n.data.Internal, nil, nil},
		{"labels", &n.data.Labels, convertToStringMap, nil},
		{"ipam", n.data.IPAM, convertIPAM, nil},
		//// External networks
		{"external", &n.external, nil, nil},
		{"name", &n.name, nil, nil},
	}
	if err := setValues(mapping, config); err != nil {
		return err
	}

	return nil
}

func validateNetworkDriver(input interface{}) error {
	if driver, isStr := input.(string); isStr {
		for _, validDriver := range []string{"bridge", "overlay", "host", "none"} {
			if driver == validDriver {
				return nil
			}
		}
		return fmt.Errorf("%s is not a valid driver", driver)
	}
	return fmt.Errorf("driver must be a string")
}

func convertIPAM(input interface{}) (interface{}, error) {
	if config, isMap := input.(map[string]interface{}); isMap {
		ipam := network.IPAM{}
		// TODO: add validation functions
		mapping := []setValueMapping{
			{"driver", &ipam.Driver, nil, nil},
			{"config", &ipam.Config, convertIPAMConfig, nil},
		}
		if err := setValues(mapping, config); err != nil {
			return nil, err
		}
		return ipam, nil
	}
	return nil, fmt.Errorf("ipam should be a map")
}

var ipv4Cidr = "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/(3[0-2]|[1-2][0-9]|[0-9]))$"
var ipv6Cidr = "^s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:)))(%.+)?s*"

func convertIPAMConfig(input interface{}) (interface{}, error) {
	if config, isList := input.([]interface{}); isList {
		ipamConfig := []network.IPAMConfig{}
		for _, element := range config {
			if element, isMap := element.(map[string]interface{}); isMap {
				// subnet should be only element so any more if incorrect
				if len(element) > 1 {
					return nil, fmt.Errorf("contained invalid element")
				}
				if subnet, isSet := element["subnet"]; isSet {
					subnet, err := getString(subnet)
					if err != nil {
						return nil, fmt.Errorf("subnet: %s", err.Error())
					}
					if match, _ := regexp.MatchString(ipv4Cidr+"|"+ipv6Cidr, subnet); !match {
						return nil, fmt.Errorf("subnet was not valid")
					}
					ipamConfig = append(ipamConfig, network.IPAMConfig{Subnet: subnet})
				} else {
					return nil, fmt.Errorf("did not contain subnet")
				}
			} else {
				return nil, fmt.Errorf("should be a map[string]string")
			}
		}
		return ipamConfig, nil
	}
	return nil, fmt.Errorf("config should be a list")
}

func getDefaultNetwork() types.NetworkCreate {
	return types.NetworkCreate{
		Driver:     "bridge",
		Options:    map[string]string{},
		Attachable: false,
		EnableIPv6: false,
		IPAM:       new(network.IPAM),
		Internal:   false,
		Labels:     map[string]string{},
	}
}
