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
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/nandanurseptama/bitbucket-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

type refsData struct {
	workspace  string
	project    string
	repository string
	total      uint64
}

var (
	repositoryRefsLabels      = []string{"workspace", "project", "repository"}
	repositoryRefsTotalBranch = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemRepoRefs,
			"total_branch",
		),
		"Total branch of this repo",
		repositoryRefsLabels,
		nil,
	)
	repositoryRefsTotalTag = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemRepoRefs,
			"total_tag",
		),
		"Total tag of this repo",
		repositoryRefsLabels,
		nil,
	)
)

type refsCollector struct {
	config                    *config.RefsCollectorConfig
	refsRepositoryDataChannel <-chan Repository
	// channel to passing refs data
	//refsBranchDataChannel chan<- Refs

	totalTagsHolder   DataHolder[[]refsData]
	totalBranchHolder DataHolder[[]refsData]
}

func NewRefsCollector(config *config.RefsCollectorConfig, refsRepositoryDataChannel <-chan Repository) *refsCollector {
	return &refsCollector{
		config:                    config,
		refsRepositoryDataChannel: refsRepositoryDataChannel,
		totalTagsHolder: DataHolder[[]refsData]{
			data: []refsData{},
		},
		totalBranchHolder: DataHolder[[]refsData]{
			data: []refsData{},
		},
	}
}

// Collect implements the prometheus.Collector interface.
func (c *refsCollector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		c.totalBranchHolder.Lock()
		defer c.totalBranchHolder.Unlock()
		defer wg.Done()
		for _, v := range c.totalBranchHolder.data {
			labels := []string{v.workspace, v.project, v.repository}
			ch <- prometheus.MustNewConstMetric(
				repositoryRefsTotalBranch,
				prometheus.GaugeValue,
				float64(v.total),
				labels...,
			)
		}
	}()

	wg.Add(1)
	go func() {
		c.totalTagsHolder.Lock()
		defer c.totalTagsHolder.Unlock()
		defer wg.Done()
		for _, v := range c.totalTagsHolder.data {
			labels := []string{v.workspace, v.project, v.repository}
			ch <- prometheus.MustNewConstMetric(
				repositoryRefsTotalTag,
				prometheus.GaugeValue,
				float64(v.total),
				labels...,
			)
		}
	}()

	wg.Wait()
}

// Describe implements the prometheus.Collector interface.
func (p *refsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- repositoryRefsTotalTag
	ch <- repositoryRefsTotalBranch
}
func (c *refsCollector) Exec(ctx context.Context, instance *instance) error {

	for repo := range c.refsRepositoryDataChannel {
		if c.config == nil {
			return errors.New("refs_collector : config nil")
		}

		if len(c.config.IncludedRepository) < 1 {
			return nil
		}

		if !c.config.CollectTotalBranch && !c.config.CollectTotalTag {
			continue
		}

		first := c.config.IncludedRepository[0]
		if first == "*" && len(c.config.IncludedRepository) == 1 {
			go c.collectRefs(ctx, repo, instance)
			continue
		}
		i := slices.Index(
			c.config.IncludedRepository,
			fmt.Sprintf("%s/%s", repo.Workspace.Slug, repo.Slug),
		)

		if i < 0 {
			continue
		}

		go c.collectRefs(ctx, repo, instance)
	}

	return nil
}

func (c *refsCollector) collectRefs(ctx context.Context, repo Repository, instance *instance) {
	var wg sync.WaitGroup

	if c.config.CollectTotalTag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			totalTag, _ := c.getTags(ctx, repo, instance)

			c.totalTagsHolder.Lock()
			c.totalTagsHolder.data = append(c.totalTagsHolder.data, refsData{
				workspace:  repo.Workspace.Slug,
				project:    repo.Project.Key,
				repository: repo.Slug,
				total:      totalTag,
			})
			c.totalTagsHolder.Unlock()
		}()
	}

	if c.config.CollectTotalBranch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			totalBranch, _ := c.getBranches(ctx, repo, instance)
			c.totalBranchHolder.Lock()
			c.totalBranchHolder.data = append(c.totalBranchHolder.data, refsData{
				workspace:  repo.Workspace.Slug,
				project:    repo.Project.Key,
				repository: repo.Slug,
				total:      totalBranch,
			})
			c.totalBranchHolder.Unlock()
		}()
	}
	if c.config.CollectTotalBranch || c.config.CollectTotalTag {
		wg.Wait()
	}
}

func (c *refsCollector) getTags(ctx context.Context, repo Repository, instance *instance) (uint64, error) {
	return c.getRefs(ctx, repo.Workspace.Slug, repo.Slug, "tag", instance)
}
func (c *refsCollector) getBranches(ctx context.Context, repo Repository, instance *instance) (uint64, error) {
	return c.getRefs(ctx, repo.Workspace.Slug, repo.Slug, "branch", instance)
}

func (c *refsCollector) getRefs(
	ctx context.Context,
	workspace, repo, refType string,
	instance *instance,
) (uint64, error) {
	endpoint := strings.ReplaceAll(refsRepositoryEndpoint, ":workspace", workspace)
	endpoint = strings.ReplaceAll(endpoint, ":repo_slug", repo)
	params := map[string]string{"q": fmt.Sprintf("type=\"%s\"", refType)}
	var respBody PaginationResponse[Refs]

	err := instance.GET(ctx, endpoint, params, &respBody)

	if err != nil {
		return 0, err
	}

	return respBody.Size, nil
}
