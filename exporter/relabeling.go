package exporter

import (
	"errors"
	"fmt"
	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metricdata"
	"strings"
)

type LabelInfo struct {
	Labels          []string
	Values          []string
	PreResourceName string
}

var (
	defaultLabelsToResource = map[string]string{
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

	privateResourceFlag = map[string]string{
		"kafka_broker":              "broker",
		"kafka_topics":              "topics",
		"kafka_partitions":          "partitions",
		"kafka_groups":              "groups",
		"rabbitmq_node":             "rabbitmq_node",
		"rds_instance_id":           "instance",
		"postgresql_instance_id":    "instance",
		"rds_instance_sqlserver_id": "instance",
	}
)

func sanitazeNamespace(namespace string) string {
	namespace = strings.Replace(namespace, ".", "_", -1)
	namespace = strings.ToLower(namespace)

	return namespace
}

func getMetricPrefixName(prefix string, namespace string) string {
	return fmt.Sprintf("%s_%s", prefix, sanitazeNamespace(namespace))
}

func relabelMetricData(allResourcesInfo map[string][]string, metric metricdata.MetricData) (*LabelInfo, error) {
	labels, values, preResourceName, privateFlag := getOriginalLabelInfo(&metric.Dimensions)

	if isInTheResourceList(&metric.Dimensions, &allResourcesInfo) {
		labels = getExtensionLabels(labels, preResourceName, metric.Namespace, privateFlag)
		values = getExtensionLabelValues(values, &allResourcesInfo, getOriginalID(&metric.Dimensions))
	}

	if len(labels) != len(values) {
		return nil, fmt.Errorf("inconsistent label and value: expected %d label %#v, but values got %d in %#v", len(labels), labels, len(values), values)
	}
	return &LabelInfo{
		Labels:          labels,
		Values:          values,
		PreResourceName: preResourceName,
	}, nil
}

func isInTheResourceList(dims *[]metricdata.Dimension, allResourceInfo *map[string][]string) bool {
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

func getExtensionLabels(
	labels []string, preResourceName string, namespace string, privateFlag string) []string {

	namespace = sanitazeNamespace(namespace)
	if preResourceName != "" {
		namespace = namespace + "_" + preResourceName
	}

	if privateFlag != "" {
		namespace = namespace + "_" + privateFlag
	}

	newlabels := append(labels, defaultExtensionLabels[namespace]...)

	return newlabels
}

func getExtensionLabelValues(
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
