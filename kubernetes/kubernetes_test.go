package kubernetes

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRepositoryInitContainer(t *testing.T) {
	t.Parallel()
	Namespace = "automation"
	command := "./test.sh"
	kubeClient, err := createKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	output, exitCode, err := createJob(
		strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)),
		kubeClient,
		"https://github.com/mxinden/sample-project",
		"master",
		command,
		"golang",
	)
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
	Namespace = "automation"
	command := "echo test"
	kubeClient, err := createKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	output, exitCode, err := createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, "https://github.com/mxinden/sample-project", "master", command, "golang")
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
	Namespace = "automation"
	command := "false"
	kubeClient, err := createKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	_, exitCode, err := createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, "https://github.com/mxinden/sample-project", "master", command, "golang")
	if err != nil {
		t.Fatal(err)
	}

	if exitCode == 0 {
		t.Fatalf("expected exit code to be non-zero")
	}
}