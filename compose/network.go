package compose

import (
	"fmt"

	"github.com/docker/docker/api/types"
	networktypes "github.com/docker/docker/api/types/network"
)

// Network contains information from compose file to create the corresponding network
type Network struct {
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

// NewNetwork creates new network config based on an element of the networks section of the compose file
func NewNetwork(config interface{}) (Network, error) {
	network := getDefaultNetwork()
	if networkConfig, isMap := config.(map[string]interface{}); isMap {
		if err := network.parseConfig(networkConfig); err != nil {
			return network, err
		}
	} else {
		return network, fmt.Errorf("network should be a map")
	}

	return network, nil
}

// GetCreateConfig returns the NetworkCreate struct required to create network with the docker API
func (n Network) GetCreateConfig() types.NetworkCreate {
	if n.external {
		return types.NetworkCreate{}
	}

	createOpts := types.NetworkCreate{
		CheckDuplicate: true,
		Labels:         n.labels,
		Driver:         n.driver,
		Options:        n.driverOpts,
		Internal:       n.internal,
		Attachable:     n.attachable,
		IPAM:           &networktypes.IPAM{},
		EnableIPv6:     n.enableIPv6,
	}

	if n.ipamDriver != "" {
		var config networktypes.IPAMConfig
		if n.ipamSubnet != "" {
			config = networktypes.IPAMConfig{
				Subnet: n.ipamSubnet,
			}
		}

		createOpts.IPAM = &networktypes.IPAM{
			Driver: n.ipamDriver,
			Config: []networktypes.IPAMConfig{config},
		}
	}

	return createOpts
}

// GetExternalName will return name of external network that has been created seperate to compose
// If network not external, it will return empty string and false.
func (n Network) GetExternalName() (string, bool) {
	if n.external {
		return n.name, n.external
	}
	return "", n.external
}

func (n *Network) parseConfig(config map[string]interface{}) error {
	//TODO: still need to process name
	if err := setValue(&n.driver, "driver", config, nil); err != nil {
		return err
	}
	if n.driver != "bridge" && n.driver != "overlay" && n.driver != "host" && n.driver != "none" {
		return fmt.Errorf("driver must be set to one of bridge, overlay, host or none")
	}

	if err := setValue(&n.driverOpts, "driver_opts", config, nil); err != nil {
		return err
	}

	if err := setValue(&n.attachable, "attachable", config, nil); err != nil {
		return err
	}

	if err := setValue(&n.enableIPv6, "enable_ipv6", config, nil); err != nil {
		return err
	}

	if err := setValue(&n.external, "external", config, nil); err != nil {
		return err
	}

	if err := setValue(&n.internal, "internal", config, nil); err != nil {
		return err
	}

	if err := setValue(&n.labels, "labels", config, nil); err != nil {
		return err
	}

	if err := setValue(&n.name, "name", config, nil); err != nil {
		return err
	}

	return n.parseIPAM(config)
}

// TODO: figure out if we need to accept multiple subnets or other config
func (n *Network) parseIPAM(config map[string]interface{}) error {
	if ipamInterface, isSet := config["ipam"]; isSet {
		ipam, isMap := ipamInterface.(map[string]interface{})
		if !isMap {
			return fmt.Errorf("ipam must be a map")
		}
		if driverInterface, isDriverSet := ipam["driver"]; isDriverSet {
			driver, isString := driverInterface.(string)
			if !isString {
				return fmt.Errorf("IPAM driver should be a string")
			}
			n.ipamDriver = driver
		}
		if configInterface, isConfigSet := ipam["config"]; isConfigSet {
			conf, isArray := configInterface.([]map[string]string)
			if !isArray {
				return fmt.Errorf("IPAM config should be an array")
			}
			if len(conf) > 0 {
				if subnet, isSubnetSet := conf[0]["subnet"]; isSubnetSet {
					n.ipamSubnet = subnet
				}
			}
		}
	}
	return nil
}

func getDefaultNetwork() Network {
	return Network{
		driver:     "bridge",
		driverOpts: map[string]string{},
		attachable: false,
		enableIPv6: false,
		ipamDriver: "",
		ipamSubnet: "",
		internal:   false,
		labels:     map[string]string{},
		external:   false,
		name:       "",
	}
}
