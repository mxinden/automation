package execution

import (
	"context"
	"os"

	"github.com/google/go-github/github"
	"github.com/mxinden/automation/kubernetes"
	"github.com/mxinden/automation/repository"
	"golang.org/x/oauth2"
)

type ExecutionStatus string

var (
	ExecutionStatusPending ExecutionStatus = "pending"
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailure ExecutionStatus = "failure"
)

type Execution struct {
	Owner          string
	RepositoryName string
	Sha            string
	Status         ExecutionStatus
	PRNumber       int
}

func NewExecution(owner, repositoryName, sha string, prNumber int) Execution {
	e := Execution{
		Owner:          owner,
		RepositoryName: repositoryName,
		Sha:            sha,
		Status:         ExecutionStatusPending,
		PRNumber:       prNumber,
	}
	return e
}

func (e *Execution) Execute() (string, int32, error) {
	output := ""
	exitCode := int32(1)

	err := e.changeStatus(ExecutionStatusPending)
	if err != nil {
		return output, exitCode, err
	}

	config, err := repository.GetConfigurationFromGitHub(e.Owner, e.RepositoryName, e.Sha)
	if err != nil {
		return output, exitCode, err
	}

	output, exitCode, err = kubernetes.RunRepositoryTest(config, e.Owner, e.RepositoryName, e.Sha)
	if exitCode == 0 {
		err := e.changeStatus(ExecutionStatusSuccess)
		if err != nil {
			return output, exitCode, err
		}
	} else {
		err := e.changeStatus(ExecutionStatusFailure)
		if err != nil {
			return output, exitCode, err
		}
	}
	return output, exitCode, err
}

func (e *Execution) changeStatus(s ExecutionStatus) error {
	t := os.Getenv("GITHUB_API_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	state := string(s)
	context := "Automation"
	status := github.RepoStatus{
		State:   &state,
		Context: &context,
	}

	_, _, err := client.Repositories.CreateStatus(ctx, e.Owner, e.RepositoryName, e.Sha, &status)
	return err
}
