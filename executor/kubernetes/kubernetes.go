package kubernetes

import (
	"fmt"
	"github.com/mxinden/automation/executor"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"time"
)

type KubernetesExecutor struct {
	namespace string
}

func NewKubernetesExecutor(ns string) KubernetesExecutor {
	return KubernetesExecutor{
		namespace: ns,
	}
}

func (k *KubernetesExecutor) Execute(c executor.ExecutionConfiguration) (executor.ExecutionResult, error) {
	executionResult := executor.ExecutionResult{}

	for _, stage := range c.Stages {
		stageResult, err := k.ExecuteStage(stage)
		if err != nil {
			return executionResult, err
		}

		executionResult.Stages = append(executionResult.Stages, stageResult)

		if !stageResult.DidSucceed() {
			return executionResult, nil
		}
	}

	return executionResult, nil
}

// TODO: Does not need to be public method
func (k *KubernetesExecutor) ExecuteStage(s executor.StageConfiguration) (executor.StageResult, error) {
	stageResult := executor.StageResult{}

	// TODO: Steps should be executed in parallel
	for _, step := range s.Steps {
		stepResult, err := k.ExecuteStep(step)
		if err != nil {
			return stageResult, err
		}

		stageResult.Steps = append(stageResult.Steps, stepResult)
	}

	return stageResult, nil
}

// TODO: Does not need to be public method
func (k *KubernetesExecutor) ExecuteStep(step executor.StepConfiguration) (executor.StepResult, error) {
	stepResult := executor.StepResult{}

	kubeClient, err := createKubeClient()
	if err != nil {
		return stepResult, errors.Wrap(err, "faile to create kubeclient")
	}

	job := stepConfigToK8sJob(step)

	job, err = kubeClient.BatchV1().Jobs(k.namespace).Create(job)
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to create job %v", job.ObjectMeta.Name)
	}

	err = waitForJobToFinish(kubeClient, k.namespace, job.ObjectMeta.Name)
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to waitForJobToFinish for job %v", job.ObjectMeta.Name)
	}

	stepResult, err = k.getJobResult(kubeClient, job.UID)
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to get job result for job %v", job.ObjectMeta.Name)
	}

	return stepResult, nil
}

func waitForJobToFinish(kubeClient *kubernetes.Clientset, namespace string, jobName string) error {
	err := wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		job, err := kubeClient.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (k *KubernetesExecutor) getJobResult(kubeClient *kubernetes.Clientset, jobUID types.UID) (executor.StepResult, error) {
	stepResult := executor.StepResult{}

	pods, err := getPodsOfJob(kubeClient, k.namespace, jobUID)
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to get pods of job uid %v", jobUID)
	}

	if len(pods) != 1 {
		return stepResult, fmt.Errorf("retrieving job result: expected 1 pod, but got %v", len(pods))
	}

	pod := pods[0]

	for _, c := range pod.Status.InitContainerStatuses {
		stepResult.InitContainers = append(stepResult.InitContainers, getContainerResult(c))
	}

	for _, c := range pod.Status.ContainerStatuses {
		stepResult.Containers = append(stepResult.Containers, getContainerResult(c))
	}

	options := &v1.PodLogOptions{}
	req := kubeClient.CoreV1().Pods(k.namespace).GetLogs(pod.ObjectMeta.Name, options)
	result, err := req.Do().Raw()
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to retrieve logs for pod %v", pod.ObjectMeta.Name)
	}
	stepResult.Output = string(result)

	return stepResult, nil
}

func getContainerResult(s v1.ContainerStatus) executor.ContainerResult {
	exitCode := int32(0)
	if s.State.Terminated != nil {
		exitCode = s.State.Terminated.ExitCode
	}
	return executor.ContainerResult{
		ExitCode: exitCode,
	}
}

func getPodsOfJob(kubeClient *kubernetes.Clientset, namespace string, uid types.UID) ([]v1.Pod, error) {
	pods := []v1.Pod{}

	podList, err := kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return pods, err
	}

	for _, pod := range podList.Items {
		for _, ref := range pod.OwnerReferences {
			if ref.UID == uid && ref.Kind == "Job" {
				pods = append(pods, pod)
				continue
			}
		}
	}

	return pods, nil
}

func createKubeClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("failed to get in-cluster k8s client-go configuration, trying out-of-cluster next")
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

func stepConfigToK8sJob(config executor.StepConfiguration) *batchv1.Job {

	containers := containerConfsToK8sContainers(config.Containers)
	initContainers := containerConfsToK8sContainers(config.InitContainers)

	job := &batchv1.Job{}

	job.ObjectMeta.Name = getRandomName()
	// TODO: Make this configurable
	job.Spec.Template.Spec.ServiceAccountName = "automation"
	job.Spec.Template.Spec.RestartPolicy = "Never"
	job.Spec.Template.Spec.Containers = containers
	job.Spec.Template.Spec.InitContainers = initContainers
	job.Spec.Template.Spec.Volumes = config.Volumes
	job.Spec.BackoffLimit = new(int32)

	return job
}

func containerConfsToK8sContainers(configs []executor.ContainerConfiguration) []v1.Container {
	containers := []v1.Container{}

	for _, c := range configs {
		containers = append(containers, containerConfToK8sContainer(c))
	}

	return containers
}

func containerConfToK8sContainer(config executor.ContainerConfiguration) v1.Container {
	volumeMounts := []v1.VolumeMount{}
	for _, m := range config.VolumeMounts {
		volumeMounts = append(volumeMounts, v1.VolumeMount{Name: m.Name, MountPath: m.MountPath})
	}

	container := v1.Container{
		Name:         getRandomName(),
		Image:        config.Image,
		Command:      []string{"/bin/bash", "-c"},
		Args:         []string{config.Command},
		Env:          config.Env,
		VolumeMounts: volumeMounts,
		WorkingDir:   config.WorkingDir,
	}

	return container
}
