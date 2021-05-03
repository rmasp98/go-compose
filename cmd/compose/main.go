package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rmasp98/go-compose/compose"
	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("go-compose")

	fileContents, readErr := ioutil.ReadFile("compose.yaml")
	if readErr != nil {
		fmt.Println(readErr.Error())
		os.Exit(1)
	}

	var data interface{}
	if err := yaml.Unmarshal(fileContents, &data); err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	compose.NewStack(data)
}
