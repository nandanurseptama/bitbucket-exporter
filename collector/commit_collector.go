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
	"slices"
	"strconv"
	"sync"

	"github.com/nandanurseptama/bitbucket-exporter/config"
	"github.com/nandanurseptama/bitbucket-exporter/helpers"
	"github.com/prometheus/client_golang/prometheus"
)

type repoCommitData struct {
	workspace string
	project   string
	repo      string
	total     uint64
}

type userCommitData struct {
	workspace string
	project   string
	repo      string
	nickname  string
	// nickname user
	total uint64
}

func (r *repoCommitData) Inc() {
	r.total = r.total + 1
}

type commitCollector struct {
	config                *config.CommitCollectorConfig
	commitRepoDataChannel <-chan Repository
	repoTotalCommit       DataHolder[map[string]*repoCommitData]
	userTotalCommit       DataHolder[map[string]*userCommitData]
}

func NewCommitCollector(
	config *config.CommitCollectorConfig,
	commitRepoDataChannel <-chan Repository,

) *commitCollector {
	return &commitCollector{
		config:                config,
		commitRepoDataChannel: commitRepoDataChannel,
		userTotalCommit: DataHolder[map[string]*userCommitData]{
			data: map[string]*userCommitData{},
		},
		repoTotalCommit: DataHolder[map[string]*repoCommitData]{
			data: map[string]*repoCommitData{},
		},
	}
}

var (
	repoCommitLabels = []string{"workspace", "project", "repository"}
	userCommitLabels = []string{"workspace", "project", "repository", "user"}

	repoTotalCommitDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemCommit,
			"total",
		),
		"Total commit of this repo",
		repoCommitLabels,
		nil,
	)

	userTotalCommitDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemMember,
			"total_commit",
		),
		"Total commit user",
		userCommitLabels,
		nil,
	)
)

// Collect implements the prometheus.Collector interface.
func (c *commitCollector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		c.userTotalCommit.Lock()
		defer c.userTotalCommit.Unlock()
		defer wg.Done()
		for _, v := range c.userTotalCommit.data {
			labels := []string{v.workspace, v.project, v.repo, v.nickname}
			ch <- prometheus.MustNewConstMetric(
				userTotalCommitDesc,
				prometheus.GaugeValue,
				float64(v.total),
				labels...,
			)
		}
	}()

	wg.Add(1)
	go func() {
		c.repoTotalCommit.Lock()
		defer c.repoTotalCommit.Unlock()
		defer wg.Done()
		for _, v := range c.repoTotalCommit.data {
			labels := []string{v.workspace, v.project, v.repo}
			ch <- prometheus.MustNewConstMetric(
				repoTotalCommitDesc,
				prometheus.GaugeValue,
				float64(v.total),
				labels...,
			)
		}
	}()

	wg.Wait()
}

// Describe implements the prometheus.Collector interface.
func (p *commitCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- repoTotalCommitDesc
	ch <- userTotalCommitDesc
}

func (c *commitCollector) Exec(ctx context.Context, instance *instance) error {
	if len(c.config.IncludedRepository) < 1 {
		return nil
	}

	for repo := range c.commitRepoDataChannel {
		if c.config == nil {
			continue
		}

		if !c.config.CollectTotalCommitRepo && !c.config.CollectTotalCommitUser {
			continue
		}
		first := c.config.IncludedRepository[0]
		if first == "*" && len(c.config.IncludedRepository) == 1 {
			go c.getTotalCommit(ctx, instance, repo)
			continue
		}

		i := slices.Index(
			c.config.IncludedRepository,
			fmt.Sprintf("%s/%s", repo.Workspace.Slug, repo.Slug),
		)

		if i < 0 {
			continue
		}

		go c.getTotalCommit(ctx, instance, repo)

	}
	return nil
}

func (c *commitCollector) getTotalCommit(
	ctx context.Context,
	instance *instance,
	repo Repository,
) error {
	page := 1
	for {
		endpoint := helpers.StrReplace(
			listCommitRepositoryEndpoint,
			map[string]string{":workspace_repo_slug": fmt.Sprintf("%s/%s", repo.Workspace.Slug, repo.Slug)},
		)
		var responseBody PaginationResponse[Commit]
		err := instance.GET(ctx, endpoint, map[string]string{"page": strconv.Itoa(page)}, &responseBody)

		if err != nil {
			return err
		}

		values := responseBody.Values
		if len(values) > 0 {
			// write total commit repo
			if c.config.CollectTotalCommitRepo {
				c.appendTotalCommitRepo(repo, responseBody.PageLen)
			}

			if c.config.CollectTotalCommitUser {
				c.appendTotalCommitUser(repo, values)
			}
		}

		if responseBody.Next == nil {
			return nil
		}

		if *responseBody.Next == "" {
			return nil
		}

		nextPageUrl, err := url.Parse(*responseBody.Next)

		if err != nil {
			return err
		}

		nextPageStr := nextPageUrl.Query().Get("page")
		if nextPageStr == "" {
			return nil
		}

		nextPage, err := strconv.Atoi(nextPageStr)
		if err != nil {
			return err
		}

		page = nextPage

	}
}
func (c *commitCollector) appendTotalCommitRepo(repo Repository, totalCommit uint64) {
	c.repoTotalCommit.Lock()
	var repoCommit = c.repoTotalCommit.data[repo.Uuid]
	if repoCommit == nil {
		repoCommit = &repoCommitData{
			workspace: repo.Workspace.Slug,
			project:   repo.Project.Key,
			repo:      repo.Slug,
			total:     totalCommit,
		}
	} else {
		repoCommit.total = repoCommit.total + totalCommit
	}
	c.repoTotalCommit.data[repo.Uuid] = repoCommit
	c.repoTotalCommit.Unlock()
}
func (c *commitCollector) appendTotalCommitUser(repo Repository, commits []Commit) {
	for _, commit := range commits {
		c.userTotalCommit.Lock()
		var userCommit = c.userTotalCommit.data[commit.Author.User.Uuid]
		if userCommit == nil {
			userCommit = &userCommitData{
				workspace: repo.Workspace.Slug,
				project:   repo.Project.Key,
				repo:      repo.Slug,
				nickname:  commit.Author.User.Nickname,
				total:     1,
			}
		} else {
			userCommit.total = userCommit.total + 1
		}

		c.userTotalCommit.data[commit.Author.User.Uuid] = userCommit
		c.userTotalCommit.Unlock()
	}
}
