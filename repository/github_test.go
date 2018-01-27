package repository

import (
	"fmt"
	"testing"
)

func TestGetRepositoryConfiguration(t *testing.T) {
	t.Parallel()
	r := NewGithubRepository("mxinden", "sample-project")
	config, err := r.GetConfiguration("master")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(config.Command)
}
