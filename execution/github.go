package execution

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
)

type GithubExecution struct {
	Owner    string
	Name     string
	ref      string
	client   *github.Client
	ctx      context.Context
	status   ExecutionStatus
	exitCode int32
	logs     string
	prNumber int
}

func NewGithubExecution(owner, name, ref string, prNumber int) *GithubExecution {
	e := &GithubExecution{Owner: owner, Name: name, ref: ref, prNumber: prNumber}
	t := os.Getenv("GITHUB_API_TOKEN")
	e.ctx = context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t},
	)
	tc := oauth2.NewClient(e.ctx, ts)
	e.client = github.NewClient(tc)

	return e
}

func (e *GithubExecution) GetOwner() string {
	return e.Owner
}

func (e *GithubExecution) GetName() string {
	return e.Name
}

func (e *GithubExecution) GetRef() string {
	return e.ref

}

type Configuration struct {
	Command string `yaml:"command"`
	Image   string `yaml:"image"`
}

func (r *GithubExecution) GetConfiguration() (Configuration, error) {
	log.Printf("get configuration for repository %v/%v\n", r.Owner, r.Name)
	var config Configuration
	ctx := context.Background()

	client := github.NewClient(&http.Client{})

	file, _, _, err := client.Repositories.GetContents(ctx, r.Owner, r.Name, "automation-config.yaml", &github.RepositoryContentGetOptions{Ref: r.GetRef()})
	if err != nil {
		log.Fatal(err)
	}

	rawConfig, err := file.GetContent()
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal([]byte(rawConfig), &config)
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

func (e *GithubExecution) SetStatusPending() error {
	e.status = ExecutionStatusPending
	return e.updateGithubCommitStatus(ExecutionStatusPending)
}

func (e *GithubExecution) SetStatusSuccess(exitCode int32, logs string) error {
	e.status = ExecutionStatusSuccess
	e.exitCode = exitCode
	e.logs = logs

	err := e.addResultAsPRComment(ExecutionStatusSuccess)
	if err != nil {
		return err
	}
	return e.updateGithubCommitStatus(ExecutionStatusSuccess)
}

func (e *GithubExecution) SetStatusFailure(exitCode int32, logs string) error {
	e.status = ExecutionStatusFailure
	e.exitCode = exitCode
	e.logs = logs

	err := e.addResultAsPRComment(ExecutionStatusFailure)
	if err != nil {
		return err
	}
	return e.updateGithubCommitStatus(ExecutionStatusFailure)
}

func (e *GithubExecution) updateGithubCommitStatus(s ExecutionStatus) error {
	context := "Automation"
	state := string(s)
	status := github.RepoStatus{
		State:   &state,
		Context: &context,
	}

	_, _, err := e.client.Repositories.CreateStatus(e.ctx, e.Owner, e.Name, e.ref, &status)
	return err
}

func (e *GithubExecution) addResultAsPRComment(s ExecutionStatus) error {
	body := "Result for " + e.GetRef() + ": " + string(s) + "<details><summary>Logs</summary><p>\n\n```\n" + e.logs + "\n```\n</p></details>"
	comment := github.IssueComment{
		Body: &body,
	}
	_, _, err := e.client.Issues.CreateComment(e.ctx, e.Owner, e.Name, e.prNumber, &comment)
	return err
}
