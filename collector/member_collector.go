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
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type memberCollector struct {
	workspaces []string
	holders    *DataHolder[map[string]uint64]
}

var (
	memberLabels             = []string{"workspace"}
	bitbucketTotalMemberDesc = prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			subSystemMember,
			"total",
		),
		"Total of member inside the workspace",
		memberLabels,
		nil,
	)
)

func NewMemberCollector(workspaces []string) *memberCollector {
	return &memberCollector{
		workspaces: workspaces,
		holders: &DataHolder[map[string]uint64]{
			data: map[string]uint64{},
		},
	}
}

func (c *memberCollector) Collect(ch chan<- prometheus.Metric) {
	c.holders.Lock()
	defer c.holders.Unlock()
	for k, v := range c.holders.data {
		labels := []string{k}
		ch <- prometheus.MustNewConstMetric(
			bitbucketTotalMemberDesc,
			prometheus.CounterValue,
			float64(v),
			labels...,
		)
	}
}

func (c *memberCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- bitbucketTotalMemberDesc
}

func (c *memberCollector) Exec(ctx context.Context, instance *instance) error {
	for _, workspace := range c.workspaces {

		var responsebody PaginationResponse[any]
		endpoint := strings.ReplaceAll(workspaceMembersEndpoint, ":workspace", workspace)
		err := instance.GET(ctx, endpoint, map[string]string{}, &responsebody)

		if err != nil {
			return err
		}
		c.holders.Lock()
		c.holders.data[workspace] = responsebody.Size
		c.holders.Unlock()

		time.Sleep(time.Second * 5)
	}

	return nil
}
