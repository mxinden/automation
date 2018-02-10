package execution

import (
	"testing"
)

func TestGetRepositoryConfiguration(t *testing.T) {
	t.Parallel()
	r := NewGithubExecution("mxinden", "sample-project", "master", 1)
	config, err := r.GetConfiguration()
	if err != nil {
		t.Fatal(err)
	}

	if config.Stages[0].Command != "./test.sh" {
		t.Fatalf("expected %v but got %v", "./test.sh", config.Stages[0].Command)
	}
}
