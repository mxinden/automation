package repository

import (
	"context"
	"github.com/google/go-github/github"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
)

type Configuration struct {
	Command string `yaml:"command"`
}

func GetConfigurationFromGitHub(owner string, repositoryName string, ref string) (Configuration, error) {
	log.Printf("get configuration for repository %v/%v\n", owner, repositoryName)
	var config Configuration
	ctx := context.Background()

	client := github.NewClient(&http.Client{})

	file, _, _, err := client.Repositories.GetContents(ctx, owner, repositoryName, "automation-config.yaml", &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		log.Fatal(err)
	}

	rawConfig, err := file.GetContent()
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal([]byte(rawConfig), &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
