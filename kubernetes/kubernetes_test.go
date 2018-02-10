package kubernetes

import (
	"github.com/mxinden/automation/execution"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestExecuteMultiStageAllSuccessful(t *testing.T) {
	t.Parallel()
	k := NewKubernetesExecutor("automation")

	e := execution.NewGithubExecution("mxinden", "sample-project", "b91693045f0f9c9cc45b78be2534555fc1fec7eb", 2)
	err := k.Execute(e)
	if err != nil {
		t.Fatal(err)
	}

	if e.GetStatus() != execution.ExecutionStatusSuccess {
		t.Fatalf("expected %v but got %v status", execution.ExecutionStatusSuccess, e.GetStatus())
	}
}

func TestExecuteDontRunSecondStageIfFirstFails(t *testing.T) {
	t.Parallel()
	k := NewKubernetesExecutor("automation")

	e := execution.NewGithubExecution("mxinden", "sample-project", "ea4c7da0f74c82f2dcf832962f772fe6f1943c35", 3)
	err := k.Execute(e)
	if err != nil {
		t.Fatal(err)
	}

	if e.GetStatus() != execution.ExecutionStatusFailure {
		t.Fatalf("expected %v but got %v status", execution.ExecutionStatusFailure, e.GetStatus())
	}

	if !strings.Contains(e.GetLogs(), "first stage") {
		t.Fatal("expected first stage to have run")
	}

	if strings.Contains(e.GetLogs(), "second stage") {
		t.Fatal("expected second stage not to run due to failure in first stage")
	}
}

func TestRepositoryInitContainer(t *testing.T) {
	t.Parallel()
	k := NewKubernetesExecutor("automation")
	command := "./test.sh"
	kubeClient, err := createKubeClient()
	if err != nil {
		t.Fatal(err)
	}

	e := execution.NewGithubExecution("mxinden", "sample-project", "master", 1)
	output, exitCode, err := k.createJob(
		strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)),
		kubeClient,
		e,
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
	k := NewKubernetesExecutor("automation")
	command := "echo test"
	kubeClient, err := createKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	e := execution.NewGithubExecution("mxinden", "sample-project", "master", 1)
	output, exitCode, err := k.createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, e, command, "golang")
	if err != nil {

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
	k := NewKubernetesExecutor("automation")
	command := "false"
	kubeClient, err := createKubeClient()
	if err != nil {
		t.Fatal(err)
	}
	e := execution.NewGithubExecution("mxinden", "sample-project", "master", 1)
	_, exitCode, err := k.createJob(strings.ToLower(t.Name()+strconv.FormatInt(time.Now().Unix(), 10)), kubeClient, e, command, "golang")
	if err != nil {
		t.Fatal(err)
	}

	if exitCode == 0 {
		t.Fatalf("expected exit code to be non-zero")
	}
}
