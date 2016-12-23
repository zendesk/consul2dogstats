package main

import (
	"os"
	"syscall"

	"time"

	"os/signal"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/ooyala/go-dogstatsd"
)

func main() {
	dogstatsdAddr := os.Getenv("DOGSTATSD_ADDR")
	if dogstatsdAddr == "" {
		dogstatsdAddr = "127.0.0.1:8125"
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

	statsdClient, err := dogstatsd.New(dogstatsdAddr)
	defer statsdClient.Close()
	if err != nil {
		log.Fatal(err)
	}
	statsdClient.Namespace = "consul."

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

		go mainLoop(consulClient, statsdClient, collectInterval, doneCh)

		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		select {
		case signal := <-sigCh:
			log.Infof("Received signal %v, terminating cleanly", signal)
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

func mainLoop(consulClient *consul.Client, statsdClient *dogstatsd.Client, interval time.Duration, doneCh <-chan struct{}) {
	health := consulClient.Health()
	catalog := consulClient.Catalog()
	queryOptions := consul.QueryOptions{}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			// wait for next tick, then leave select loop
		case <-doneCh:
			return
		}
		services, _, err := catalog.Services(&queryOptions)
		if err != nil {
			log.Fatal(err)
		}
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
					datadogTags := []string{
						"status:" + checkStatus,
						tag,
					}
					statsdClient.Gauge("service."+serviceName, float64(count), datadogTags, 1)
				}
			}
		}
	}
}
