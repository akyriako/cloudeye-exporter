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
		slog.Warn(fmt.Sprintf("[%s] metrics of %s were not found", c.txnKey, namespace))
		return
	}

	slog.Debug(fmt.Sprintf("[%s] scraping metric data", c.txnKey))
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

				slog.Debug(fmt.Sprintf("[%s] getting batch metric data, metric count: %d", c.txnKey, len(tmpMetrics)))
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
	slog.Debug(fmt.Sprintf("[%s] scraped all metric data", c.txnKey))
}

func (c *CloudEyeCollector) getAllMetricsAndResourcesByNamespace(namespace string) ([]metrics.Metric, map[string][]string) {
	allResourcesInfo, filterMetrics := c.getAllResources(namespace)
	slog.Debug(fmt.Sprintf("[%s] found %d resources in %s: ", c.txnKey, len(allResourcesInfo), namespace))

	if len(*filterMetrics) > 0 {
		return *filterMetrics, allResourcesInfo
	}

	slog.Debug(fmt.Sprintf("[%s] collecting all metrics from CES", c.txnKey))
	allMetrics, err := c.getAllMetrics(namespace)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s] collecting all metrics failed: %s", c.txnKey, err.Error()))
		return nil, nil
	}
	slog.Debug(fmt.Sprintf("[%s] number of collected metrics: %d", c.txnKey, len(*allMetrics)))
	return *allMetrics, allResourcesInfo
}

func (c *CloudEyeCollector) getBatchMetricData(metrics *[]metricdata.Metric, from string, to string) (*[]metricdata.MetricData, error) {
	ifrom, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		slog.Error(fmt.Sprintf("parse failed: %s", err.Error()))
		return nil, err
	}
	ito, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		slog.Error(fmt.Sprintf("parse failed: %s", err.Error()))
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
		slog.Error(fmt.Sprintf("acquiring a CES client failed: %s", err.Error()))
		return nil, err
	}

	v, err := metricdata.BatchQuery(client, options).ExtractMetricDatas()
	if err != nil {
		slog.Error(fmt.Sprintf("collecting metricdata from batch query failed: %s", err.Error()))
		return nil, err
	}

	return &v, nil
}

func (c *CloudEyeCollector) getAllMetrics(namespace string) (*[]metrics.Metric, error) {
	client, err := c.Client.GetCESClient()
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a CES client failed: %s", err.Error()))
		return nil, err
	}
	limit := 1000
	allPages, err := metrics.List(client, metrics.ListOpts{Namespace: namespace, Limit: &limit}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("getting all metrics pages failed: %s", err.Error()))
		return nil, err
	}

	v, err := metrics.ExtractAllPagesMetrics(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("extracting all metrics pages failed: %s", err.Error()))
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
		_, err := validateMetricData(metric)
		if err != nil {
			slog.Error(fmt.Sprintf("[%s] validating metric failed: %s", c.txnKey, err.Error()))
		}
		//slog.Debug(fmt.Sprintf("[%s] validated metric: %s", c.txnKey, string(dataJson)))

		data, err := getLatestData(metric.Datapoints)
		if err != nil {
			slog.Warn(fmt.Sprintf("[%s] gettig latest data failed: %s, metric_name: %s, dimension: %+v", c.txnKey, err.Error(), metric.MetricName, metric.Dimensions))
			continue
		}

		labelInfo, err := relabelMetricData(allResourcesInfo, metric)
		if err != nil {
			slog.Error(fmt.Sprintf("[%s] %s", c.txnKey, err.Error()))
			continue
		}

		fqName := prometheus.BuildFQName(getMetricPrefixName(c.Prefix, metric.Namespace), labelInfo.PreResourceName, metric.MetricName)
		proMetric := prometheus.MustNewConstMetric(
			prometheus.NewDesc(fqName, fqName, labelInfo.Labels, nil),
			prometheus.GaugeValue, data, labelInfo.Values...)
		if err := pushMetricData(ctx, ch, proMetric); err != nil {
			slog.Error(fmt.Sprintf("[%s] context cancellation detected while push metric: %s", c.txnKey, fqName))
		}
	}
}
