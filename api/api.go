package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/execution"
	"github.com/mxinden/automation/kubernetes"
)

var config configuration.Configuration

type API struct {
	config   configuration.Configuration
	executor kubernetes.Executor
}

func NewAPI(c configuration.Configuration, e kubernetes.Executor) API {
	return API{config: c, executor: e}
}

func (api *API) HandleRequests() {
	http.HandleFunc("/trigger", api.triggerHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type triggerPayload struct {
	Respository repository  `json:"repository"`
	PullRequest pullRequest `json:"pull_request"`
}

type repository struct {
	FullName string `json:"full_name"`
}

type AuthorAssociation string

var (
	AuthorAssociationCOLLABORATOR AuthorAssociation = "COLLABORATOR"
	AuthorAssociationMEMBER       AuthorAssociation = "MEMBER"
	AuthorAssociationOWNER        AuthorAssociation = "OWNER"
)

type pullRequest struct {
	Head              head              `json:"head"`
	Number            int               `json:"number"`
	AuthorAssociation AuthorAssociation `json:"author_association"`
}

type head struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}

func (api *API) triggerHandler(w http.ResponseWriter, r *http.Request) {
	s := os.Getenv("GITHUB_WEBHOOK_SECRET")
	payload, err := github.ValidatePayload(r, []byte(s))
	if err != nil {
		log.Printf("Error validating payload: %v", err)
		http.Error(w, "error validating payload", http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("Error parsing payload: %v", err)
		http.Error(w, "error parsing payload", http.StatusBadRequest)
		return
	}

	pullRequestEvent, ok := event.(*github.PullRequestEvent)
	if !ok {
		log.Printf("Error, expecting pull request event but got: %v", github.WebHookType(r))
		http.Error(w, "error expecting pull request event", http.StatusBadRequest)
		return
	}

	err = checkPermissions(api.config, pullRequestEvent)
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}

	e := execution.NewExecution(
		api.executor,
		pullRequestEvent.Repo.GetOwner().GetLogin(),
		pullRequestEvent.Repo.GetName(),
		*pullRequestEvent.PullRequest.GetHead().SHA,
		pullRequestEvent.PullRequest.GetNumber(),
	)
	output, exitCode, err := e.Execute()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"testing repository %v returned:\n%v\nwith exit code: %v",
		pullRequestEvent.GetRepo().GetFullName(),
		output,
		exitCode,
	)
}

func checkPermissions(c configuration.Configuration, event *github.PullRequestEvent) error {
	event.PullRequest.GetAuthorAssociation()

	if !equalsAny(
		event.PullRequest.GetAuthorAssociation(),
		[]AuthorAssociation{AuthorAssociationCOLLABORATOR, AuthorAssociationMEMBER, AuthorAssociationOWNER},
	) {
		return errors.New(
			fmt.Sprintf(
				"event author not one of %v, %v, %v",
				AuthorAssociationOWNER,
				AuthorAssociationMEMBER,
				AuthorAssociationCOLLABORATOR,
			),
		)
	}

	if !c.ContainsRepository("github.com/" + event.Repo.GetFullName()) {
		return errors.New(fmt.Sprintf(
			"%v is not a configured repository",
			event.Repo.GetFullName(),
		))
	}
	return nil
}

func equalsAny(s string, list []AuthorAssociation) bool {
	for _, e := range list {
		if string(e) == s {
			return true
		}
	}
	return false
}
