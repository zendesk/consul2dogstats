package consul2dogstats

import (
	"github.com/zorkian/go-datadog-api"
)

type datadogClient interface {
	PostMetrics(series []datadog.Metric) error
}
