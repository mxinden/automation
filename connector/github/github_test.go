package github

import (
	"github.com/mxinden/automation/executor"
	"k8s.io/api/core/v1"
	"testing"
)

func TestAddEnvVars(t *testing.T) {
	t.Parallel()

	c := MakeExecutionConfigurationWithOneContainer()

	url := "my fancy url"
	ref := "master"
	sha := "custom sha"

	enrichedConfig, err := addEnvVars(url, ref, sha, c)
	if err != nil {
		t.Fatal(err)
	}

	expectedVars := []struct {
		Name  string
		Value string
	}{
		{"GIT_REPOSITORY_URL", url},
		{"GIT_SHA", sha},
	}

	for _, envVar := range expectedVars {
		if !findEnvVarInConfig(envVar.Name, envVar.Value, enrichedConfig) {
			t.Fatalf("expected env var with name '%v' and value '%v' to be added to config", envVar.Name, envVar.Value)
		}
	}
}

func TestAddEnvVarsAppendsNotReplaces(t *testing.T) {
	t.Parallel()

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
	ref := "master"
	sha := "1234"

	enrichedConfig, err := addEnvVars(url, ref, sha, c)
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

func TestGitReferenceToBranchName(t *testing.T) {
	ref := "refs/heads/push-test"
	expectedBranch := "push-test"

	branch := gitRefToBranchName(ref)

	if branch != expectedBranch {
		t.Fatalf("expected %v but got %v", expectedBranch, branch)
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
