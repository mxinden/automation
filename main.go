package main

import (
	"github.com/mxinden/automation/configuration"
	"github.com/mxinden/automation/connector/github"
	"github.com/mxinden/automation/executor/kubernetes"
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
	githubConnector := github.NewGithubConnector(config, &kubernetesExecutor)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/api/github/trigger", githubConnector.TriggerHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
