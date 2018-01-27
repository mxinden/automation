package api

import (
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/kubernetes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var triggerEndpoinTests = []struct {
	requestBodyPath        string
	expectedHTTPStatusCode int
}{
	{"../scripts/sample-github-payload-CONTRIBUTOR.json", http.StatusBadRequest},
	{"../scripts/sample-github-payload-random-repo.json", http.StatusBadRequest},
}

func TestTableTriggerEndpoint(t *testing.T) {
	// Set package variable "config"
	c := configuration.Configuration{
		Repositories: []string{"github.com/mxinden/sample-project"},
	}
	executor := kubernetes.NewKubernetesExecutor("automation")
	automationAPI := NewAPI(c, &executor)

	for _, tt := range triggerEndpoinTests {
		req, err := httpReqFromFile(tt.requestBodyPath)
		if err != nil {
			t.Fatal(err)
		}

		recorder := httptest.NewRecorder()

		automationAPI.triggerHandler(recorder, req)

		statusCode := recorder.Result().StatusCode

		if statusCode != tt.expectedHTTPStatusCode {
			t.Fatalf("expected http status to be %v, but got %v", tt.expectedHTTPStatusCode, statusCode)
		}

	}
}

func httpReqFromFile(p string) (*http.Request, error) {
	body, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "/trigger", body)
	if err != nil {
		return nil, err
	}
	return req, err
}
