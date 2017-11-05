package configuration

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Repositories []string `yaml:"repositories"`
	Namespace    string   `yaml:"namespace"`
}

func Parse() (Configuration, error) {
	var config Configuration
	rawConfig, err := ioutil.ReadFile("configuration.yaml")
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
