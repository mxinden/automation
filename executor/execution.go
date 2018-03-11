package executor

import (
	"io"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"time"
)

type ExecutionConfiguration struct {
	Stages []StageConfiguration `yaml:"stages"`
}

func DecodeExecutionConfiguration(r io.Reader) (ExecutionConfiguration, error) {
	c := ExecutionConfiguration{}
	err := yaml.NewYAMLOrJSONDecoder(r, 4096).Decode(&c)
	return c, err
}

type StageConfiguration struct {
	Steps []StepConfiguration `yaml:"steps"`
}

type StepConfiguration struct {
	InitContainers     []ContainerConfiguration `yaml:"initContainers"`
	Containers         []ContainerConfiguration `yaml:"containers"`
	Volumes            []v1.Volume              `yaml:"volumes"`
	ServiceAccountName string                   `yaml:"serviceAccountName"`
}

type ContainerConfiguration struct {
	Command         string              `yaml:"command"`
	Image           string              `yaml:"image"`
	Env             []v1.EnvVar         `yaml:"env"`
	VolumeMounts    []VolumeMount       `yaml:"volumeMounts"`
	WorkingDir      string              `yaml:"workingDir"`
	SecurityContext *v1.SecurityContext `yaml:"securityContext"`
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
	StartTime      time.Time
	CompletionTime time.Time
}

type ContainerResult struct {
	ExitCode int32
}
