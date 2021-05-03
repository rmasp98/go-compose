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

	mapping := []setValueMapping{
		{"hostname", &s.containerConfig.Hostname, nil},
		{"domainname", &s.containerConfig.Domainname, nil},
		{"user", &s.containerConfig.User, nil},
		{"expose", &s.containerConfig.ExposedPorts, convertExposed},
		{"image", &s.containerConfig.Image, nil},
		{"working_dir", &s.containerConfig.WorkingDir, nil},
		{"mac_address", &s.containerConfig.MacAddress, nil},
		{"stop_signal", &s.containerConfig.StopSignal, nil},
		{"tty", &s.containerConfig.Tty, nil},
		{"stdin_open", &s.containerConfig.OpenStdin, nil},
		{"environment", &s.containerConfig.Env, convertEnvironment},
		{"labels", &s.containerConfig.Labels, convertLabels},
		{"command", &s.containerConfig.Cmd, convertShellCommand},
		{"entrypoint", &s.containerConfig.Entrypoint, convertShellCommand},
		{"volumes", &s.containerConfig.Volumes, convertVolumes},
		{"stop_grace_period", s.containerConfig.StopTimeout, convertStopTimeout},
		{"healthcheck", s.containerConfig.Healthcheck, convertHealthCheck},
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

	mapping := []setValueMapping{
		{"volumes", &s.hostConfig.Binds, convertBinds},
		{"logging", &s.hostConfig.LogConfig, convertLogConfig},
		{"network_mode", &s.hostConfig.NetworkMode, convertNetworkMode},
		{"ports", &s.hostConfig.PortBindings, convertPortBindings},
		{"restart", &s.hostConfig.RestartPolicy, convertRestartPolicy},
		{"cap_add", &s.hostConfig.CapAdd, nil},
		{"cap_drop", &s.hostConfig.CapDrop, nil},
		{"dns", &s.hostConfig.DNS, nil},
		{"dns_search", &s.hostConfig.DNSSearch, nil},
		{"extra_hosts", &s.hostConfig.ExtraHosts, nil},
		{"ipc", &s.hostConfig.IpcMode, convertIpc},
		{"pid", &s.hostConfig.PidMode, convertPid},
		{"external_links", &s.hostConfig.Links, nil},
		{"privileged", &s.hostConfig.Privileged, nil},
		{"read_only", &s.hostConfig.ReadonlyRootfs, nil},
		{"security_opt", &s.hostConfig.SecurityOpt, nil},
		{"tmpfs", &s.hostConfig.Tmpfs, convertTmpfs},
		{"userns_mode", &s.hostConfig.UsernsMode, convertUserNs},
		{"shm_size", &s.hostConfig.ShmSize, convertShmSize},
		{"sysctls", &s.hostConfig.Sysctls, convertSysctls},
		{"init", s.hostConfig.Init, nil},
		// Resource data. May want to populate other parts of resources from config
		{"cgroup_parent", &s.hostConfig.Resources.CgroupParent, nil},
		{"devices", &s.hostConfig.Resources.Devices, convertDevices},
		{"ulimits", &s.hostConfig.Resources.Ulimits, convertUlimits},
	}
	if err := setValues(mapping, config); err != nil {
		return err
	}

	return nil
}

func (s *Service) parseNetworkConfig(input map[string]interface{}) error {
	var links []string
	if config, isSet := input["links"]; isSet {
		var isList bool
		links, isList = config.([]string)
		if !isList {
			return fmt.Errorf("Links should have been []string")
		}
	}

	if config, isSet := input["networks"]; isSet {
		networks, isMap := config.(map[string]interface{})
		if !isMap {
			return fmt.Errorf("")
		}

		endpoints := make(map[string]*networktypes.EndpointSettings)
		for name, networkConfig := range networks {
			network, isMap := networkConfig.(map[string]interface{})
			if !isMap {
				return fmt.Errorf("")
			}

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
			mapping := []setValueMapping{
				{"aliases", &endpoint.Aliases, nil},
				{"ipv4_address", &endpoint.IPAddress, nil},
				{"ipv6_address", &endpoint.GlobalIPv6Address, nil},
			}
			if err := setValues(mapping, network); err != nil {
				return err
			}

			endpoints[name] = &endpoint
		}
		s.networkConfig = networktypes.NetworkingConfig{EndpointsConfig: endpoints}
	}

	return nil
}

func (s Service) GetPlatformConfig() v1.Platform {
	return v1.Platform{
		Architecture: "",
		OS:           "",
		OSVersion:    "",
		OSFeatures:   []string{},
		Variant:      "",
	}
}

func convertLabels(input interface{}) (interface{}, error) {
	switch input := input.(type) {
	case map[string]string:
		return input, nil
	case []string:
		output := make(map[string]string)
		for _, label := range input {
			split := strings.Split(label, "=")
			if len(split) != 2 {
				return nil, fmt.Errorf("label format should be key=value format")
			}
			output[split[0]] = split[1]
		}
		return output, nil
	}
	return nil, fmt.Errorf("labels should be a map[string]string or []string")
}

func convertShellCommand(input interface{}) (interface{}, error) {
	switch input := input.(type) {
	case []string:
		return input, nil
	case string:
		return strings.Split(input, " "), nil
	}
	return nil, fmt.Errorf("command should be a string or []string")
}

func convertStopTimeout(input interface{}) (interface{}, error) {
	if timeoutString, isString := input.(string); isString {
		timeout, err := time.ParseDuration(timeoutString)
		if err != nil {
			return nil, err
		}
		timeoutSeconds := int(timeout.Seconds())
		return timeoutSeconds, nil
	}
	return nil, fmt.Errorf("Stop Timeout was not a string")
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
	ports := make(map[nat.Port]struct{})
	switch input := input.(type) {
	case []interface{}:
		for _, port := range input {
			switch port := port.(type) {
			case string:
				ports[nat.Port(port)] = struct{}{}
			case int:
				ports[nat.Port(strconv.Itoa(port))] = struct{}{}
			default:
				return nil, fmt.Errorf("Exposed ports should be a string or int")
			}
		}
		//TODO: check if this would ever happen?
	case []string:
		for _, port := range input {
			ports[nat.Port(port)] = struct{}{}
		}
	default:
		return nil, fmt.Errorf("Expose should be a []string or []interface{}")
	}
	return ports, nil
}

func convertEnvironment(input interface{}) (interface{}, error) {
	switch input := input.(type) {
	case []string:
		return input, nil
	case map[string]string:
		var varList []string
		for key, value := range input {
			varList = append(varList, key+"="+value)
		}
		return varList, nil
	}
	return nil, fmt.Errorf("Environments must be a string list or map")
}

func convertHealthCheck(input interface{}) (interface{}, error) {
	if hc, isMap := input.(map[string]interface{}); isMap {
		hcOut := container.HealthConfig{}
		mapping := []setValueMapping{
			{"test", &hcOut.Test, nil},
			{"interval", &hcOut.Interval, convertDuration},
			{"timeout", &hcOut.Timeout, convertDuration},
			{"start_period", &hcOut.StartPeriod, convertDuration},
			{"retries", &hcOut.Retries, nil},
		}
		if err := setValues(mapping, hc); err != nil {
			return nil, err
		}
		return hcOut, nil
	}
	return nil, nil
}

func convertDevices(input interface{}) (interface{}, error) {
	if config, isList := input.([]string); isList {
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
	return nil, fmt.Errorf("devices should be a []string")
}

func convertPortBindings(input interface{}) (interface{}, error) {
	if config, isList := input.([]string); isList {
		_, bindings, err := nat.ParsePortSpecs(config)
		if err != nil {
			return nil, err
		}
		return bindings, nil
	}
	return nil, fmt.Errorf("ports must be a []string")
}

func convertSysctls(input interface{}) (interface{}, error) {
	if config, isList := input.([]string); isList {
		sysctls := make(map[string]string)
		for _, option := range config {
			values := strings.Split(option, "=")
			if len(values) == 2 {
				sysctls[values[0]] = values[1]
			}
		}
		return sysctls, nil
	}
	return nil, fmt.Errorf("sysctls should have been []string")
}

func convertShmSize(input interface{}) (interface{}, error) {
	if size, isStr := input.(string); isStr {
		return units.FromHumanSize(size)
	}
	return nil, fmt.Errorf("shm_size should have been a string")
}

func convertUserNs(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.UsernsMode(mode), nil
	}
	return nil, fmt.Errorf("userns_mode should be a string")
}

func convertTmpfs(input interface{}) (interface{}, error) {
	if config, isList := input.([]string); isList {
		tmpfs := make(map[string]string)
		for _, mount := range config {
			options := strings.Split(mount, ":")
			if len(options) == 2 {
				tmpfs[options[0]] = options[1]
			}
		}
		return tmpfs, nil
	}
	return nil, fmt.Errorf("tmpfs should have been a []string")
}

func convertIpc(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.IpcMode(mode), nil
	}
	return nil, fmt.Errorf("ipc should be a string")
}

func convertPid(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.PidMode(mode), nil
	}
	return nil, fmt.Errorf("pid should be a string")
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
	return nil, fmt.Errorf("restart should be a string")
}

func convertNetworkMode(input interface{}) (interface{}, error) {
	if mode, isStr := input.(string); isStr {
		return container.NetworkMode(mode), nil
	}
	return nil, fmt.Errorf("network_mode should have been string")
}

func convertLogConfig(input interface{}) (interface{}, error) {
	if logging, isMap := input.(map[string]interface{}); isMap {
		logConfig := container.LogConfig{}
		mapping := []setValueMapping{
			{"driver", &logConfig.Type, nil},
			{"options", &logConfig.Config, nil},
		}
		if err := setValues(mapping, logging); err != nil {
			return nil, err
		}
		return logConfig, nil
	}
	return nil, fmt.Errorf("logging should be a map")
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
			case map[string]int:
				// TODO: check if setting one or the other in mandatory
				limit := units.Ulimit{Name: name}
				if soft, isSet := limits["soft"]; isSet {
					limit.Soft = int64(soft)
				}
				if hard, isSet := limits["hard"]; isSet {
					limit.Hard = int64(hard)
				}
				ulimits = append(ulimits, &limit)
			default:
				return nil, fmt.Errorf("%s ulimit should be a string or map but was %t", name, limits)
			}
		}
		return ulimits, nil
	}
	return nil, fmt.Errorf("ulimits should be a map")
}
