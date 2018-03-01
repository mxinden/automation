package kubernetes

import (
	"github.com/mxinden/automation/executor"
	"k8s.io/api/core/v1"
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

	stepResult, err := k.executeStep(stepConfig)
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

	stepResult, err := k.executeStep(stepConfig)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range stepResult.Containers {
		if c.ExitCode == 0 {
			t.Fatalf("expected step to fail, but got exit code %v", c.ExitCode)
		}
	}
}

func TestExecuteStepEnv(t *testing.T) {
	t.Parallel()

	k := NewKubernetesExecutor("automation")

	stepConfig := executor.StepConfiguration{
		Containers: []executor.ContainerConfiguration{
			{
				Command: "echo $TEST_KEY",
				Image:   "debian",
				Env: []v1.EnvVar{
					{
						Name:  "TEST_KEY",
						Value: "TEST_VALUE",
					},
				},
			},
		},
	}

	stepResult, err := k.executeStep(stepConfig)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(stepResult.Output) != stepConfig.Containers[0].Env[0].Value {
		t.Fatalf(
			"expected output to be %v but got %v",
			stepConfig.Containers[0].Env[0].Value,
			stepResult.Output,
		)
	}
}

func TestWorkingDir(t *testing.T) {
	t.Parallel()

	k := NewKubernetesExecutor("automation")

	stepConfig := executor.StepConfiguration{
		Containers: []executor.ContainerConfiguration{
			{
				Command:    "pwd",
				Image:      "debian",
				WorkingDir: "/etc",
			},
		},
	}

	stepResult, err := k.executeStep(stepConfig)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(stepResult.Output) != stepConfig.Containers[0].WorkingDir {
		t.Fatalf(
			"expected output to be %v but got %v",
			stepConfig.Containers[0].WorkingDir,
			stepResult.Output,
		)
	}
}

func TestExecuteStepInitContainerShareDataWithContainer(t *testing.T) {
	t.Parallel()

	k := NewKubernetesExecutor("automation")

	stepConfig := executor.StepConfiguration{
		Volumes: []v1.Volume{
			{
				Name: "sample-volume",
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		InitContainers: []executor.ContainerConfiguration{
			{
				Command: "touch /sample_dir/testfile.txt",
				Image:   "debian",
				VolumeMounts: []executor.VolumeMount{
					{
						Name:      "sample-volume",
						MountPath: "/sample_dir",
					},
				},
			},
		},
		Containers: []executor.ContainerConfiguration{
			{
				Command: "cat /sample_dir/testfile.txt",
				Image:   "debian",
				VolumeMounts: []executor.VolumeMount{
					{
						Name:      "sample-volume",
						MountPath: "/sample_dir",
					},
				},
			},
		},
	}

	stepResult, err := k.executeStep(stepConfig)
	if err != nil {
		t.Fatal(err)
	}

	if len(stepResult.InitContainers) != 1 {
		t.Fatalf("expected 1 init container result but got %v", len(stepResult.InitContainers))
	}

	if len(stepResult.Containers) != 1 {
		t.Fatalf("expected 1 container result but got %v", len(stepResult.Containers))
	}

	if stepResult.Containers[0].ExitCode != 0 {
		t.Fatalf("expected container to exit with 0 but got %v with logs \n %v", stepResult.Containers[0].ExitCode, stepResult.Output)
	}
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
