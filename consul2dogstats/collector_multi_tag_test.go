package consul2dogstats

import (
	"fmt"
	"testing"

	consul "github.com/hashicorp/consul/api"
)

// This mock catalog lists two instances of a single service, "testService1",
// which runs on two nodes, but has different tags on each.
func multiTagHealthService(service, tag string, passingOnly bool, q *consul.QueryOptions) ([]*consul.ServiceEntry, *consul.QueryMeta, error) {
	var serviceEntries []*consul.ServiceEntry
	switch service {
	case "testService1":
		var healthChecks []*consul.HealthCheck
		serviceEntry1 := new(consul.ServiceEntry)
		serviceEntry1.Service = new(consul.AgentService)
		serviceEntry1.Service.Service = "testService1"
		serviceEntry1.Service.Tags = []string{"environment:production", "shard:1"}
		serviceEntry1.Checks = append(healthChecks, &consul.HealthCheck{Node: "testNode1", ServiceName: "testService1", Status: "passing"})
		serviceEntries = append(serviceEntries, serviceEntry1)

		serviceEntry2 := new(consul.ServiceEntry)
		serviceEntry2.Service = new(consul.AgentService)
		serviceEntry2.Service.Service = "testService1"
		serviceEntry2.Service.Tags = []string{"environment:production", "shard:2"}
		serviceEntry2.Checks = append(healthChecks, &consul.HealthCheck{Node: "testNode2", ServiceName: "testService1", Status: "passing"})
		serviceEntries = append(serviceEntries, serviceEntry2)

	default:
		return nil, nil, fmt.Errorf("Unknown service %s", service)
	}
	return serviceEntries, nil, nil
}

func multiTagCatalogServices(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	services := make(map[string][]string)
	services["testService1"] = make([]string, 0)
	return services, nil, nil
}

func TestMultiTagMetric(t *testing.T) {
	c, err := newTestCollector(&testCollectorConfig{
		catalogServicesFunc: multiTagCatalogServices,
		healthServiceFunc:   multiTagHealthService,
	})
	if err != nil {
		t.Fatal(err)
	}
	c.mainLoop(nil, 1)

	tagsFound := make(map[string]bool)
	for _, metric := range c.datadogClient.(*testDatadogClient).metrics {
		for _, tag := range metric.Tags {
			tagsFound[tag] = true
		}
	}
	for _, tag := range []string{"service:testService1", "environment:production", "shard:1", "shard:2"} {
		if _, ok := tagsFound[tag]; !ok {
			t.Fatalf("failed to find '%s' tag in metric", tag)
		}
	}
	c.datadogClient.(*testDatadogClient).validateMetrics(t,
		[]string{"environment:production"},
		&testStatusCounts{passing: 2, warning: 0, critical: 0})
	c.datadogClient.(*testDatadogClient).validateMetrics(t,
		[]string{"shard:1"},
		&testStatusCounts{passing: 1, warning: 0, critical: 0})
}
