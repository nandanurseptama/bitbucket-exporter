// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/nandanurseptama/bitbucket-exporter/helpers"
	"github.com/prometheus/client_golang/prometheus"
)

var repoLabels = []string{
	"workspace",
	"project",
	"name",
	"language",
	"has_issues",
	"has_wiki",
	"is_private",
}
var (
	repoInfoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemRepositories,
			"info",
		),
		"Information about a Bitbucket repo",
		repoLabels,
		nil,
	)
	repoCreatedOnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemRepositories,
			"created_on",
		),
		"Timestamp of creation of repo",
		repoLabels,
		nil,
	)
	repoUpdatedOnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemRepositories,
			"updated_on",
		),
		"Timestamp of the last modification of repo",
		repoLabels,
		nil,
	)
	repoSizeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemRepositories,
			"size",
		),
		"Size of repo in KB",
		repoLabels,
		nil,
	)
)

type repositoriesCollector struct {
	workspaces                []string
	holders                   *DataHolder[[]Repository]
	repositoryRefsDataChannel chan<- Repository
	commitRepoDataChannel     chan<- Repository
}

func NewRepositoriesCollector(
	workspaces []string,
	repositoryRefsDataChannel chan<- Repository,
	commitRepoDataChannel chan<- Repository,
) *repositoriesCollector {
	return &repositoriesCollector{
		workspaces:                workspaces,
		holders:                   &DataHolder[[]Repository]{},
		repositoryRefsDataChannel: repositoryRefsDataChannel,
		commitRepoDataChannel:     commitRepoDataChannel,
	}
}

// Collect implements the prometheus.Collector interface.
func (c *repositoriesCollector) Collect(ch chan<- prometheus.Metric) {
	c.holders.Lock()
	defer c.holders.Unlock()

	for _, v := range c.holders.data {
		labels := []string{
			v.Workspace.Slug,
			v.Project.Key,
			v.Slug,
			v.Language,
			helpers.BoolToString(v.HasIssues),
			helpers.BoolToString(v.HasWiki),
			helpers.BoolToString(v.IsPrivate),
		}
		ch <- prometheus.MustNewConstMetric(
			repoInfoDesc,
			prometheus.GaugeValue,
			1,
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			repoCreatedOnDesc,
			prometheus.GaugeValue,
			float64(v.CreatedOn.Unix()),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			repoUpdatedOnDesc,
			prometheus.GaugeValue,
			float64(v.UpdatedOn.Unix()),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			repoSizeDesc,
			prometheus.GaugeValue,
			float64(v.Size),
			labels...,
		)
	}
}

// Describe implements the prometheus.Collector interface.
func (p *repositoriesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- repoInfoDesc
	ch <- repoCreatedOnDesc
	ch <- repoUpdatedOnDesc
	ch <- repoSizeDesc

}

func (c *repositoriesCollector) Exec(
	ctx context.Context,
	instance *instance,
) error {
	page := 1
	for _, workspace := range c.workspaces {
		for {
			var params = map[string]string{"role": "member", "sort": "-created_on", "page": strconv.Itoa(page)}

			var respBody PaginationResponse[Repository]
			err := instance.GET(ctx, fmt.Sprintf("%s/%s", repositoriesEndpoint, workspace), params, &respBody)

			if err != nil {
				return err
			}

			values := respBody.Values
			// add to data holder
			if len(values) > 0 {
				c.holders.Lock()
				c.holders.data = append(c.holders.data, values...)
				c.holders.Unlock()
			}

			fmt.Println("passing to refs collector")
			// send to refs data channel
			for _, v := range values {
				c.repositoryRefsDataChannel <- v
				c.commitRepoDataChannel <- v
			}
			fmt.Println("passing to other done")

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

			nextPageInt, err := strconv.Atoi(nextPage)

			if err != nil {
				return err
			}

			fmt.Println("nextPage :", nextPage)
			page = nextPageInt

			time.Sleep(1 * time.Second)
		}
	}

	return nil
}
