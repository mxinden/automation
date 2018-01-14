package configuration

import (
	"fmt"
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

func (c *Configuration) ContainsRepository(url string) bool {
	if equalsAny(url, c.Repositories) {
		return true
	}
	return false
}

func equalsAny(s string, list []string) bool {
	fmt.Println(s, list)
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
