package repository

import (
	"fmt"
	"testing"
)

func TestGetRepositoryConfiguration(t *testing.T) {
	t.Parallel()
	config, err := GetConfigurationFromGitHub("mxinden", "sample-project", "master")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(config.Command)
}
