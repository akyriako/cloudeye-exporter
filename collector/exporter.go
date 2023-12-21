package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metricdata"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"strings"
	"sync"
)

var defaultLabelsToResource = map[string]string{
	"lbaas_listener_id":         "listener",
	"lb_instance_id":            "lb",
	"direct_connect_id":         "direct",
	"history_direct_connect_id": "history",
	"virtual_interface_id":      "virtual",
	"bandwidth_id":              "bandwidth",
	"publicip_id":               "eip",
	"rabbitmq_instance_id":      "instance",
	"kafka_instance_id":         "instance",
}

var privateResourceFlag = map[string]string{
	"kafka_broker":              "broker",
	"kafka_topics":              "topics",
	"kafka_partitions":          "partitions",
	"kafka_groups":              "groups",
	"rabbitmq_node":             "rabbitmq_node",
	"rds_instance_id":           "instance",
	"postgresql_instance_id":    "instance",
	"rds_instance_sqlserver_id": "instance",
}

func replaceName(name string) string {
	newName := strings.Replace(name, ".", "_", -1)
	newName = strings.ToLower(newName)

	return newName
}

func GetMetricPrefixName(prefix string, namespace string) string {
	return fmt.Sprintf("%s_%s", prefix, replaceName(namespace))
}

func (c *CloudEyeCollector) listMetrics(namespace string) ([]metrics.Metric, map[string][]string) {
	allResourcesInfo, metrics := c.getAllResources(namespace)
	slog.Debug(fmt.Sprintf("[%s] Resource number of %s: %d", c.txnKey, namespace, len(allResourcesInfo)))

	if len(*metrics) > 0 {
		return *metrics, allResourcesInfo
	}
	slog.Debug(fmt.Sprintf("[%s] Start to getAllMetric from CES", c.txnKey))
	allMetrics, err := c.Client.getAllMetrics(namespace)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s] Get all metrics error: %s", c.txnKey, err.Error()))
		return nil, nil
	}
	slog.Debug(fmt.Sprintf("[%s] End to getAllMetric, Total number of of metrics: %d", c.txnKey, len(*allMetrics)))
	return *allMetrics, allResourcesInfo
}

type LabelInfo struct {
	Labels          []string
	Values          []string
	PreResourceName string
}

func (c *CloudEyeCollector) getLabelInfo(allResourcesInfo map[string][]string, metric metricdata.MetricData) *LabelInfo {
	labels, values, preResourceName, privateFlag := getOriginalLabelInfo(&metric.Dimensions)

	if isResourceExist(&metric.Dimensions, &allResourcesInfo) {
		labels = c.getExtensionLabels(labels, preResourceName, metric.Namespace, privateFlag)
		values = c.getExtensionLabelValues(values, &allResourcesInfo, getOriginalID(&metric.Dimensions))
	}

	if len(labels) != len(values) {
		slog.Error(fmt.Sprintf("[%s] Inconsistent label and value: expected %d label %#v, but values got %d in %#v", c.txnKey,
			len(labels), labels, len(values), values))
		return nil
	}
	return &LabelInfo{
		Labels:          labels,
		Values:          values,
		PreResourceName: preResourceName,
	}
}

func (c *CloudEyeCollector) setProData(ctx context.Context, ch chan<- prometheus.Metric,
	dataList []metricdata.MetricData, allResourcesInfo map[string][]string) {
	for _, metric := range dataList {
		c.debugMetricInfo(metric)
		data, err := getLatestData(metric.Datapoints)
		if err != nil {
			slog.Warn(fmt.Sprintf("[%s] Get data point error: %s, metric_name: %s, dimension: %+v", c.txnKey, err.Error(), metric.MetricName, metric.Dimensions))
			continue
		}

		labelInfo := c.getLabelInfo(allResourcesInfo, metric)
		if labelInfo == nil {
			continue
		}

		fqName := prometheus.BuildFQName(GetMetricPrefixName(c.Prefix, metric.Namespace), labelInfo.PreResourceName, metric.MetricName)
		proMetric := prometheus.MustNewConstMetric(
			prometheus.NewDesc(fqName, fqName, labelInfo.Labels, nil),
			prometheus.GaugeValue, data, labelInfo.Values...)
		if err := sendMetricData(ctx, ch, proMetric); err != nil {
			slog.Error(fmt.Sprintf("[%s] Context has canceled, no need to send metric data, metric name: %s", c.txnKey, fqName))
		}
	}
}

func (c *CloudEyeCollector) collectMetricByNamespace(ctx context.Context, ch chan<- prometheus.Metric, namespace string) {
	defer func() {
		if err := recover(); err != nil {
			//logs.Logger.Error(fmt.Sprintf(err)
		}
	}()

	allMetrics, allResourcesInfo := c.listMetrics(namespace)
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
		tmpMetrics = append(tmpMetrics, transMetric(metric))
		if (len(tmpMetrics) == c.ScrapeBatchSize) || (count == len(allMetrics)) {
			workChan <- struct{}{}
			wg.Add(1)
			go func(tmpMetrics []metricdata.Metric) {
				defer func() {
					<-workChan
					wg.Done()
				}()
				slog.Debug(fmt.Sprintf("[%s] Start to getBatchMetricData, metric count: %d", c.txnKey, len(tmpMetrics)))
				dataList, err := c.Client.getBatchMetricData(&tmpMetrics, c.From, c.To)
				if err != nil {
					return
				}
				c.setProData(ctx, ch, *dataList, allResourcesInfo)
			}(tmpMetrics)
			tmpMetrics = make([]metricdata.Metric, 0, c.ScrapeBatchSize)
		}
	}

	wg.Wait()
	slog.Debug(fmt.Sprintf("[%s] End to scrape all metric data", c.txnKey))
}

func sendMetricData(ctx context.Context, ch chan<- prometheus.Metric, metric prometheus.Metric) error {
	// Check whether the Context has canceled
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

func (c *CloudEyeCollector) debugMetricInfo(md metricdata.MetricData) {
	dataJson, err := json.Marshal(md)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s] Marshal metricData error: %s", c.txnKey, err.Error()))
		return
	}
	slog.Debug(fmt.Sprintf("[%s] Get data points of metric are: %s", c.txnKey, string(dataJson)))
}

func isResourceExist(dims *[]metricdata.Dimension, allResourceInfo *map[string][]string) bool {
	if _, ok := (*allResourceInfo)[getOriginalID(dims)]; ok {
		return true
	}

	return false
}

func getLatestData(data []metricdata.Data) (float64, error) {
	if len(data) == 0 {
		return 0, errors.New("data not found")
	}

	return data[len(data)-1].Average, nil
}

func getOriginalID(dimensions *[]metricdata.Dimension) string {
	id := ""

	if len(*dimensions) == 1 {
		id = (*dimensions)[0].Value
	} else if len(*dimensions) == 2 {
		id = (*dimensions)[1].Value
	}

	return id
}

func getOriginalLabelInfo(dims *[]metricdata.Dimension) ([]string, []string, string, string) {
	labels := []string{}
	dimensionValues := []string{}
	preResourceName := ""
	privateFlag := ""
	for _, dimension := range *dims {
		if val, ok := defaultLabelsToResource[dimension.Name]; ok {
			preResourceName = val
		}

		if val, ok := privateResourceFlag[dimension.Name]; ok {
			privateFlag = val
		}

		dimensionValues = append(dimensionValues, dimension.Value)
		if strings.ContainsAny(dimension.Name, "-") {
			labels = append(labels, strings.Replace(dimension.Name, "-", "_", -1))
			continue
		}
		labels = append(labels, dimension.Name)
	}

	return labels, dimensionValues, preResourceName, privateFlag
}

func (c *CloudEyeCollector) getExtensionLabels(
	labels []string, preResourceName string, namespace string, privateFlag string) []string {

	namespace = replaceName(namespace)
	if preResourceName != "" {
		namespace = namespace + "_" + preResourceName
	}

	if privateFlag != "" {
		namespace = namespace + "_" + privateFlag
	}

	newlabels := append(labels, defaultExtensionLabels[namespace]...)

	return newlabels
}

func (c *CloudEyeCollector) getExtensionLabelValues(
	dimensionValues []string,
	allResourceInfo *map[string][]string,
	originalID string) []string {

	for lb := range *allResourceInfo {
		if lb == originalID {
			dimensionValues = append(dimensionValues, (*allResourceInfo)[lb]...)
			return dimensionValues
		}
	}

	return dimensionValues
}
