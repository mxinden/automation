package executor

import (
	"os"
	"testing"
)

func TestDecodeExecutionConfigurationEnv(t *testing.T) {
	rawContent, err := os.Open("./execution_test_env_fixture.yaml")
	if err != nil {
		t.Fatal(err)
	}

	c, err := DecodeExecutionConfiguration(rawContent)
	if err != nil {
		t.Fatal(err)
	}

	if c.Stages[0].Steps[0].Containers[0].Env[0].ValueFrom.SecretKeyRef.Key != "sample-secret-key" {
		t.Fatal("expected env var from secret to be parsed properly in execution configuration decoding")
	}
}

func TestDecodeExecutionConfigurationPrivileged(t *testing.T) {
	rawContent, err := os.Open("./execution_test_privileged_fixture.yaml")
	if err != nil {
		t.Fatal(err)
	}

	c, err := DecodeExecutionConfiguration(rawContent)
	if err != nil {
		t.Fatal(err)
	}

	if !*c.Stages[0].Steps[0].Containers[0].SecurityContext.Privileged {
		t.Fatal("expected container security context to be privileged")
	}
}
