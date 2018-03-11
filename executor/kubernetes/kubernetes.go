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
	"strings"
	"sync"
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
		stageResult, err := k.executeStage(stage)
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

func (k *KubernetesExecutor) executeStage(s executor.StageConfiguration) (executor.StageResult, error) {
	var wg sync.WaitGroup
	stepResults := make(chan executor.StepResult, len(s.Steps))
	stepErrors := make(chan error, len(s.Steps))

	for _, step := range s.Steps {
		wg.Add(1)
		go func(step executor.StepConfiguration) {
			defer wg.Done()
			stepResult, err := k.executeStep(step)
			if err != nil {
				stepErrors <- err
			} else {
				stepResults <- stepResult
			}
		}(step)
	}

	wg.Wait()
	close(stepResults)
	close(stepErrors)

	stageResult := executor.StageResult{}

	combinedError := []string{}
	for err := range stepErrors {
		combinedError = append(combinedError, err.Error())
	}
	if len(combinedError) != 0 {
		return stageResult, errors.New(strings.Join(combinedError, "\n"))
	}

	for stepResult := range stepResults {
		stageResult.Steps = append(stageResult.Steps, stepResult)
	}

	return stageResult, nil
}

func (k *KubernetesExecutor) executeStep(step executor.StepConfiguration) (executor.StepResult, error) {
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

	stepResult, err = k.getJobResult(kubeClient, job.Name)
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to get job result for job %v", job.ObjectMeta.Name)
	}

	return stepResult, nil
}

func waitForJobToFinish(kubeClient *kubernetes.Clientset, namespace string, jobName string) error {
	err := wait.Poll(time.Second, 30*time.Minute, func() (bool, error) {
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

func (k *KubernetesExecutor) getJobResult(kubeClient *kubernetes.Clientset, jobName string) (executor.StepResult, error) {
	stepResult := executor.StepResult{}

	job, err := kubeClient.BatchV1().Jobs(k.namespace).Get(jobName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrapf(err, "failed to retrieve job %v for StartTime and CompletionTime", job.Name)
		return stepResult, err
	}

	if job.Status.StartTime != nil {
		stepResult.StartTime = job.Status.StartTime.Time
	}

	if job.Status.CompletionTime != nil {
		stepResult.CompletionTime = job.Status.CompletionTime.Time
	}

	pods, err := getPodsOfJob(kubeClient, k.namespace, job.UID)
	if err != nil {
		return stepResult, errors.Wrapf(err, "failed to get pods of job uid %v", job.UID)
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
	job.Spec.Template.Spec.ServiceAccountName = config.ServiceAccountName
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
		Name:            getRandomName(),
		Image:           config.Image,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{config.Command},
		Env:             config.Env,
		VolumeMounts:    volumeMounts,
		WorkingDir:      config.WorkingDir,
		SecurityContext: config.SecurityContext,
	}

	return container
}
