package github

import (
	"github.com/google/go-github/github"
	"github.com/mxinden/automation/executor"
	"testing"
)

func TestAddEnvVars(t *testing.T) {
	t.Parallel()

	repo := github.Repository{}
	head := github.PullRequestBranch{}
	pr := github.PullRequest{
		Head: &head,
	}
	e := github.PullRequestEvent{
		Repo:        &repo,
		PullRequest: &pr,
	}
	c := executor.ExecutionConfiguration{
		Stages: []executor.StageConfiguration{
			{
				Steps: []executor.StepConfiguration{
					{
						Containers: []executor.ContainerConfiguration{
							{},
						},
					},
				},
			},
		},
	}

	url := "my fancy url"
	e.Repo.CloneURL = &url
	sha := "custom sha"
	e.PullRequest.Head.SHA = &sha

	enrichedConfig, err := addEnvVars(e, c)
	if err != nil {
		t.Fatal(err)
	}

	expectedVars := []struct {
		Name  string
		Value string
	}{
		{"GIT_REPOSITORY_URL", url},
		{"GIT_REF", sha},
	}

	for _, envVar := range expectedVars {
		if !findEnvVarInConfig(envVar.Name, envVar.Value, enrichedConfig) {
			t.Fatalf("expected env var with name '%v' and value '%v' to be added to config", envVar.Name, envVar.Value)
		}
	}
}

func findEnvVarInConfig(name, value string, config executor.ExecutionConfiguration) bool {
	for _, stage := range config.Stages {
		for _, step := range stage.Steps {
			for _, container := range step.Containers {
				for _, envVar := range container.Env {
					if envVar.Name == name && envVar.Value == value {
						return true
					}
				}
			}
		}
	}
	return false
}
