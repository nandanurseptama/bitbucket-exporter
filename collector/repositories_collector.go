package collector

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type repositoriesCollector struct {
	workspaces []string
}

var (
	bitbucketTotalReposioriesCollector = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subSystemRepositories,
			Name:      "total",
			Help:      "Total of repositories",
		},
		[]string{"workspace", "project"},
	)
)

func (c *repositoriesCollector) Collect(ch chan<- prometheus.Metric) {
	bitbucketTotalReposioriesCollector.Collect(ch)
}

// Describe implements the prometheus.Collector interface.
func (p *repositoriesCollector) Describe(ch chan<- *prometheus.Desc) {
	bitbucketTotalReposioriesCollector.Describe(ch)
}

func (c *repositoriesCollector) Exec(
	ctx context.Context,
	instance *instance,
) error {
	page := 1
	for _, workspace := range c.workspaces {
		var params = map[string]string{"role": "member", "sort": "-created_on", "page": strconv.Itoa(page)}
		for {
			var respBody PaginationResponse[Repositories]
			err := instance.GET(ctx, fmt.Sprintf("%s/%s", repositoriesEndpoint, workspace), params, &respBody)

			if err != nil {
				return err
			}

			values := respBody.Values

			if len(values) < 1 {
				return nil
			}

			for _, v := range values {
				labels := []string{v.Workspace.Slug, v.Project.Key}
				bitbucketTotalReposioriesCollector.WithLabelValues(labels...).Inc()
			}

			if respBody.Next == nil {
				return nil
			}

			if *respBody.Next == "" {
				return nil
			}

			url, err := url.Parse(*respBody.Next)
			if err != nil {
				return err
			}

			nextPage := url.Query().Get("page")
			if nextPage == "" {
				return nil
			}

			_, err = strconv.Atoi(nextPage)

			if err != nil {
				return err
			}

			fmt.Println("nextPage :", nextPage)
			params["page"] = nextPage
		}
	}

	return nil
}
