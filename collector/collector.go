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
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"bitbucket_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"bitbucket_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

type BitbucketCollector struct {
	logger     *slog.Logger
	instance   *instance
	collectors map[string]Collector
}

type Collector interface {
	Exec(ctx context.Context, instance *instance, ch chan<- prometheus.Metric) error
}

func NewBitbucketCollector(logger *slog.Logger, authConfig *config.AuthConfig) *BitbucketCollector {
	return &BitbucketCollector{
		instance: newInstance(authConfig),
		logger:   logger,
		collectors: map[string]Collector{
			keyRepositoriesCollector: &repositoriesCollector{},
		},
	}
}

// Describe implements the prometheus.Collector interface.
func (p BitbucketCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (p BitbucketCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()

	wg := sync.WaitGroup{}
	wg.Add(len(p.collectors))
	for name, c := range p.collectors {
		go func(name string, c Collector) {
			execute(ctx, name, c, p.instance, ch, p.logger)

			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(ctx context.Context, name string, c Collector, instance *instance, ch chan<- prometheus.Metric, logger *slog.Logger) {
	begin := time.Now()
	err := c.Exec(ctx, instance, ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		logger.Error("collector failed", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		success = 0
	} else {
		logger.Debug("collector succeeded", "name", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}
