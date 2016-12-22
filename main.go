package main

import (
	"os"

	"time"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/zendesk/consul2dogstats/consul2dogstats"
	"github.com/zendesk/consul2dogstats/version"
	datadog "github.com/zorkian/go-datadog-api"
)

func main() {
	log.Infof("Starting %s version git-%s", os.Args[0], version.GitRevision)

	consulLockKeypath := os.Getenv("C2D_LOCK_PATH")
	if consulLockKeypath == "" {
		consulLockKeypath = "consul2dogstats/.lock"
	}
	collectIntervalStr := os.Getenv("C2D_COLLECT_INTERVAL")
	if collectIntervalStr == "" {
		collectIntervalStr = "10s"
	}
	collectInterval, err := time.ParseDuration(collectIntervalStr)
	if err != nil {
		log.Fatal(err)
	}

	datadogAPIKey := os.Getenv("DATADOG_API_KEY")
	if datadogAPIKey == "" {
		log.Fatal("DATADOG_API_KEY environment variable must be set")
	}
	datadogClient := datadog.NewClient(datadogAPIKey, "")
	if ok, err := datadogClient.Validate(); !ok || err != nil {
		if err == nil {
			log.Fatal("Invalid Datadog API key")
		}
		log.Fatal(err)
	}

	consulClient, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	collector, err := consul2dogstats.NewCollector(datadogClient,
		consulClient, consulLockKeypath, collectInterval)
	if err != nil {
		log.Fatal(err)
	}

	if err = collector.Run(nil, nil); err != nil {
		log.Fatal(err)
	}
}
