package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/executor"
	"golang.org/x/oauth2"
	"k8s.io/api/core/v1"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type GithubConnector struct {
	config   configuration.Configuration
	executor executor.Executor
}

func NewGithubConnector(c configuration.Configuration, e executor.Executor) GithubConnector {
	return GithubConnector{
		config:   c,
		executor: e,
	}
}

type PRExecution struct {
	owner    string
	name     string
	sha      string
	prNumber int
	client   *github.Client
	ctx      context.Context
}

func NewPRExecution(owner, name, sha string, prNumber int) *PRExecution {
	e := &PRExecution{owner: owner, name: name, sha: sha, prNumber: prNumber}
	t := os.Getenv("GITHUB_API_TOKEN")
	e.ctx = context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t},
	)
	tc := oauth2.NewClient(e.ctx, ts)
	e.client = github.NewClient(tc)

	return e
}

func (c *GithubConnector) runFromPREvent(event github.PullRequestEvent) error {
	// TODO: Still needed?
	e := NewPRExecution(
		*event.Repo.Owner.Login,
		*event.Repo.Name,
		*event.PullRequest.Head.SHA,
		*event.PullRequest.Number,
	)

	err := e.SetStatusPending()
	if err != nil {
		return err
	}

	executionResult, err := c.run(*event.Repo.CloneURL, e.owner, e.name, *event.PullRequest.Head.Ref, *event.PullRequest.Head.SHA)
	if err != nil {
		return err
	}

	err = e.SetStatus(executionResult)
	if err != nil {
		return err
	}

	return nil
}

func (c *GithubConnector) runFromPushEvent(event github.PushEvent) error {
	_, err := c.run(
		*event.Repo.CloneURL,
		*event.Repo.Owner.Name,
		*event.Repo.Name,
		// TODO: Find cleaner solution
		gitRefToBranchName(*event.Ref),
		*event.After,
	)
	return err
}

func (c *GithubConnector) run(repoURL, repoOwner, repoName, branchName, sha string) (executor.ExecutionResult, error) {
	executionResult := executor.ExecutionResult{}
	config, err := GetConfiguration(repoOwner, repoName, sha)
	if err != nil {
		return executionResult, err
	}

	config, err = addEnvVars(repoURL, branchName, sha, config)
	if err != nil {
		return executionResult, err
	}

	return c.executor.Execute(config)
}

func addEnvVars(repoURL, branch, sha string, c executor.ExecutionConfiguration) (executor.ExecutionConfiguration, error) {
	config := c

	env := []v1.EnvVar{
		v1.EnvVar{Name: "GIT_REPOSITORY_URL", Value: repoURL},
		v1.EnvVar{Name: "GIT_SHA", Value: sha},
		v1.EnvVar{Name: "GIT_BRANCH_NAME", Value: branch},
	}

	for stageI, stage := range config.Stages {
		for stepI, step := range stage.Steps {
			for containerI, container := range step.Containers {
				config.Stages[stageI].Steps[stepI].Containers[containerI].Env = append(container.Env, env...)
			}
			for initContainerI, initContainer := range step.InitContainers {
				config.Stages[stageI].Steps[stepI].InitContainers[initContainerI].Env = append(initContainer.Env, env...)
			}
		}
	}

	return config, nil
}

func gitRefToBranchName(ref string) string {
	// extract e.g. "push-test" out of "refs/heads/push-test"
	re := regexp.MustCompile("[^/]*$")
	return re.FindString(ref)
}

func GetConfiguration(owner, name, sha string) (executor.ExecutionConfiguration, error) {
	var config executor.ExecutionConfiguration
	ctx := context.Background()

	client := github.NewClient(&http.Client{})

	file, _, _, err := client.Repositories.GetContents(ctx, owner, name, "automation-config.yaml", &github.RepositoryContentGetOptions{Ref: sha})
	if err != nil {
		return config, err
	}

	rawConfig, err := file.GetContent()
	if err != nil {
		return config, err
	}

	config, err = executor.DecodeExecutionConfiguration(strings.NewReader(rawConfig))
	if err != nil {
		return config, err
	}

	return config, nil
}

type ExecutionStatus string

var (
	ExecutionStatusPending ExecutionStatus = "pending"
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailure ExecutionStatus = "failure"
)

func (e *PRExecution) SetStatusPending() error {
	return e.updateGithubCommitStatus(ExecutionStatusPending)
}

func (e *PRExecution) SetStatus(r executor.ExecutionResult) error {
	executionStatus := ExecutionStatusFailure
	if r.DidSucceed() {
		executionStatus = ExecutionStatusSuccess
	}

	err := e.addResultAsPRComment(executionStatus, r)
	if err != nil {
		return err
	}

	return e.updateGithubCommitStatus(executionStatus)
}

func (e *PRExecution) updateGithubCommitStatus(s ExecutionStatus) error {
	context := "Automation"
	state := string(s)
	status := github.RepoStatus{
		State:   &state,
		Context: &context,
	}

	_, _, err := e.client.Repositories.CreateStatus(e.ctx, e.owner, e.name, e.sha, &status)
	return err
}

func (e *PRExecution) addResultAsPRComment(s ExecutionStatus, r executor.ExecutionResult) error {
	body := "Result for " + e.sha + ": " + string(s) + formatLogsForGithubComment(r)
	comment := github.IssueComment{
		Body: &body,
	}
	_, _, err := e.client.Issues.CreateComment(e.ctx, e.owner, e.name, e.prNumber, &comment)
	return err
}

func formatLogsForGithubComment(r executor.ExecutionResult) string {
	comment := "\n\n"
	for stageI, stageResult := range r.Stages {
		comment = comment + fmt.Sprintf("\n\nStage %v<p>", stageI)

		for stepI, stepResult := range stageResult.Steps {
			comment = comment + fmt.Sprintf("\n\n<details><summary>Step %v</summary><p>", stepI)

			for initContainerI, initContainerResult := range stepResult.InitContainers {
				comment = comment + fmt.Sprintf("\n\nInitContainer %v ExitCode %v", initContainerI, initContainerResult.ExitCode)
			}

			for containerI, containerResult := range stepResult.Containers {
				comment = comment + fmt.Sprintf("\n\nContainer %v ExitCode %v", containerI, containerResult.ExitCode)
			}

			comment = comment + fmt.Sprintf("\n\nLogs: \n\n ```\n\n%v```", stepResult.Output)

			comment = comment + "\n\n</p></details>"
		}

		comment = comment + "\n\n</p>"
	}

	return comment
}
