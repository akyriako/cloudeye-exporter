package exporter

import (
	"context"
	"encoding/json"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metricdata"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"unsafe"
)

func metricsToMetricData(metric metrics.Metric) metricdata.Metric {
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

func validateMetricData(md metricdata.MetricData) ([]byte, error) {
	dataJson, err := json.Marshal(md)
	if err != nil {
		return nil, err
	}

	return dataJson, nil
}

func pushMetricData(ctx context.Context, ch chan<- prometheus.Metric, metric prometheus.Metric) error {
	// Check whether the Context is cancelled
	select {
	case _, ok := <-ctx.Done():
		if !ok {
			return ctx.Err()
		}
	default: // continue
	}
	// If no, send the metric
	ch <- metric
	return nil
}
