package main

import (
	"fmt"
	"questspace/pkg/application"
)

var config struct {
	Section struct {
		Key string `yaml:"key"`
	} `yaml:"section"`
}

func Init(app application.App) error {
	fmt.Printf("Got key: %s", config.Section.Key)
	return nil
}

func main() {
	application.Run(Init, &config)
}
