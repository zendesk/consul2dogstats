package consul2dogstats

import (
	"fmt"
	"testing"
	"time"

	consul "github.com/hashicorp/consul/api"
)

// This mock catalog lists two services, "testService1" and "testService2",
// both of which have a single "test" tag, and run on separate nodes, and are
// passing on both.
func basicHealthService(service, tag string, passingOnly bool, q *consul.QueryOptions) ([]*consul.ServiceEntry, *consul.QueryMeta, error) {
	var serviceEntries []*consul.ServiceEntry
	switch service {
	case "testService1":
		var healthChecks []*consul.HealthCheck
		serviceEntry := new(consul.ServiceEntry)
		serviceEntry.Service = new(consul.AgentService)
		serviceEntry.Service.Service = "testService1"
		serviceEntry.Service.Tags = []string{"test"}
		serviceEntry.Checks = append(healthChecks, &consul.HealthCheck{Node: "testNode1", Status: "passing"})
		serviceEntries = append(serviceEntries, serviceEntry)
	case "testService2":
		var healthChecks []*consul.HealthCheck
		serviceEntry := new(consul.ServiceEntry)
		serviceEntry.Service = new(consul.AgentService)
		serviceEntry.Service.Service = "testService2"
		serviceEntry.Service.Tags = []string{"test"}
		serviceEntry.Checks = append(healthChecks, &consul.HealthCheck{Node: "testNode1", Status: "passing"})
		serviceEntries = append(serviceEntries, serviceEntry)
	default:
		return nil, nil, fmt.Errorf("Unknown service %s", service)
	}
	return serviceEntries, nil, nil
}

func basicCatalogServices(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	services := make(map[string][]string)
	services["testService1"] = make([]string, 0)
	services["testService2"] = make([]string, 0)
	return services, nil, nil
}

var basicTestCollectorConfig = testCollectorConfig{
	catalogServicesFunc: basicCatalogServices,
	healthServiceFunc:   basicHealthService,
}

func TestLockReleasedAfterRun(t *testing.T) {
	c, err := newTestCollector(&basicTestCollectorConfig)
	if err != nil {
		t.Fatal(err)
	}

	stopCh := make(chan struct{})
	stoppedCh := make(chan struct{})
	go c.Run(stopCh, stoppedCh)
	time.Sleep(time.Second)
	close(stopCh) // terminate loop
	<-stoppedCh   // wait for loop to exit cleanly

	if c.lock.(*testConsulLock).Locked() {
		t.Fatal("Consul lock was not released")
	}
}

func TestDatacenter(t *testing.T) {
	c, err := newTestCollector(&basicTestCollectorConfig)
	if err != nil {
		t.Fatal(err)
	}
	agentInfo, err := c.agentSelfFunc()
	if err != nil {
		t.Fatal(err)
	}
	datacenter := agentInfo["Config"]["Datacenter"].(string)
	if datacenter != "dc1" {
		t.Fatal("datacenter != dc1")
	}
}

func TestBasicServiceCatalog(t *testing.T) {
	c, err := newTestCollector(&basicTestCollectorConfig)
	if err != nil {
		t.Fatal(err)
	}
	services, _, err := c.catalogServicesFunc(&consul.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 {
		t.Fatal("Should be 2 services returned")
	}
	if _, ok := services["testService1"]; !ok {
		t.Fatal("testService1 does not exist")
	}
	if _, ok := services["testService2"]; !ok {
		t.Fatal("testService2 does not exist")
	}
}

func TestBasicServiceHealth(t *testing.T) {
	c, err := newTestCollector(&basicTestCollectorConfig)
	if err != nil {
		t.Fatal(err)
	}
	services, _, err := c.catalogServicesFunc(&consul.QueryOptions{})
	for serviceName := range services {
		serviceHealth, _, err := c.healthServiceFunc(serviceName, "", false, &consul.QueryOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range serviceHealth {
			if !stringInSlice("test", entry.Service.Tags) {
				t.Fatal("failed to find 'test' tag in service")
			}
		}
	}
}

func TestBasicSimpleServiceMetric(t *testing.T) {
	var foundService bool
	c, err := newTestCollector(&basicTestCollectorConfig)
	if err != nil {
		t.Fatal(err)
	}
	c.mainLoop(nil, 1)
	for _, metric := range c.datadogClient.(*testDatadogClient).metrics {
		if stringInSlice("service:testService1", metric.Tags) {
			foundService = true
			if !stringInSlice("test", metric.Tags) {
				t.Fatal("failed to find 'test' tag in metric")
			}
			if !stringInSlice("service:testService1", metric.Tags) {
				t.Fatal("failed to find 'service:testService1' tag in metric")
			}
		}
	}
	if !foundService {
		t.Fatal("failed to find 'service:testService1' tag in metric")
	}
	c.datadogClient.(*testDatadogClient).validateMetrics(t,
		[]string{"service:testService1"},
		&testStatusCounts{passing: 1, warning: 0, critical: 0})
}
