package repository

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
)

type GithubRepository struct {
	Owner string
	Name  string
}

func NewGithubRepository(owner, name string) *GithubRepository {
	return &GithubRepository{Owner: owner, Name: name}
}

type Configuration struct {
	Command string `yaml:"command"`
	Image   string `yaml:"image"`
}

func (r *GithubRepository) GetOwner() string {
	return r.Owner
}

func (r *GithubRepository) GetName() string {
	return r.Name
}

func (r *GithubRepository) GetConfiguration(ref string) (Configuration, error) {
	log.Printf("get configuration for repository %v/%v\n", r.Owner, r.Name)
	var config Configuration
	ctx := context.Background()

	client := github.NewClient(&http.Client{})

	file, _, _, err := client.Repositories.GetContents(ctx, r.Owner, r.Name, "automation-config.yaml", &github.RepositoryContentGetOptions{Ref: ref})
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

func (r *GithubRepository) ChangeStatus(ref, state string) error {
	t := os.Getenv("GITHUB_API_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	context := "Automation"
	status := github.RepoStatus{
		State:   &state,
		Context: &context,
	}

	_, _, err := client.Repositories.CreateStatus(ctx, r.Owner, r.Name, ref, &status)
	return err
}
