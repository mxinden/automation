package github

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/mxinden/automation/configuration"
)

type AuthorAssociation string

var (
	AuthorAssociationCOLLABORATOR AuthorAssociation = "COLLABORATOR"
	AuthorAssociationMEMBER       AuthorAssociation = "MEMBER"
	AuthorAssociationOWNER        AuthorAssociation = "OWNER"
)

func (c *GithubConnector) TriggerHandler(w http.ResponseWriter, r *http.Request) {
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

	err = checkPermissions(c.config, pullRequestEvent)
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}

	go log.Println(c.runFromPREvent(*pullRequestEvent))

	log.Printf(
		"Triggered execution for repository %v",
		pullRequestEvent.GetRepo().GetFullName(),
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
