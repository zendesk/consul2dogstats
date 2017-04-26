package consul2dogstats

import (
	"sync"
	"testing"

	consul "github.com/hashicorp/consul/api"
	"github.com/zorkian/go-datadog-api"
)

type testCollectorConfig struct {
	// Function having the same signature as https://godoc.org/github.com/hashicorp/consul/api#Catalog.Services
	catalogServicesFunc func(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error)
	// Function having the same signature as https://godoc.org/github.com/hashicorp/consul/api#Health.Service
	healthServiceFunc func(service, tag string, passingOnly bool, q *consul.QueryOptions) ([]*consul.ServiceEntry, *consul.QueryMeta, error)
}

type testDatadogClient struct {
	// Array of metrics that we otherwise would have posted to the Datadog API endpoint
	metrics []datadog.Metric
}

type testConsulClient struct{}
type testConsulHealth struct{}
type testConsulCatalog struct{}
type testConsulAgent struct{}

// Mocks a Consul lock
type testConsulLock struct {
	lockPath   string
	mtx        *sync.Mutex
	locked     bool
	lockLostCh chan struct{}
}

// lockKey returns a Consul lock mock object.  The lockKey value is
// a Consul key path (which we ignore).
func lockKey(lockKey string) (*testConsulLock, error) {
	lock := new(testConsulLock)
	lock.mtx = new(sync.Mutex)
	return lock, nil
}

// Lock locks the mock Consul Lock.
func (l *testConsulLock) Lock(stopCh <-chan struct{}) (<-chan struct{}, error) {
	l.mtx.Lock()
	l.locked = true
	ch := make(chan struct{})
	l.lockLostCh = ch
	return ch, nil
}

// LoseLock forces the mock Consul lock to be lost.
func (l *testConsulLock) LoseLock() {
	close(l.lockLostCh)
	return
}

// Unlock unlocks the mock Consul lock.
func (l *testConsulLock) Unlock() error {
	l.mtx.Unlock()
	l.locked = false
	return nil
}

// Destroy is a mock Consul lock method that does nothing; we provide it here
// only to satisfy the interface.
func (l *testConsulLock) Destroy() error {
	return nil
}

// Locked returns true IFF the mock Consul lock is locked.
func (l *testConsulLock) Locked() bool {
	return l.locked
}

// PostMetrics posts the given Metrics to our mock Datadog API client.
func (c *testDatadogClient) PostMetrics(metrics []datadog.Metric) error {
	for _, metric := range metrics {
		c.metrics = append(c.metrics, metric)
	}
	return nil
}

// basicAgentSelf mocks https://godoc.org/github.com/hashicorp/consul/api#Agent.Self
func basicAgentSelf() (map[string]map[string]interface{}, error) {
	info := make(map[string]map[string]interface{})
	info["Config"] = make(map[string]interface{})
	info["Config"]["Datacenter"] = "dc1"
	return info, nil
}

// newTestCollector returns a mock Collector object.  If provided a
// pointer to a testCollectorConfig, the mocked Consul functions in it will
// be called to collect the data, and posted to our mock Datadog client.
func newTestCollector(cfg *testCollectorConfig) (*Collector, error) {
	c := new(Collector)
	c.agentSelfFunc = basicAgentSelf

	if cfg == nil {
		c.healthServiceFunc = basicHealthService
		c.catalogServicesFunc = basicCatalogServices
	} else {
		c.healthServiceFunc = cfg.healthServiceFunc
		c.catalogServicesFunc = cfg.catalogServicesFunc
	}

	c.datadogClient = new(testDatadogClient)
	c.collectInterval = 1
	c.lockKey = "consul2dogstats/test_lock"
	c.lock, _ = lockKey(c.lockKey)

	return c, nil
}

// stringInSlice finds the needle string in the haystack array.
func stringInSlice(needle string, haystack []string) bool {
	for _, elem := range haystack {
		if elem == needle {
			return true
		}
	}
	return false
}

// test cases calling validateMetrics will create one of these and pass it in
type testStatusCounts struct {
	passing, warning, critical int
}

// validateMetrics ensures that the metrics associated with the given list of tags,
// and that have been posted to the mock Datadog client, match the counts provided.
// The counts are expressed in terms of a testStatusCounts struct, the pointer to
// which must be provided as well.
func (c *testDatadogClient) validateMetrics(t *testing.T,
	tags []string, // all tags must match
	wanted *testStatusCounts) {

	var passingCount, warningCount, criticalCount int

	for _, metric := range c.metrics {
		var allTagsPresent = true
		for _, tag := range tags {
			if !stringInSlice(tag, metric.Tags) {
				allTagsPresent = false
			}
		}
		if !allTagsPresent {
			continue
		}
		if stringInSlice("status:passing", metric.Tags) {
			for _, point := range metric.Points {
				passingCount += int(point[1])
			}
		}
		if stringInSlice("status:warning", metric.Tags) {
			for _, point := range metric.Points {
				warningCount += int(point[1])
			}
		}
		if stringInSlice("status:critical", metric.Tags) {
			for _, point := range metric.Points {
				criticalCount += int(point[1])
			}
		}
	}
	if passingCount != wanted.passing {
		t.Fatalf("expected passing count to be %d instead of %d", wanted.passing, passingCount)
	}
	if warningCount != wanted.warning {
		t.Fatalf("expected warning count to be %d instead of %d", wanted.warning, warningCount)
	}
	if criticalCount != wanted.critical {
		t.Fatalf("expected critical count to be %d instead of %d", wanted.critical, criticalCount)
	}
}
