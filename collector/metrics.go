package collector

import (
	"fmt"
	"log/slog"
	"strconv"
	"unsafe"

	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metricdata"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metrics"
)

func transMetric(metric metrics.Metric) metricdata.Metric {
	var m metricdata.Metric
	m.Namespace = metric.Namespace
	m.MetricName = metric.MetricName
	m.Dimensions = *(*[]metricdata.Dimension)(unsafe.Pointer(&metric.Dimensions))
	return m
}

func buildSingleDimensionMetrics(metricNames []string, namespace, dimName, dimValue string) []metrics.Metric {
	filterMetrics := make([]metrics.Metric, 0)
	for index := range metricNames {
		filterMetrics = append(filterMetrics, metrics.Metric{
			Namespace:  namespace,
			MetricName: metricNames[index],
			Dimensions: []metrics.Dimension{
				{
					Name:  dimName,
					Value: dimValue,
				},
			},
		})
	}
	return filterMetrics
}

func (c *OpenTelekomCloudClient) getBatchMetricData(
	metrics *[]metricdata.Metric,
	from string,
	to string,
) (*[]metricdata.MetricData, error) {

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

	client, err := c.GetCESClient()
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

func (c *OpenTelekomCloudClient) getAllMetrics(namespace string) (*[]metrics.Metric, error) {
	client, err := c.GetCESClient()
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
