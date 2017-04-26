package consul2dogstats

import (
	"fmt"
	"testing"

	consul "github.com/hashicorp/consul/api"
)

// This mock catalog lists a single service that has a single "test" tag, but two
// checks whose statuses conflict with each other (one passing, one critical).
func multiCheckHealthService(service, tag string, passingOnly bool, q *consul.QueryOptions) ([]*consul.ServiceEntry, *consul.QueryMeta, error) {
	var serviceEntries []*consul.ServiceEntry
	switch service {
	case "testService1":
		var healthChecks []*consul.HealthCheck
		serviceEntry := new(consul.ServiceEntry)
		serviceEntry.Service = new(consul.AgentService)
		serviceEntry.Service.Service = "testService1"
		serviceEntry.Service.Tags = []string{"test"}
		serviceEntry.Checks = append(healthChecks, &consul.HealthCheck{Node: "testNode1", ServiceName: "testService1", Status: "passing"})
		serviceEntry.Checks = append(healthChecks, &consul.HealthCheck{Node: "testNode1", ServiceName: "testService1", Status: "critical"})
		serviceEntries = append(serviceEntries, serviceEntry)
	default:
		return nil, nil, fmt.Errorf("Unknown service %s", service)
	}
	return serviceEntries, nil, nil
}

func multiCheckCatalogServices(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	services := make(map[string][]string)
	services["testService1"] = make([]string, 0)
	return services, nil, nil
}

func TestMultiCheckMetric(t *testing.T) {
	var foundService bool
	c, err := newTestCollector(&testCollectorConfig{
		catalogServicesFunc: multiCheckCatalogServices,
		healthServiceFunc:   multiCheckHealthService,
	})
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
		[]string{},
		&testStatusCounts{passing: 0, warning: 0, critical: 1})
}
