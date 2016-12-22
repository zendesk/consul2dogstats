package main

import (
	"os"

	"time"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/ooyala/go-dogstatsd"
)

func main() {
	dogstatsdAddr := os.Getenv("DOGSTATSD_ADDR")
	if dogstatsdAddr == "" {
		dogstatsdAddr = "127.0.0.1:8125"
	}

	consulClient, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}
	health := consulClient.Health()
	catalog := consulClient.Catalog()

	statsd, err := dogstatsd.New(dogstatsdAddr)
	defer statsd.Close()
	if err != nil {
		log.Fatal(err)
	}
	statsd.Namespace = "consul."

	queryOptions := consul.QueryOptions{}

	ticker := time.NewTicker(5 * time.Second)

	for {
		<-ticker.C // wait for next tick
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
					statsd.Gauge("service."+serviceName, float64(count), datadogTags, 1)
				}
			}
		}
	}
}
