package executor

import (
	"k8s.io/api/core/v1"
)

type ExecutionConfiguration struct {
	Stages []StageConfiguration `yaml:"stages"`
}

type StageConfiguration struct {
	Steps []StepConfiguration `yaml:"steps"`
}

type StepConfiguration struct {
	InitContainers []ContainerConfiguration `yaml:"initContainers"`
	Containers     []ContainerConfiguration `yaml:"containers"`
	Volumes        []v1.Volume              `yaml:"volumes"`
}

type ContainerConfiguration struct {
	Command      string        `yaml:"command"`
	Image        string        `yaml:"image"`
	Env          []v1.EnvVar   `yaml:"env"`
	VolumeMounts []VolumeMount `yaml:"volumeMounts"`
	WorkingDir   string        `yaml:"workingDir"`
}

type VolumeMount struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
}

type ExecutionResult struct {
	Stages []StageResult
}

func (r *ExecutionResult) DidSucceed() bool {
	for _, stage := range r.Stages {
		if !stage.DidSucceed() {
			return false
		}
	}
	return true
}

type StageResult struct {
	Steps []StepResult
}

func (r *StageResult) DidSucceed() bool {
	for _, stepResult := range r.Steps {
		for _, containerResult := range stepResult.Containers {
			if containerResult.ExitCode != 0 {
				return false
			}
		}
		for _, initContainerResult := range stepResult.InitContainers {
			if initContainerResult.ExitCode != 0 {
				return false
			}

		}
	}
	return true
}

type StepResult struct {
	InitContainers []ContainerResult
	Containers     []ContainerResult
	Output         string
}

type ContainerResult struct {
	ExitCode int32
}
