package kubernetes

import (
	"github.com/mxinden/automation/executor"
	"strings"
	"testing"
)

// ExecuteStep

func TestExecuteStep(t *testing.T) {
	t.Parallel()
	expectedOutput := "test"

	k := NewKubernetesExecutor("automation")

	stepConfig := executor.StepConfiguration{}
	stepConfig.Containers = []executor.ContainerConfiguration{
		{Command: "echo " + expectedOutput, Image: "debian"},
	}

	stepResult, err := k.ExecuteStep(stepConfig)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(stepResult.Output) != expectedOutput {
		t.Fatalf("expected output %v but got output %v", expectedOutput, stepResult.Output)
	}
}

func TestExecuteStepFailure(t *testing.T) {
	t.Parallel()

	k := NewKubernetesExecutor("automation")

	stepConfig := executor.StepConfiguration{}
	stepConfig.Containers = []executor.ContainerConfiguration{
		{Command: "false", Image: "debian"},
	}

	stepResult, err := k.ExecuteStep(stepConfig)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range stepResult.Containers {
		if c.ExitCode == 0 {
			t.Fatalf("expected step to fail, but got exit code %v", c.ExitCode)
		}
	}
}

func TestExecuteStepInitContainer(t *testing.T) {
	t.Skip("TODO: Implement")
}

// ExecuteStage

func TestExecuteStageStepsRunInParallel(t *testing.T) {
	t.Skip("TODO: Implement")
}

// Execute

func TestExecuteDontRunSecondStageIfFirstFails(t *testing.T) {
	t.Parallel()

	k := NewKubernetesExecutor("automation")

	config := executor.ExecutionConfiguration{
		Stages: []executor.StageConfiguration{
			{
				Steps: []executor.StepConfiguration{
					{
						Containers: []executor.ContainerConfiguration{
							{
								Command: "false",
								Image:   "debian",
							},
						},
					},
				},
			},
			{
				Steps: []executor.StepConfiguration{
					{
						Containers: []executor.ContainerConfiguration{
							{
								Command: "echo 'this should never run'",
								Image:   "debian",
							},
						},
					},
				},
			},
		},
	}

	result, err := k.Execute(config)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Stages) != 1 {
		t.Fatalf("expected length of result.Stages to be 1, but got %v", len(result.Stages))
	}
}
