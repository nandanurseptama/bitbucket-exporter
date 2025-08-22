package collector

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type memberData struct {
	sync.Mutex
	// mapping of workspace to total member
	total map[string]uint64
}

type memberCollector struct {
	workspaces []string
	holders    *DataHolder[memberData]
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
		holders: &DataHolder[memberData]{
			data: memberData{
				total: map[string]uint64{},
			},
		},
	}
}

func (c *memberCollector) Collect(ch chan<- prometheus.Metric) {
	c.holders.Lock()
	defer c.holders.Unlock()
	for k, v := range c.holders.data.total {
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
		c.holders.data.total[workspace] = responsebody.Size
		c.holders.Unlock()

		time.Sleep(time.Second * 5)
	}

	return nil
}
