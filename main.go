package main

import (
	"github.com/mxinden/automation/api"
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/kubernetes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

func main() {
	config, err := configuration.Parse()
	if err != nil {
		panic(err)
	}

	kubernetesExecutor := kubernetes.NewKubernetesExecutor(config.Namespace)

	automationAPI := api.NewAPI(config, &kubernetesExecutor)
	automationAPI.RegisterHandlers()

	http.Handle("/metrics", promhttp.Handler())

	log.Fatal(http.ListenAndServe(":8080", nil))
}
