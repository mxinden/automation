package github

import (
	"github.com/google/go-github/github"
	"github.com/mxinden/automation/executor"
	"k8s.io/api/core/v1"
	"testing"
)

func TestAddEnvVars(t *testing.T) {
	t.Parallel()

	e := MakePullRequestEvent()
	c := MakeExecutionConfigurationWithOneContainer()

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

func TestAddEnvVarsAppendsNotReplaces(t *testing.T) {
	t.Parallel()

	e := MakePullRequestEvent()
	c := MakeExecutionConfigurationWithOneContainer()

	key := "pre-existing secret key"

	c.Stages[0].Steps[0].Containers[0].Env = []v1.EnvVar{
		{ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				Key: key,
			},
		}},
	}

	url := "my fancy url"
	e.Repo.CloneURL = &url
	sha := "custom sha"
	e.PullRequest.Head.SHA = &sha

	enrichedConfig, err := addEnvVars(e, c)
	if err != nil {
		t.Fatal(err)
	}

	if enrichedConfig.
		Stages[0].
		Steps[0].
		Containers[0].
		Env[0].
		ValueFrom.
		SecretKeyRef.
		Key != key {
		t.Fatal("expected pre-existing env variables to stay")
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

func MakePullRequestEvent() github.PullRequestEvent {
	repo := github.Repository{}
	head := github.PullRequestBranch{}
	pr := github.PullRequest{
		Head: &head,
	}
	e := github.PullRequestEvent{
		Repo:        &repo,
		PullRequest: &pr,
	}
	return e
}

func MakeExecutionConfigurationWithOneContainer() executor.ExecutionConfiguration {
	return executor.ExecutionConfiguration{
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
}
