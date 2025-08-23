package collector

import (
	"context"
	"errors"
	"fmt"
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
	if c.config == nil {
		return errors.New("refs_collector : config nil")
	}

	for v := range c.refsRepositoryDataChannel {
		var wg sync.WaitGroup

		if c.config.CollectTotalTag {
			wg.Add(1)
			go func() {
				defer wg.Done()
				totalTag, _ := c.getTags(ctx, v, instance)

				c.totalTagsHolder.Lock()
				c.totalTagsHolder.data = append(c.totalTagsHolder.data, refsData{
					workspace:  v.Workspace.Slug,
					project:    v.Project.Key,
					repository: v.Slug,
					total:      totalTag,
				})
				c.totalTagsHolder.Unlock()
			}()
		}

		if c.config.CollectTotalBranch {
			wg.Add(1)
			go func() {
				defer wg.Done()
				totalBranch, _ := c.getBranches(ctx, v, instance)
				c.totalBranchHolder.Lock()
				c.totalBranchHolder.data = append(c.totalBranchHolder.data, refsData{
					workspace:  v.Workspace.Slug,
					project:    v.Project.Key,
					repository: v.Slug,
					total:      totalBranch,
				})
				c.totalBranchHolder.Unlock()
			}()
		}
		if c.config.CollectTotalBranch || c.config.CollectTotalTag {
			wg.Wait()
		}
	}

	return nil
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
