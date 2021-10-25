package compose_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/rmasp98/go-compose/compose"
	"gopkg.in/yaml.v3"
)

func TestErrorOnInvalidVersionFormat(t *testing.T) {
	yamlData := parseYaml("version: \"test\"")
	if _, err := compose.NewStack(yamlData); err == nil {
		t.Errorf("Parse did not return an error")
	}
}

func TestErrorOnVersionGreaterThan3Point8(t *testing.T) {
	yamlData := parseYaml("version: \"3.9\"")
	if _, err := compose.NewStack(yamlData); err == nil {
		t.Errorf("Parse did not return an error")
	}
}

func TestErrorOnVersionLessThank3Point0(t *testing.T) {
	yamlData := parseYaml("version: \"2.9\"")
	if _, err := compose.NewStack(yamlData); err == nil {
		t.Errorf("Parse did not return an error")
	}
}

func TestErrorIfNetworksNotMap(t *testing.T) {
	yamlData := parseYaml("networks: \"Not a network\"")
	if _, err := compose.NewStack(yamlData); err == nil {
		t.Errorf("Should have returned an error but returned nothing")
	}
}

func TestParseNetworks(t *testing.T) {
	networkCompose := "networks:\n   testNetwork:\n      driver: \"host\""
	yamlData := parseYaml(networkCompose)
	stack, _ := compose.NewStack(yamlData)
	network := stack.GetNetworkCreate("testNetwork")
	if network.Driver != "host" {
		t.Errorf("Driver should be \"host\" but got \"%s\"", network.Driver)
	}
}

func TestErrorIfVolumesNotMap(t *testing.T) {
	yamlData := parseYaml("volumes: \"Not a volume\"")
	if _, err := compose.NewStack(yamlData); err == nil {
		t.Errorf("Should have returned an error but returned nothing")
	}
}

func TestParseVolumes(t *testing.T) {
	volumeCompose := "volumes:\n   testVolume:\n      driver: \"local\""
	yamlData := parseYaml(volumeCompose)
	stack, _ := compose.NewStack(yamlData)
	volume := stack.GetVolumeCreate("testVolume")
	if volume.Driver != "local" {
		t.Errorf("Driver should be \"local\" but got \"%s\"", volume.Driver)
	}
}

func TestErrorIfServicesNotMap(t *testing.T) {
	yamlData := parseYaml("services: \"Not a service\"")
	if _, err := compose.NewStack(yamlData); err == nil {
		t.Errorf("Should have returned an error but returned nothing")
	}
}

func TestParseServicess(t *testing.T) {
	serviceCompose := "services:\n   testService:\n      image: \"SomeImage\""
	yamlData := parseYaml(serviceCompose)
	stack, _ := compose.NewStack(yamlData)
	service := stack.GetServiceContainerCreate("testService")
	if service.Image != "SomeImage" {
		t.Errorf("Driver should be \"SomeImage\" but got \"%s\"", service.Image)
	}
}

func TestCanParseFullComposeFile(t *testing.T) {
	composeFile, _ := ioutil.ReadFile("../test_data/compose.yaml")
	yamlData := parseYaml(string(composeFile))
	_, err := compose.NewStack(yamlData)
	if err != nil {
		t.Error(err)
	}
}

func parseYaml(data string) interface{} {
	var yamlOut interface{}
	if err := yaml.Unmarshal([]byte(data), &yamlOut); err != nil {
		fmt.Printf("Cannot parse test yaml:\n%s", data)
		os.Exit(-1)
	}
	return yamlOut
}
