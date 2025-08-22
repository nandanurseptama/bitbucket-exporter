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
	"log/slog"
	"sync"
	"time"

	"github.com/nandanurseptama/bitbucket-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	scrapeDurationOpts = prometheus.Opts{
		Namespace:   namespace,
		Subsystem:   "scrape",
		Name:        "collector_duration_seconds",
		Help:        "bitbucket_exporter: Duration of a collector scrape.",
		ConstLabels: nil,
	}
	scrapeSuccessOpts = prometheus.Opts{
		Namespace:   namespace,
		Subsystem:   "scrape",
		Name:        "collector_success",
		Help:        "bitbucket_exporter:  Whether a collector succeeded.",
		ConstLabels: nil,
	}

	scrapeDurationGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: scrapeDurationOpts.Namespace,
			Subsystem: scrapeDurationOpts.Subsystem,
			Name:      scrapeDurationOpts.Name,
		},
		[]string{"collector"},
	)
	scrapeSuccessGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: scrapeSuccessOpts.Namespace,
			Subsystem: scrapeSuccessOpts.Subsystem,
			Name:      scrapeSuccessOpts.Name,
		},
		[]string{"collector"},
	)
)

type BitbucketCollector struct {
	logger     *slog.Logger
	instance   *instance
	collectors map[string]Collector
}

type Collector interface {
	prometheus.Collector

	// collect metrics at background
	Exec(ctx context.Context, instance *instance) error
}

func NewBitbucketCollector(
	logger *slog.Logger,
	config *config.Config,
) *BitbucketCollector {
	return &BitbucketCollector{
		instance: newInstance(config.Auth),
		logger:   logger,
		collectors: map[string]Collector{
			keyRepositoriesCollector: NewRepositoriesCollector(config.IncludedWorkspace),
			keyMemberCollector:       NewMemberCollector(config.IncludedWorkspace),
		},
	}
}

type mainCollector struct {
}

func (c *mainCollector) Collect(ch chan<- prometheus.Metric) {
	scrapeDurationGaugeVec.Collect(ch)
	scrapeSuccessGaugeVec.Collect(ch)
}

// Describe implements the prometheus.Collector interface.
func (p *mainCollector) Describe(ch chan<- *prometheus.Desc) {
	scrapeDurationGaugeVec.Describe(ch)
	scrapeSuccessGaugeVec.Describe(ch)
}

// Get all collectors
func (c *BitbucketCollector) GetCollectors() []prometheus.Collector {
	var collectors []prometheus.Collector
	collectors = append(collectors, &mainCollector{})
	for _, v := range c.collectors {
		collectors = append(collectors, v)
	}
	return collectors
}

// collect bitbucket data at background
func (c *BitbucketCollector) Exec(ctx context.Context) {
	var wg sync.WaitGroup

	for name, collector := range c.collectors {
		wg.Add(1)
		go func(name string, collector Collector) {
			defer wg.Done()
			execute(ctx, name, collector, c.instance, c.logger)
		}(name, collector)
	}

	// wait until all collectors finish
	wg.Wait()

	// wait until context canceled
	<-ctx.Done()
}

func execute(
	ctx context.Context, name string, c Collector, instance *instance, logger *slog.Logger,
) {
	begin := time.Now()
	err := c.Exec(ctx, instance)
	duration := time.Since(begin)
	var success float64
	if err != nil {
		logger.Error("collector failed", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		success = 0
	} else {
		logger.Debug("collector succeeded", "name", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	scrapeDurationGaugeVec.WithLabelValues(name).Add(duration.Seconds())
	scrapeSuccessGaugeVec.WithLabelValues(name).Set(success)
}
