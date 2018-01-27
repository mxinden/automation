package main

import (
	"github.com/mxinden/automation/api"
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/kubernetes"
)

func main() {
	config, err := configuration.Parse()
	if err != nil {
		panic(err)
	}

	kubernetesExecutor := kubernetes.NewKubernetesExecutor(config.Namespace)

	automationAPI := api.NewAPI(config, &kubernetesExecutor)

	automationAPI.HandleRequests()
}
