package main

import (
	"os"
	"syscall"

	"time"

	"os/signal"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/zendesk/consul2dogstats/version"
	"github.com/zorkian/go-datadog-api"
)

func main() {
	log.Infof("Starting %s version git-%s", os.Args[0], version.GitRevision)

	datadogAPIKey := os.Getenv("DATADOG_API_KEY")
	if datadogAPIKey == "" {
		log.Fatal("DATADOG_API_KEY environment variable must be set")
	}

	datadogClient := datadog.NewClient(datadogAPIKey, "")
	if ok, err := datadogClient.Validate(); !ok || err != nil {
		if err == nil {
			log.Fatal("Invalid Datadog API key")
		} else {
			log.Fatal(err)
		}
	}

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

	consulClient, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	lock, err := consulClient.LockKey(consulLockKeypath)
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal)
	doneCh := make(chan struct{})

	for {
		log.Infof("Attempting to acquire lock at %s", consulLockKeypath)
		lockLost, err := lock.Lock(nil)
		if err != nil {
			log.Fatal(err)
		}
		defer lock.Unlock()
		log.Info("Lock acquired")

		go mainLoop(consulClient, datadogClient, collectInterval, doneCh)

		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		select {
		case signal := <-sigCh:
			log.Infof("Received %s signal, terminating cleanly", signal)
			doneCh <- struct{}{}
			return
		case <-lockLost:
			log.Info("Lost Consul lock!  Stopping service poller")
			doneCh <- struct{}{}
			lock.Unlock()
			lock.Destroy()
			break
		}
	}
}

func mainLoop(consulClient *consul.Client, datadogClient *datadog.Client, interval time.Duration, doneCh <-chan struct{}) {
	health := consulClient.Health()
	catalog := consulClient.Catalog()
	queryOptions := consul.QueryOptions{}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-doneCh:
			return
		case <-ticker.C:
			// wait for next tick, then leave select loop
		}

		services, _, err := catalog.Services(&queryOptions)
		if err != nil {
			log.Fatal(err)
		}

		var metrics []datadog.Metric

		for serviceName := range services {
			serviceHealth, _, err := health.Service(serviceName, "", false, &queryOptions)
			if err != nil {
				log.Fatal(err)
			}
			// Initialize the map that will be holding the service counts for us.
			// First level is tag, second level is status (passing, critical, etc.)
			countByTagAndStatus := make(map[string]map[string]uint)
			for _, entry := range serviceHealth {
				for _, tag := range entry.Service.Tags {
					if countByTagAndStatus[tag] == nil {
						countByTagAndStatus[tag] = make(map[string]uint)
					}
					for _, check := range entry.Checks {
						countByTagAndStatus[tag][check.Status]++
					}
				}
			}
			for tag, countByStatus := range countByTagAndStatus {
				for checkStatus, count := range countByStatus {
					metric := datadog.Metric{
						Metric: "consul.service.count",
						Points: []datadog.DataPoint{
							{
								float64(time.Now().Unix()),
								float64(count),
							},
						},
						Tags: []string{
							"status:" + checkStatus,
							"service:" + serviceName,
							tag,
						},
					}
					metrics = append(metrics, metric)
				}
			}
		}
		if err := datadogClient.PostMetrics(metrics); err != nil {
			log.Fatal(err)
		}
	}
}
