package consul2dogstats

import (
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/zorkian/go-datadog-api"
)

type Collector struct {
	datadogClient       datadogClient
	collectInterval     time.Duration
	lockKey             string
	lock                consulLock
	healthServiceFunc   func(service, tag string, passingOnly bool, q *consul.QueryOptions) ([]*consul.ServiceEntry, *consul.QueryMeta, error)
	catalogServicesFunc func(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error)
	agentSelfFunc       func() (map[string]map[string]interface{}, error)
}

func NewCollector(datadogClient datadogClient,
	consulClient *consul.Client,
	lockKey string,
	collectInterval time.Duration) (*Collector, error) {

	var err error

	c := new(Collector)
	c.healthServiceFunc = consulClient.Health().Service
	c.catalogServicesFunc = consulClient.Catalog().Services
	c.agentSelfFunc = consulClient.Agent().Self

	c.lock, err = consulClient.LockKey(lockKey)
	if err != nil {
		return nil, err
	}

	c.collectInterval = collectInterval
	c.lockKey = lockKey
	c.datadogClient = datadogClient

	return c, err
}

func (c *Collector) Run(stopCh <-chan struct{}, stoppedCh chan<- struct{}) error {
	defer func() {
		if stoppedCh != nil {
			close(stoppedCh)
		}
	}()

	for {
		sigCh := make(chan os.Signal)
		stopMainLoopCh := make(chan struct{})
		log.Infof("Attempting to acquire lock at %s", c.lockKey)
		lockLost, err := c.lock.Lock(nil)
		if err != nil {
			return err
		}
		defer c.lock.Unlock()
		log.Info("Lock acquired")

		go c.mainLoop(stopMainLoopCh, 0)

		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		select {
		case signal := <-sigCh:
			log.Infof("Received %s signal, terminating cleanly", signal)
			close(stopMainLoopCh)
			return nil
		case <-lockLost:
			log.Info("Lost Consul lock!  Stopping service poller")
			close(stopMainLoopCh)
			c.lock.Unlock()
			c.lock.Destroy()
			break
		case <-stopCh: // not normally closed, except in test cases
			close(stopMainLoopCh)
			return nil
		}
	}
}

func (c *Collector) mainLoop(stopLoopCh <-chan struct{}, stopAfterCount int) {
	queryCount := 0
	metricName := "consul.service.count"
	queryOptions := consul.QueryOptions{}

	agentInfo, err := c.agentSelfFunc()
	if err != nil {
		log.Fatal(err)
	}
	datacenter := agentInfo["Config"]["Datacenter"].(string)

	ticker := time.NewTicker(c.collectInterval)
	for {
		queryCount++
		if queryCount > 0 && queryCount > stopAfterCount {
			return
		}
		select {
		case <-stopLoopCh:
			return
		case <-ticker.C:
			// wait for next tick, then leave select loop
		}

		services, _, err := c.catalogServicesFunc(&queryOptions)
		if err != nil {
			log.Fatal(err)
		}

		var metrics []datadog.Metric

		for serviceName := range services {
			serviceHealth, _, err := c.healthServiceFunc(serviceName, "", false, &queryOptions)
			if err != nil {
				log.Fatal(err)
			}
			// Initialize the outer map that will be holding the service counts
			// for us. The key of the outer map is the union of tags (in
			// lexicographically sorted order, joined by the "|" character) for
			// a given consul.ServiceEntry.  The value is a map of service
			// statuses ("passing", "warning", "critical") to the count of each
			// status.
			countByTagsAndStatus := make(map[string]map[string]uint)
		ENTRY:
			for _, entry := range serviceHealth {
				tags := entry.Service.Tags
				sort.Strings(tags)
				joinedTags := strings.Join(tags, "|")

				// Initialize inner status map if necessary
				if countByTagsAndStatus[joinedTags] == nil {
					countByTagsAndStatus[joinedTags] = make(map[string]uint)
					for _, status := range []string{"critical", "warning", "passing"} {
						countByTagsAndStatus[joinedTags][status] = 0
					}
				}
				for _, check := range entry.Checks {
					// If any check returns critical, the status of the service is critical.
					if check.Status == "critical" {
						countByTagsAndStatus[joinedTags]["critical"]++
						continue ENTRY
					}
				}
				for _, check := range entry.Checks {
					// If any check returns warning, the status of the service is warning.
					if check.Status == "warning" {
						countByTagsAndStatus[joinedTags]["warning"]++
						continue ENTRY
					}
				}
				countByTagsAndStatus[joinedTags]["passing"]++
			}

			for joinedTags, countByStatus := range countByTagsAndStatus {
				for checkStatus, count := range countByStatus {
					tags := append(strings.Split(joinedTags, "|"),
						"status:"+checkStatus,
						"service:"+serviceName,
						"datacenter:"+datacenter)
					metric := datadog.Metric{
						Metric: &metricName,
						Points: []datadog.DataPoint{
							{
								float64(time.Now().Unix()),
								float64(count),
							},
						},
						Tags: tags,
					}
					metrics = append(metrics, metric)
				}
			}
		}
		if err := c.datadogClient.PostMetrics(metrics); err != nil {
			log.Fatal(err)
		}
	}
}
