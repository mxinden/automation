package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthorAssociationCheck(t *testing.T) {
	body, err := os.Open("../scripts/sample-github-payload-CONTRIBUTOR.json")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/trigger", body)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	trigger(recorder, req)

	statusCode := recorder.Result().StatusCode

	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected http status to be %v, but got %v", http.StatusBadRequest, statusCode)
	}
}
