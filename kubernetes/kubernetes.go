package kubernetes

import (
	"fmt"
	"github.com/mxinden/automation/repository"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"strconv"
	"time"
)

var Namespace string

func RunRepositoryTest(config repository.Configuration, owner, name, sha string) (string, int32, error) {
	output := ""
	exitCode := int32(1)

	kubeClient, err := createKubeClient()
	if err != nil {
		return output, exitCode, err
	}

	repositoryURL := fmt.Sprintf("https://github.com/%v/%v.git", owner, name)
	output, exitCode, err = createJob(strconv.FormatInt(time.Now().Unix(), 10), kubeClient, repositoryURL, sha, config.Command, config.Image)
	if err != nil {
		return output, exitCode, err
	}

	return output, exitCode, nil
}

func createJob(jobName string, kubeClient *kubernetes.Clientset, repositoryURL, sha string, command string, image string) (string, int32, error) {
	output := ""
	exitCode := int32(1)

	job := makeJobDefinition(jobName, repositoryURL, sha, command, image)
	log.Println("create k8s job")
	job, err := kubeClient.BatchV1().Jobs(Namespace).Create(job)
	if err != nil {
		return output, exitCode, err
	}

	log.Println("wait for job to finish")
	err = wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		job, err := kubeClient.BatchV1().Jobs(Namespace).Get(job.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if job.Status.Succeeded == 1 || job.Status.Failed == 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return output, exitCode, err
	}

	output, exitCode, err = getJobResult(kubeClient, job.UID)
	if err != nil {
		return output, exitCode, err
	}

	return output, exitCode, nil
}

func getJobResult(kubeClient *kubernetes.Clientset, jobUID types.UID) (string, int32, error) {
	output := ""
	exitCode := int32(1)

	pods, err := getPodsOfJob(kubeClient, Namespace, jobUID)
	if err != nil {
		return output, exitCode, err
	}

	log.Println("retrieve output and exit code of pod")
	for _, pod := range pods {
		exitCode = pod.Status.InitContainerStatuses[0].State.Terminated.ExitCode
		if exitCode != 0 {
			options := &v1.PodLogOptions{Container: "repository"}
			req := kubeClient.CoreV1().Pods(Namespace).GetLogs(pod.ObjectMeta.Name, options)
			result, err := req.Do().Raw()
			if err != nil {
				return output, exitCode, err
			}
			return string(result), exitCode, nil
		}
		exitCode = pod.Status.ContainerStatuses[0].State.Terminated.ExitCode
		req := kubeClient.CoreV1().Pods(Namespace).GetLogs(pod.ObjectMeta.Name, &v1.PodLogOptions{})
		result, err := req.Do().Raw()
		if err != nil {
			return output, exitCode, err
		}
		return string(result), exitCode, nil
	}

	return output, exitCode, nil
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
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func makeJobDefinition(jobName, repositoryURL, ref string, command string, image string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pi",
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "automation",
					Containers: []v1.Container{
						{
							Name:       "debian",
							Image:      image,
							Args:       []string{"/bin/bash", "-c", command},
							WorkingDir: "/go/src/github.com/mxinden/automation",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "repository",
									MountPath: "/go/src/github.com/mxinden/automation",
								},
							},
						},
					},
					InitContainers: []v1.Container{
						{
							Name:    "repository",
							Image:   "governmentpaas/git-ssh",
							Command: []string{"/bin/bash", "-c"},
							Args:    []string{fmt.Sprintf("git clone $(REPOSITORY) /go/src/github.com/mxinden/automation && cd /go/src/github.com/mxinden/automation && git checkout %v", ref)},
							Env: []v1.EnvVar{
								{
									Name:  "REPOSITORY",
									Value: repositoryURL,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "repository",
									MountPath: "/go/src/github.com/mxinden/automation",
								},
							},
						},
					},
					RestartPolicy: "Never",
					Volumes: []v1.Volume{
						{
							Name: "repository",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

}
