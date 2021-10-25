package compose_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/rmasp98/go-compose/compose"
)

// TODO: Sort expose order (what am I talking about?)
// TODO: Read enironment variables from file (do we do this here?)

func TestCanParseContainerConfig(t *testing.T) {
	for _, mapping := range getContainerMapping() {
		service, err := compose.NewService(map[string]interface{}{mapping.name: mapping.source})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if err := verifyContainerConfig(mapping.name, mapping.expected, service.GetContainerConfig()); err != nil {
			t.Errorf(err.Error())
		}
	}
}

func TestReturnsErrorForInvalidTypeInContainerConfig(t *testing.T) {
	for _, mapping := range getContainerMapping() {
		if _, err := compose.NewService(map[string]interface{}{mapping.name: 0}); err == nil {
			t.Errorf("%s should have returned an error but did not", mapping.name)
		}
	}
}

func TestReturnsErrorIfContainerConfigDataNotValid(t *testing.T) {
	for _, mapping := range getInvalidContainerMapping() {
		if _, err := compose.NewService(map[string]interface{}{mapping.name: 0}); err == nil {
			t.Errorf("%s should have returned an error but did not", mapping.name)
		}
	}
}

func TestCanParseHostConfig(t *testing.T) {
	for _, mapping := range getHostMapping() {
		service, err := compose.NewService(map[string]interface{}{mapping.name: mapping.source})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if err := verifyHostConfig(mapping.name, mapping.expected, service.GetHostConfig()); err != nil {
			t.Errorf(err.Error())
		}
	}
}

func TestReturnsErrorForInvalidTypeInHostConfig(t *testing.T) {
	for _, mapping := range getHostMapping() {
		if _, err := compose.NewService(map[string]interface{}{mapping.name: 0}); err == nil {
			t.Errorf("%s should have returned an error but did not", mapping.name)
		}
	}
}

func TestCanParseNetworkConfig(t *testing.T) {
	for _, mapping := range getContainerNetworkMapping() {
		service, err := compose.NewService(map[string]interface{}{
			"links": sourceLinks,
			"networks": map[string]interface{}{
				"test": map[string]interface{}{
					mapping.name: mapping.source,
				},
			},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if err := verifyContainerNetworkConfig(mapping.name, mapping.expected, service.GetNetworkConfig()); err != nil {
			t.Errorf(err.Error())
		}
	}
}

// TODO: figure out if we still need this test
//func TestReturnsErrorForInvalidTypeInNetworkConfig(t *testing.T) {
//	for name := range getNetworkMapping() {
//		if _, err := compose.NewService(map[string]interface{}{name: 0}); err == nil {
//			t.Errorf("%s should have returned an error but did not", name)
//		}
//	}
//}

// Helper functions and test data

type verifyMapping struct {
	name     string
	source   interface{}
	expected interface{}
}

var (
	sourceHealthCheck   = map[string]interface{}{"test": []interface{}{"CMD", "curl"}, "interval": "1m30s", "timeout": "10s", "retries": 3, "start_period": "40s"}
	expectedHealthCheck = container.HealthConfig{Test: []string{"CMD", "curl"}, Interval: time.Duration(90 * 1e9), Timeout: time.Duration(10 * 1e9), Retries: 3, StartPeriod: time.Duration(40 * 1e9)}
	expectedVolumes     = map[string]struct{}{"/directory:rw": {}, "volume:/target:ro": {}}
	sourceAltVolumes    = []map[string]interface{}{{"type": "volume", "source": "mydata", "target": "/data", "volume": map[string]bool{"nocopy": true}}}
	expectedAltVolumes  = map[string]struct{}{"mydata:/data:rw,nocopy": {}}
)

func getContainerMapping() []verifyMapping {
	return []verifyMapping{
		{"command", []interface{}{"/bin/bash", "script"}, strslice.StrSlice{"/bin/bash", "script"}},
		{"command", "/bin/bash script", strslice.StrSlice{"/bin/bash", "script"}},
		{"domainname", "Some Domain", "Some Domain"},
		{"entrypoint", []interface{}{"/bin/bash", "startup"}, strslice.StrSlice{"/bin/bash", "startup"}},
		{"entrypoint", "/bin/bash startup", strslice.StrSlice{"/bin/bash", "startup"}},
		{"environment", []interface{}{"Test=var"}, []string{"Test=var"}},
		{"environment", map[string]interface{}{"test1": "var", "test2": nil, "test3": 1, "test4": true}, []string{"test1=var", "test2=", "test3=1", "test4=true"}},
		{"expose", []interface{}{3000, "2000"}, nat.PortSet{"3000": {}, "2000": {}}},
		{"healthcheck", sourceHealthCheck, &expectedHealthCheck},
		{"hostname", "some host", "some host"},
		{"image", "someimage", "someimage"},
		{"labels", map[string]interface{}{"Some": "Label"}, map[string]string{"Some": "Label"}},
		{"labels", []interface{}{"Some=Label"}, map[string]string{"Some": "Label"}},
		{"mac_address", "SomeMac", "SomeMac"},
		{"stdin_open", true, true},
		{"stop_grace_period", "1m30s", func(x int) *int { return &x }(90)}, //bodge for inline *int
		{"stop_signal", "somesignal", "somesignal"},
		{"tty", true, true},
		{"user", "Some User", "Some User"},
		{"volumes", []interface{}{"/directory", "volume:/target:ro", "/abs/dir:/target", "./reldir:/target", "~/home:/target"}, expectedVolumes},
		{"volumes", sourceAltVolumes, expectedAltVolumes},
		{"working_dir", "SomeDirectory", "SomeDirectory"},
	}
}

func verifyContainerConfig(name string, expected interface{}, config container.Config) error {
	var err error
	switch name {
	case "command":
		err = verifyValue(expected, config.Cmd)
	case "domainname":
		err = verifyValue(expected, config.Domainname)
	case "entrypoint":
		err = verifyValue(expected, config.Entrypoint)
	case "environment":
		sort.Strings(expected.([]string))
		sort.Strings(config.Env)
		err = verifyValue(expected, config.Env)
	case "expose":
		err = verifyValue(expected, config.ExposedPorts)
	case "healthcheck":
		err = verifyValue(expected, config.Healthcheck)
	case "hostname":
		err = verifyValue(expected, config.Hostname)
	case "image":
		err = verifyValue(expected, config.Image)
	case "labels":
		err = verifyValue(expected, config.Labels)
	case "mac_address":
		err = verifyValue(expected, config.MacAddress)
	case "stdin_open":
		err = verifyValue(expected, config.OpenStdin)
	case "stop_grace_period":
		err = verifyValue(expected, config.StopTimeout)
	case "stop_signal":
		err = verifyValue(expected, config.StopSignal)
	case "tty":
		err = verifyValue(expected, config.Tty)
	case "user":
		err = verifyValue(expected, config.User)
	case "volumes":
		err = verifyValue(expected, config.Volumes)
	case "working_dir":
		err = verifyValue(expected, config.WorkingDir)
	}
	if err != nil {
		return fmt.Errorf("%s: %s", name, err.Error())
	}
	return nil
}

// TODO: carry on from here
func getInvalidContainerMapping() []invalidMapping {
	return []invalidMapping{
		{"", ""},
	}
}

var (
	sourceBinds   = []interface{}{"/directory", "volume:/target", "/abs/dir:/target", "./reldir:/target", "~/home:/target"}
	expectedBinds = []string{"/abs/dir:/target:rw", "./reldir:/target:rw", "~/home:/target:rw"}

	sourceLogConfig   = map[string]interface{}{"driver": "json-file", "options": map[string]interface{}{"max-size": "12m"}}
	expectedLogconfig = container.LogConfig{Type: "json-file", Config: map[string]string{"max-size": "12m"}}

	sourcePortBindings = []interface{}{"3000", "4000-4001", "5000:5000", "6000-6001:6100-6101", "127.0.0.1:7001:7001",
		"127.0.0.1:8000-8001:8000-8001", "127.0.0.1::9000", "10000:10000/udp"}
	expectedPortBindings = nat.PortMap{
		"3000/tcp":  {{HostIP: "", HostPort: ""}},
		"4000/tcp":  {{HostIP: "", HostPort: ""}},
		"4001/tcp":  {{HostIP: "", HostPort: ""}},
		"5000/tcp":  {{HostIP: "", HostPort: "5000"}},
		"6100/tcp":  {{HostIP: "", HostPort: "6000"}},
		"6101/tcp":  {{HostIP: "", HostPort: "6001"}},
		"7001/tcp":  {{HostIP: "127.0.0.1", HostPort: "7001"}},
		"8000/tcp":  {{HostIP: "127.0.0.1", HostPort: "8000"}},
		"8001/tcp":  {{HostIP: "127.0.0.1", HostPort: "8001"}},
		"9000/tcp":  {{HostIP: "127.0.0.1", HostPort: ""}},
		"10000/udp": {{HostIP: "", HostPort: "10000"}},
	}

	expectedDevices = []container.DeviceMapping{{PathOnHost: "/dev/ttyUSB0", PathInContainer: "/dev/ttyUSB0", CgroupPermissions: "rwm"}}

	sourceUlimits   = map[string]interface{}{"nproc": 65535, "nofile": map[string]interface{}{"soft": 20000, "hard": 40000}}
	expectedUlimits = []*units.Ulimit{{Name: "nproc", Hard: 65535, Soft: 65535}, {Name: "nofile", Hard: 40000, Soft: 20000}}
)

// TODO: dns and dns_search can also be string
// TODO: sysctls can also be map
// TODO: ports can also be a map
func getHostMapping() []verifyMapping {
	return []verifyMapping{
		{"volumes", sourceBinds, expectedBinds},
		{"logging", sourceLogConfig, expectedLogconfig},
		{"network_mode", "host", container.NetworkMode("host")},
		{"ports", sourcePortBindings, expectedPortBindings},
		{"restart", "on-failure:5", container.RestartPolicy{Name: "on-failure", MaximumRetryCount: 5}},
		{"cap_add", []interface{}{"ALL"}, strslice.StrSlice{"ALL"}},
		{"cap_drop", []interface{}{"NET_ADMIN"}, strslice.StrSlice{"NET_ADMIN"}},
		{"dns", []interface{}{"8.8.8.8"}, []string{"8.8.8.8"}},
		{"dns_search", []interface{}{"example.com"}, []string{"example.com"}},
		{"extra_hosts", []interface{}{"somehost:162.242.195.82", "otherhost:50.31.209.229"}, []string{"somehost:162.242.195.82", "otherhost:50.31.209.229"}},
		{"ipc", "host", container.IpcMode("host")},
		{"pid", "host", container.PidMode("host")},
		{"external_links", []interface{}{"db", "test:external"}, []string{"db", "test:external"}},
		{"privileged", true, true},
		{"read_only", true, true},
		{"security_opt", []interface{}{"label:user:USER", "label:role:ROLE"}, []string{"label:user:USER", "label:role:ROLE"}},
		{"tmpfs", []interface{}{"/tmp:rw,size=787448k,mode=1777"}, map[string]string{"/tmp": "rw,size=787448k,mode=1777"}},
		{"userns_mode", "host", container.UsernsMode("host")},
		{"shm_size", "64M", int64(64000000)},
		{"sysctls", []interface{}{"net.core.somaxconn=1024", "net.ipv4.tcp_syncookies=0"}, map[string]string{"net.core.somaxconn": "1024", "net.ipv4.tcp_syncookies": "0"}},
		{"init", true, func(x bool) *bool { return &x }(true)}, //bodge for inline *bool
		// Resources
		{"cgroup_parent", "m-executor-abcd", "m-executor-abcd"},
		{"devices", []interface{}{"/dev/ttyUSB0:/dev/ttyUSB0"}, expectedDevices},
		{"ulimits", sourceUlimits, expectedUlimits},
	}
}

func verifyHostConfig(name string, expected interface{}, config container.HostConfig) error {
	var err error
	switch name {
	case "volumes":
		err = verifyValue(expected, config.Binds)
	case "logging":
		err = verifyValue(expected, config.LogConfig)
	case "network_mode":
		err = verifyValue(expected, config.NetworkMode)
	case "ports":
		err = verifyValue(expected, config.PortBindings)
	case "restart":
		err = verifyValue(expected, config.RestartPolicy)
	case "cap_add":
		err = verifyValue(expected, config.CapAdd)
	case "cap_drop":
		err = verifyValue(expected, config.CapDrop)
	case "dns":
		err = verifyValue(expected, config.DNS)
	case "dns_search":
		err = verifyValue(expected, config.DNSSearch)
	case "extra_hosts":
		err = verifyValue(expected, config.ExtraHosts)
	case "ipc":
		err = verifyValue(expected, config.IpcMode)
	case "pid":
		err = verifyValue(expected, config.PidMode)
	case "external_links":
		err = verifyValue(expected, config.Links)
	case "privileged":
		err = verifyValue(expected, config.Privileged)
	case "read_only":
		err = verifyValue(expected, config.ReadonlyRootfs)
	case "security_opt":
		err = verifyValue(expected, config.SecurityOpt)
	case "tmpfs":
		err = verifyValue(expected, config.Tmpfs)
	case "userns_mode":
		err = verifyValue(expected, config.UsernsMode)
	case "shm_size":
		err = verifyValue(expected, config.ShmSize)
	case "sysctls":
		err = verifyValue(expected, config.Sysctls)
	case "init":
		err = verifyValue(expected, config.Init)
	case "cgroup_parent":
		err = verifyValue(expected, config.Resources.CgroupParent)
	case "devices":
		err = verifyValue(expected, config.Resources.Devices)
	case "ulimits":
		expected := expected.([]*units.Ulimit)
		sort.Slice(expected, func(i, j int) bool { return expected[i].Name < expected[j].Name })
		actual := config.Resources.Ulimits
		sort.Slice(actual, func(i, j int) bool { return actual[i].Name < actual[j].Name })
		err = verifyValue(expected, actual)
	}
	if err != nil {
		return fmt.Errorf("%s: %s", name, err.Error())
	}
	return nil
}

var (
	sourceLinks   = []interface{}{"db", "test"}
	expectedLinks = []string{"db", "test"}
)

func getContainerNetworkMapping() []verifyMapping {
	return []verifyMapping{
		{"aliases", []interface{}{"db1", "db2"}, []string{"db1", "db2"}},
		{"ipv4_address", "172.16.238.10", "172.16.238.10"},
		{"ipv6_address", "2001:3984:3989::10", "2001:3984:3989::10"},
	}
}

func verifyContainerNetworkConfig(name string, expected interface{}, config networktypes.NetworkingConfig) error {
	network, isSet := config.EndpointsConfig["test"]
	if !isSet {
		return fmt.Errorf("\"test\" network has not been created")
	}
	if !reflect.DeepEqual(network.Links, expectedLinks) {
		return fmt.Errorf("Links was %v but should be %v", network.Links, expectedLinks)
	}

	var err error
	switch name {
	case "aliases":
		err = verifyValue(expected, network.Aliases)
	case "ipv4_address":
		err = verifyValue(expected, network.IPAddress)
	case "ipv6_address":
		err = verifyValue(expected, network.GlobalIPv6Address)
	}
	if err != nil {
		return fmt.Errorf("%s: %s", name, err.Error())
	}
	return nil
}

func verifyValue(expected, actual interface{}) error {
	if !reflect.DeepEqual(expected, actual) {
		return fmt.Errorf("\"%v\" should be \"%v\"", actual, expected)
	}
	return nil
}
