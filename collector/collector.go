package collector

import (
	"context"
	"fmt"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metrics"
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
		slog.Error(fmt.Sprintf("ParseDuration -10m error: %s", err.Error()))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()
	c.From = strconv.FormatInt(now.Add(duration).UnixNano()/1e6, 10)
	c.To = strconv.FormatInt(now.UnixNano()/1e6, 10)
	c.txnKey = fmt.Sprintf("%s-%s-%s", strings.Join(c.Namespaces, "-"), c.From, c.To)

	slog.Debug(fmt.Sprintf("[%s] Start to collect data", c.txnKey))
	var wg sync.WaitGroup
	for _, namespace := range c.Namespaces {
		wg.Add(1)
		go func(ctx context.Context, ch chan<- prometheus.Metric, namespace string) {
			defer wg.Done()
			c.collectMetricByNamespace(ctx, ch, namespace)
		}(ctx, ch, namespace)
	}
	wg.Wait()
	slog.Debug(fmt.Sprintf("[%s] End to collect data", c.txnKey))
}

func (c *CloudEyeCollector) getElbResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	elbInfo.Lock()
	defer elbInfo.Unlock()
	if elbInfo.Info == nil || time.Now().Unix() > elbInfo.TTL {
		allELBs, err := c.Client.getAllLoadBalancers()
		if err != nil {
			slog.Error(fmt.Sprintf("Get all LoadBalancer error: %s", err.Error()))
			return elbInfo.Info, &elbInfo.FilterMetrics
		}
		if allELBs == nil {
			return elbInfo.Info, &elbInfo.FilterMetrics
		}
		configMap := config.GetMetricFilters("SYS.ELB")
		for _, elb := range *allELBs {
			resourceInfos[elb.ID] = []string{elb.Name, elb.Provider, elb.VipAddress}
			if configMap == nil {
				continue
			}
			if metricNames, ok := configMap["lbaas_instance_id"]; ok {
				filterMetrics = append(filterMetrics, buildSingleDimensionMetrics(metricNames, "SYS.ELB", "lbaas_instance_id", elb.ID)...)
			}
			if metricNames, ok := configMap["lbaas_instance_id,lbaas_listener_id"]; ok {
				filterMetrics = append(filterMetrics, buildListenerMetrics(metricNames, &elb)...)
			}
			if metricNames, ok := configMap["lbaas_instance_id,lbaas_pool_id"]; ok {
				filterMetrics = append(filterMetrics, buildPoolMetrics(metricNames, &elb)...)
			}
		}

		allListeners, err := c.Client.getAllListeners()
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Listener error: %s", err.Error()))
		}
		if allListeners != nil {
			for _, listener := range *allListeners {
				resourceInfos[listener.ID] = []string{listener.Name, fmt.Sprintf("%d", listener.ProtocolPort)}
			}
		}

		elbInfo.Info = resourceInfos
		elbInfo.FilterMetrics = filterMetrics
		elbInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return elbInfo.Info, &elbInfo.FilterMetrics
}
