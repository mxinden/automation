package kubernetes

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRepositoryInitContainer(t *testing.T) {
	t.Parallel()
	command := []string{"make", "test"}
	kubeClient, err := createExternalKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	output, exitCode, err := createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, "https://github.com/mxinden/sample-project", "master", command)
	if err != nil {
		t.Fatal(err)
	}

	if exitCode != 0 {
		t.Fatal("expected zero exit code")
	}

	succeededString := "Tests succeeded"
	if !strings.Contains(output, succeededString) {
		t.Fatalf("expected output \"%v\" to contain \"%v\"", output, succeededString)
	}
}

func TestExecuteCommand(t *testing.T) {
	t.Parallel()
	command := []string{"echo", "test"}
	kubeClient, err := createExternalKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	output, exitCode, err := createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, "https://github.com/mxinden/sample-project", "master", command)
	if err != nil {
		t.Fatal(err)
	}

	if exitCode != 0 {
		t.Error("expected exit code to be zero")
	}

	expected := "test\n"
	if output != expected {
		t.Fatalf("expected %v but got %v", expected, output)
	}
}

func TestExecuteCommandForNonZeroExitCode(t *testing.T) {
	t.Parallel()
	command := []string{"false"}
	kubeClient, err := createExternalKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	_, exitCode, err := createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, "https://github.com/mxinden/sample-project", "master", command)
	if err != nil {
		t.Fatal(err)
	}

	if exitCode == 0 {
		t.Fatalf("expected exit code to be non-zero")
	}
}

func createExternalKubeClient() (*kubernetes.Clientset, error) {
	kubeconfig := "/home/mxinden/.kube/config"

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
