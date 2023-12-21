package collector

import (
	"context"
	"fmt"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metricdata"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"strconv"
	"sync"
)

func (c *CloudEyeCollector) collectMetricsByNamespace(ctx context.Context, ch chan<- prometheus.Metric, namespace string) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("fatal error occurred during collecting metrics: %s", err)
		}
	}()

	allMetrics, allResourcesInfo := c.getAllMetricsAndResourcesByNamespace(namespace)
	if len(allMetrics) == 0 {
		slog.Warn(fmt.Sprintf("[%s] Metrics of %s are not found, skip.", c.txnKey, namespace))
		return
	}

	slog.Debug(fmt.Sprintf("[%s] Start to scrape metric data", c.txnKey))
	workChan := make(chan struct{}, c.MaxRoutines)
	defer close(workChan)
	var wg sync.WaitGroup
	count := 0
	tmpMetrics := make([]metricdata.Metric, 0, c.ScrapeBatchSize)

	for _, metric := range allMetrics {
		count++
		tmpMetrics = append(tmpMetrics, metricsToMetricData(metric))
		if (len(tmpMetrics) == c.ScrapeBatchSize) || (count == len(allMetrics)) {
			workChan <- struct{}{}
			wg.Add(1)
			go func(tmpMetrics []metricdata.Metric) {
				defer func() {
					<-workChan
					wg.Done()
				}()

				slog.Debug(fmt.Sprintf("[%s] Start to getBatchMetricData, metric count: %d", c.txnKey, len(tmpMetrics)))
				dataList, err := c.getBatchMetricData(&tmpMetrics, c.From, c.To)
				if err != nil {
					return
				}
				c.pushMetricsData(ctx, ch, *dataList, allResourcesInfo)
			}(tmpMetrics)
			tmpMetrics = make([]metricdata.Metric, 0, c.ScrapeBatchSize)
		}
	}

	wg.Wait()
	slog.Debug(fmt.Sprintf("[%s] End to scrape all metric data", c.txnKey))
}

func (c *CloudEyeCollector) getAllMetricsAndResourcesByNamespace(namespace string) ([]metrics.Metric, map[string][]string) {
	allResourcesInfo, filterMetrics := c.getAllResources(namespace)
	slog.Debug(fmt.Sprintf("[%s] Resource number of %s: %d", c.txnKey, namespace, len(allResourcesInfo)))

	if len(*filterMetrics) > 0 {
		return *filterMetrics, allResourcesInfo
	}

	slog.Debug(fmt.Sprintf("[%s] Start to getAllMetric from CES", c.txnKey))
	allMetrics, err := c.getAllMetrics(namespace)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s] Get all metrics error: %s", c.txnKey, err.Error()))
		return nil, nil
	}
	slog.Debug(fmt.Sprintf("[%s] End to getAllMetric, Total number of of metrics: %d", c.txnKey, len(*allMetrics)))
	return *allMetrics, allResourcesInfo
}

func (c *CloudEyeCollector) getBatchMetricData(metrics *[]metricdata.Metric, from string, to string) (*[]metricdata.MetricData, error) {
	ifrom, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to Parse from: %s", err.Error()))
		return nil, err
	}
	ito, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to Parse to: %s", err.Error()))
		return nil, err
	}
	options := metricdata.BatchQueryOpts{
		Metrics: *metrics,
		From:    ifrom,
		To:      ito,
		Period:  "1",
		Filter:  "average",
	}

	client, err := c.Client.GetCESClient()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get ces client: %s", err.Error()))
		return nil, err
	}

	v, err := metricdata.BatchQuery(client, options).ExtractMetricDatas()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get metricdata: %s", err.Error()))
		return nil, err
	}

	return &v, nil
}

func (c *CloudEyeCollector) getAllMetrics(namespace string) (*[]metrics.Metric, error) {
	client, err := c.Client.GetCESClient()
	if err != nil {
		slog.Error(fmt.Sprintf("Get CES client error: %s", err.Error()))
		return nil, err
	}
	limit := 1000
	allPages, err := metrics.List(client, metrics.ListOpts{Namespace: namespace, Limit: &limit}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("Get all metric all pages error: %s", err.Error()))
		return nil, err
	}

	v, err := metrics.ExtractAllPagesMetrics(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Get all metric pages error: %s", err.Error()))
		return nil, err
	}

	return &v.Metrics, nil
}

func (c *CloudEyeCollector) pushMetricsData(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	dataList []metricdata.MetricData,
	allResourcesInfo map[string][]string,
) {
	for _, metric := range dataList {
		dataJson, err := validateMetricData(metric)
		if err != nil {
			slog.Error(fmt.Sprintf("[%s] Marshal metricData error: %s", c.txnKey, err.Error()))
		}
		slog.Debug(fmt.Sprintf("[%s] Get data points of metric are: %s", c.txnKey, string(dataJson)))

		data, err := getLatestData(metric.Datapoints)
		if err != nil {
			slog.Warn(fmt.Sprintf("[%s] Get data point error: %s, metric_name: %s, dimension: %+v", c.txnKey, err.Error(), metric.MetricName, metric.Dimensions))
			continue
		}

		labelInfo, err := c.relabelMetricData(allResourcesInfo, metric)
		if err != nil {
			slog.Error(fmt.Sprintf("[%s] %s", c.txnKey, err.Error()))
			continue
		}

		fqName := prometheus.BuildFQName(getMetricPrefixName(c.Prefix, metric.Namespace), labelInfo.PreResourceName, metric.MetricName)
		proMetric := prometheus.MustNewConstMetric(
			prometheus.NewDesc(fqName, fqName, labelInfo.Labels, nil),
			prometheus.GaugeValue, data, labelInfo.Values...)
		if err := pushMetricData(ctx, ch, proMetric); err != nil {
			slog.Error(fmt.Sprintf("[%s] Context has canceled, no need to send metric data, metric name: %s", c.txnKey, fqName))
		}
	}
}
