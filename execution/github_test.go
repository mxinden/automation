package execution

import (
	"fmt"
	"testing"
)

func TestGetRepositoryConfiguration(t *testing.T) {
	t.Parallel()
	r := NewGithubExecution("mxinden", "sample-project", "master", 1)
	config, err := r.GetConfiguration()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(config.Command)
}
