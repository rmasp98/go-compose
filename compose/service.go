package compose

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	networktypes "github.com/docker/docker/api/types/network"

	"github.com/docker/go-connections/nat"
	units "github.com/docker/go-units"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

//TODO: not handled container_name, depends_on, deploy, env_file, build

type Service struct {
	containerConfig container.Config
	hostConfig      container.HostConfig
	networkConfig   networktypes.NetworkingConfig

	//build         BuildConfig
	//configs       []string // This can also be in the long config format below
	//containerName string   //not used...
	//credentialSpec map[string]string //Windows specific
	//dependsOn []string //this is probably used by me
	// TODO: decide if store deploy info as it is swarm specific
	//envFile       []string    // This should probably be used by me to populate environment
	//isolation       string // windows specific
	//profiles        []string                  //Not used...
	//secrets         []Secret                  //Not used...
	//mounts          []mount.Mount
}

func NewService(yamlData interface{}) (Service, error) {
	service := Service{}
	config, isMap := yamlData.(map[string]interface{})
	if !isMap {
		return service, fmt.Errorf("yamlData was not a map[string]interface{}")
	}

	if err := service.parseContainerConfig(config); err != nil {
		return service, err
	}

	if err := service.parseHostConfig(config); err != nil {
		return service, err
	}

	if err := service.parseNetworkConfig(config); err != nil {
		return service, err
	}

	return service, nil
}

func (s Service) GetContainerConfig() container.Config {
	return s.containerConfig
}

func (s Service) GetHostConfig() container.HostConfig {
	return s.hostConfig
}

func (s Service) GetNetworkConfig() networktypes.NetworkingConfig {
	return s.networkConfig
}

func (s *Service) parseContainerConfig(config map[string]interface{}) error {
	// Set default values (kept for visibility)
	s.containerConfig = container.Config{
		AttachStdin:     false,
		AttachStdout:    false,
		AttachStderr:    false,
		StdinOnce:       false,
		Healthcheck:     new(container.HealthConfig),
		NetworkDisabled: false,
		OnBuild:         []string{}, // Don't think this is needed
		Shell:           []string{}, //TODO figure out if this is needed
		StopTimeout:     new(int),
	}

	// TODO: add validation functions
	mapping := []setValueMapping{
		{"hostname", &s.containerConfig.Hostname, nil, nil},
		{"domainname", &s.containerConfig.Domainname, nil, nil},
		{"user", &s.containerConfig.User, nil, nil},
		{"expose", &s.containerConfig.ExposedPorts, convertExposed, nil},
		{"image", &s.containerConfig.Image, nil, nil},
		{"working_dir", &s.containerConfig.WorkingDir, nil, nil},
		{"mac_address", &s.containerConfig.MacAddress, nil, nil},
		{"stop_signal", &s.containerConfig.StopSignal, nil, nil},
		{"tty", &s.containerConfig.Tty, nil, nil},
		{"stdin_open", &s.containerConfig.OpenStdin, nil, nil},
		{"environment", &s.containerConfig.Env, convertToStringList, nil},
		{"labels", &s.containerConfig.Labels, convertToStringMap, nil},
		{"command", &s.containerConfig.Cmd, convertToStringList, nil},
		{"entrypoint", &s.containerConfig.Entrypoint, convertToStringList, nil},
		{"volumes", &s.containerConfig.Volumes, convertVolumes, nil},
		{"stop_grace_period", s.containerConfig.StopTimeout, convertStopTimeout, nil},
		{"healthcheck", s.containerConfig.Healthcheck, convertHealthCheck, nil},
	}
	if err := setValues(mapping, config); err != nil {
		return err
	}

	return nil
}

func (s *Service) parseHostConfig(config map[string]interface{}) error {
	// Set default values and refernces (kept for visibility)
	s.hostConfig = container.HostConfig{
		ContainerIDFile: "",
		AutoRemove:      false,
		VolumeDriver:    "",
		VolumesFrom:     []string{}, // If sharing volumes
		CgroupnsMode:    "",
		DNSOptions:      []string{}, //Not supported in compose v3
		GroupAdd:        []string{}, //additional groups to container process to run as
		Cgroup:          "",         // Doesn't seem to be a flag in docker...
		OomScoreAdj:     0,          //TODO ensure that this is the default value
		PublishAllPorts: false,
		StorageOpt:      map[string]string{}, // Not needed
		UTSMode:         "",                  // can be set to host but leave default
		Runtime:         "",                  //TODO figure what this is

		// Windsow options
		//ConsoleSize: [2]uint
		//Isolation: container.Isolation{}

		Mounts:        []mount.Mount{}, //TODO: should be parsed from file
		MaskedPaths:   []string{},      // default not really needed
		ReadonlyPaths: []string{},      // default not really needed
		Init:          new(bool),
	}

	// TODO: add validation functions
	mapping := []setValueMapping{
		{"volumes", &s.hostConfig.Binds, convertBinds, nil},
		{"logging", &s.hostConfig.LogConfig, convertLogConfig, nil},
		{"network_mode", &s.hostConfig.NetworkMode, convertNetworkMode, nil},
		{"ports", &s.hostConfig.PortBindings, convertPortBindings, nil},
		{"restart", &s.hostConfig.RestartPolicy, convertRestartPolicy, nil},
		{"cap_add", &s.hostConfig.CapAdd, convertToStringList, nil},
		{"cap_drop", &s.hostConfig.CapDrop, convertToStringList, nil},
		{"dns", &s.hostConfig.DNS, convertToStringList, nil},
		{"dns_search", &s.hostConfig.DNSSearch, convertToStringList, nil},
		{"extra_hosts", &s.hostConfig.ExtraHosts, convertToStringList, nil},
		{"ipc", &s.hostConfig.IpcMode, convertIpc, nil},
		{"pid", &s.hostConfig.PidMode, convertPid, nil},
		{"external_links", &s.hostConfig.Links, convertToStringList, nil},
		{"privileged", &s.hostConfig.Privileged, nil, nil},
		{"read_only", &s.hostConfig.ReadonlyRootfs, nil, nil},
		{"security_opt", &s.hostConfig.SecurityOpt, convertToStringList, nil},
		{"tmpfs", &s.hostConfig.Tmpfs, convertTmpfs, nil},
		{"userns_mode", &s.hostConfig.UsernsMode, convertUserNs, nil},
		{"shm_size", &s.hostConfig.ShmSize, convertShmSize, nil},
		{"sysctls", &s.hostConfig.Sysctls, convertToStringMap, nil},
		{"init", s.hostConfig.Init, nil, nil},
		// Resource data. May want to populate other parts of resources from config
		{"cgroup_parent", &s.hostConfig.Resources.CgroupParent, nil, nil},
		{"devices", &s.hostConfig.Resources.Devices, convertDevices, nil},
		{"ulimits", &s.hostConfig.Resources.Ulimits, convertUlimits, nil},
	}
	if err := setValues(mapping, config); err != nil {
		return err
	}

	return nil
}

func (s *Service) parseNetworkConfig(input map[string]interface{}) error {
	links, err := parseStringList(input["links"])
	if err != nil {
		return nil
	}

	if config, isSet := input["networks"]; isSet {
		networks, isMap := config.(map[string]interface{})
		if !isMap {
			return fmt.Errorf("networks should be a map")
		}

		endpoints := make(map[string]*networktypes.EndpointSettings)
		for name, networkConfig := range networks {
			endpoint := networktypes.EndpointSettings{
				Links:               links,
				MacAddress:          "", //TODO: Option is a available in compose file
				IPAMConfig:          &networktypes.EndpointIPAMConfig{},
				NetworkID:           "",
				EndpointID:          "",
				Gateway:             "",
				IPPrefixLen:         0,
				IPv6Gateway:         "",
				GlobalIPv6PrefixLen: 0,
				DriverOpts:          map[string]string{},
			}
			switch network := networkConfig.(type) {
			case map[string]interface{}:
				// TODO: add validation functions
				mapping := []setValueMapping{
					{"aliases", &endpoint.Aliases, convertToStringList, nil},
					{"ipv4_address", &endpoint.IPAddress, nil, nil},
					{"ipv6_address", &endpoint.GlobalIPv6Address, nil, nil},
				}
				if err := setValues(mapping, network); err != nil {
					return err
				}
			case nil:
			default:
				return fmt.Errorf("network: %s is not a valid type", name)
			}
			endpoints[name] = &endpoint
		}
		s.networkConfig = networktypes.NetworkingConfig{EndpointsConfig: endpoints}
	}

	return nil
}

// TODO: retrieve this information
func (s Service) GetPlatformConfig() v1.Platform {
	return v1.Platform{
		Architecture: "",
		OS:           "",
		OSVersion:    "",
		OSFeatures:   []string{},
		Variant:      "",
	}
}

func convertStopTimeout(input interface{}) (interface{}, error) {
	duration, err := convertDuration(input)
	if err != nil {
		return nil, err
	}
	timeout := duration.(time.Duration)
	return int(timeout.Seconds()), nil

}

func convertVolumes(input interface{}) (interface{}, error) {
	config, err := parseVolumes(input)
	if err != nil {
		return nil, err
	}
	volumes := make(map[string]struct{})
	for _, volume := range config {
		if volume.Type == "volume" {
			volumes[getVolumeString(volume)] = struct{}{}
		}
	}
	return volumes, nil
}

func convertExposed(input interface{}) (interface{}, error) {
	stringList, err := parseStringList(input)
	if err != nil {
		return nil, err
	}

	ports := make(map[nat.Port]struct{})
	for _, port := range stringList {
		ports[nat.Port(port)] = struct{}{}
	}
	return ports, nil
}

func convertHealthCheck(input interface{}) (interface{}, error) {
	if hc, isMap := input.(map[string]interface{}); isMap {
		hcOut := container.HealthConfig{}
		// TODO: add validation functions
		mapping := []setValueMapping{
			{"test", &hcOut.Test, convertToStringList, nil},
			{"interval", &hcOut.Interval, convertDuration, nil},
			{"timeout", &hcOut.Timeout, convertDuration, nil},
			{"start_period", &hcOut.StartPeriod, convertDuration, nil},
			{"retries", &hcOut.Retries, nil, nil},
		}
		if err := setValues(mapping, hc); err != nil {
			return nil, err
		}
		return hcOut, nil
	}
	return nil, nil
}

func convertDevices(input interface{}) (interface{}, error) {
	config, err := parseStringList(input)
	if err != nil {
		return nil, err
	}
	devices := []container.DeviceMapping{}
	for _, device := range config {
		options := strings.Split(device, ":")
		if len(options) < 2 {
			return nil, fmt.Errorf("devices contains invalid device mapping")
		}
		deviceMapping := container.DeviceMapping{PathOnHost: options[0], PathInContainer: options[1], CgroupPermissions: "rwm"}
		if len(options) == 3 {
			deviceMapping.CgroupPermissions = options[2]
		}
		devices = append(devices, deviceMapping)
	}
	return devices, nil
}

func convertPortBindings(input interface{}) (interface{}, error) {
	config, err := parseStringList(input)
	if err != nil {
		return nil, err
	}

	_, bindings, err := nat.ParsePortSpecs(config)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func convertShmSize(input interface{}) (interface{}, error) {
	if size, isStr := input.(string); isStr {
		return units.FromHumanSize(size)
	}
	return nil, fmt.Errorf("should be a string")
}

func convertUserNs(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.UsernsMode(mode), nil
	}
	return nil, fmt.Errorf("should be a string")
}

func convertTmpfs(input interface{}) (interface{}, error) {
	config, err := parseStringList(input)
	if err != nil {
		return nil, err
	}

	tmpfs := make(map[string]string)
	for _, mount := range config {
		options := strings.Split(mount, ":")
		if len(options) == 2 {
			tmpfs[options[0]] = options[1]
		}
	}
	return tmpfs, nil
}

func convertIpc(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.IpcMode(mode), nil
	}
	return nil, fmt.Errorf("should be a string")
}

func convertPid(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.PidMode(mode), nil
	}
	return nil, fmt.Errorf("should be a string")
}

func convertRestartPolicy(input interface{}) (interface{}, error) {
	if restart, isStr := input.(string); isStr {
		options := strings.Split(restart, ":")
		policy := container.RestartPolicy{Name: options[0]}
		if options[0] == "on-failure" && len(options) == 2 {
			retryCount, err := strconv.ParseInt(options[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("%s should be a number", options[1])
			}
			policy.MaximumRetryCount = int(retryCount)
		}
		return policy, nil
	}
	return nil, fmt.Errorf("should be a string")
}

func convertNetworkMode(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.NetworkMode(mode), nil
	}
	return nil, fmt.Errorf("should have been string")
}

func convertLogConfig(input interface{}) (interface{}, error) {
	if logging, isMap := input.(map[string]interface{}); isMap {
		logConfig := container.LogConfig{}
		// TODO: add validation functions
		mapping := []setValueMapping{
			{"driver", &logConfig.Type, nil, nil},
			{"options", &logConfig.Config, convertToStringMap, nil},
		}
		if err := setValues(mapping, logging); err != nil {
			return nil, err
		}
		return logConfig, nil
	}
	return nil, fmt.Errorf("should be a map")
}

func convertBinds(input interface{}) (interface{}, error) {
	config, err := parseVolumes(input)
	if err != nil {
		return nil, err
	}
	var binds []string
	for _, bind := range config {
		if bind.Type == "bind" {
			binds = append(binds, getVolumeString(bind))
		}
	}
	return binds, nil
}

func convertUlimits(input interface{}) (interface{}, error) {
	if config, isMap := input.(map[string]interface{}); isMap {
		ulimits := []*units.Ulimit{}
		for name, limits := range config {
			switch limits := limits.(type) {
			case int:
				ulimits = append(ulimits, &units.Ulimit{Name: name, Hard: int64(limits), Soft: int64(limits)})
			case map[string]interface{}:
				// TODO: check if setting one or the other in mandatory
				limit := units.Ulimit{Name: name}
				if soft, isSet := limits["soft"]; isSet {
					soft, isInt := soft.(int)
					if !isInt {
						return nil, fmt.Errorf("ulimit - soft is not int")
					}
					limit.Soft = int64(soft)
				}
				if hard, isSet := limits["hard"]; isSet {
					hard, isInt := hard.(int)
					if !isInt {
						return nil, fmt.Errorf("ulimit - hard is not int")
					}
					limit.Hard = int64(hard)
				}
				ulimits = append(ulimits, &limit)
			default:
				return nil, fmt.Errorf("%s ulimit should be a string or map but was %t", name, limits)
			}
		}
		return ulimits, nil
	}
	return nil, fmt.Errorf("should be a map")
}
