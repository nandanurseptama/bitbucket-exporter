package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type repositoriesCollector struct {
}

var (
	bitbucketTotalReposioriesCollector = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subSystemRepositories,
			Name:      "total",
			Help:      "Total of repositories",
		},

		[]string{"workspace", "project"},
	)

	endpoint = "repositories"
)

func (c *repositoriesCollector) Exec(ctx context.Context, instance *instance, ch chan<- prometheus.Metric) error {
	var respBody PaginationResponse[Repositories]
	err := instance.GET(ctx, endpoint, map[string]string{"role": "member"}, &respBody)

	if err != nil {
		return err
	}

	values := respBody.Values

	if len(values) < 1 {
		return nil
	}

	for _, v := range values {

		labels := []string{v.Workspace.Slug, v.Project.Key}
		bitbucketTotalReposioriesCollector.WithLabelValues(labels...).Add(1)
	}

	bitbucketTotalReposioriesCollector.Collect(ch)

	return nil
}
