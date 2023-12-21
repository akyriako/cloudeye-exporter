package collector

import (
	"context"
	"fmt"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CloudEyeCollector struct {
	From            string
	To              string
	Namespaces      []string
	Prefix          string
	Client          *OpenTelekomCloudClient
	Region          string
	txnKey          string
	MaxRoutines     int
	ScrapeBatchSize int
}

func NewCloudEyeCollector(cloudConfig *config.CloudConfig, namespaces []string) (*CloudEyeCollector, error) {
	client, err := NewOpenTelekomCloudClient(cloudConfig)
	if err != nil {
		return nil, err
	}

	cloudEyeCollector := &CloudEyeCollector{
		Namespaces:      namespaces,
		Prefix:          cloudConfig.Global.Prefix,
		MaxRoutines:     cloudConfig.Global.MaxRoutines,
		Client:          client,
		ScrapeBatchSize: cloudConfig.Global.ScrapeBatchSize,
	}
	return cloudEyeCollector, nil
}

// Describe simply sends the two Descs in the struct to the channel.
func (c *CloudEyeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c *CloudEyeCollector) Collect(ch chan<- prometheus.Metric) {
	duration, err := time.ParseDuration("-10m")
	if err != nil {
		slog.Error(fmt.Sprintf("parse duration -10m error: %s", err.Error()))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()
	c.From = strconv.FormatInt(now.Add(duration).UnixNano()/1e6, 10)
	c.To = strconv.FormatInt(now.UnixNano()/1e6, 10)
	c.txnKey = fmt.Sprintf("%s-%s-%s", strings.Join(c.Namespaces, "-"), c.From, c.To)

	slog.Debug(fmt.Sprintf("[%s] start collecting data", c.txnKey))
	var wg sync.WaitGroup
	for _, namespace := range c.Namespaces {
		wg.Add(1)
		go func(ctx context.Context, ch chan<- prometheus.Metric, namespace string) {
			defer wg.Done()
			c.collectMetricsByNamespace(ctx, ch, namespace)
		}(ctx, ch, namespace)
	}
	wg.Wait()
	slog.Debug(fmt.Sprintf("[%s] end collecting data", c.txnKey))
}
