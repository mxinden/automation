package api

import (
	"encoding/json"
	"github.com/mxinden/automation/execution"
	"log"
	"net/http"
	"strings"
)

func HandleRequests() {
	http.HandleFunc("/trigger", trigger)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type triggerPayload struct {
	Respository repository  `json:"repository"`
	PullRequest pullRequest `json:"pull_request"`
}

type repository struct {
	FullName string `json:"full_name"`
}

type pullRequest struct {
	Head   head `json:"head"`
	Number int  `json:"number"`
}

type head struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}

func trigger(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload triggerPayload
	w.Write([]byte("project triggered\n"))

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&payload)
	if err != nil {
		log.Printf("Error decoding body: %v", err)
		http.Error(w, "can't decode body", http.StatusBadRequest)
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
