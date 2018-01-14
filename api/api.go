package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/execution"
	"log"
	"net/http"
	"strings"
)

var config configuration.Configuration

type API struct {
	config configuration.Configuration
}

func NewAPI(c configuration.Configuration) API {
	return API{config: c}
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
	defer r.Body.Close()

	var payload triggerPayload

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&payload)
	if err != nil {
		log.Printf("Error decoding body: %v", err)
		http.Error(w, "can't decode body", http.StatusBadRequest)
		return
	}

	err = checkPermissions(api.config, payload)
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		return
	}

	fullName := strings.Split(payload.Respository.FullName, "/")

	e := execution.NewExecution(fullName[0], fullName[1], payload.PullRequest.Head.Sha, payload.PullRequest.Number)
	output, exitCode, err := e.Execute()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"testing repository %v returned:\n%v\nwith exit code: %v",
		payload.Respository.FullName,
		output,
		exitCode,
	)
}

func checkPermissions(c configuration.Configuration, p triggerPayload) error {
	if !equalsAny(
		p.PullRequest.AuthorAssociation,
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

	if !config.ContainsRepository("github.com/" + p.Respository.FullName) {
		return errors.New(fmt.Sprintf(
			"%v is not a configured repository",
			p.Respository.FullName,
		))
	}
	return nil
}

func equalsAny(s AuthorAssociation, list []AuthorAssociation) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
