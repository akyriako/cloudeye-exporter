package collector

import (
	"fmt"
	"github.com/akyriako/cloudeye-exporter/config"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/huaweicloud/golangsdk/openstack/ces/v1/metrics"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
)

// If the extension labels have to added in this exporter, you only have
// to add the code to the following two parts.
// 1. Added the new labels name to defaultExtensionLabels
// 2. Added the new labels values to getAllResource
var defaultExtensionLabels = map[string][]string{
	"sys_elb":                 []string{"name", "provider", "vip_address"},
	"sys_elb_listener":        []string{"name", "port"},
	"sys_nat":                 []string{"name"},
	"sys_rds":                 []string{"name"},
	"sys_rds_instance":        []string{"port", "name", "role"},
	"sys_dcs":                 []string{"ip", "port", "name", "engine"},
	"sys_dms":                 []string{"name"},
	"sys_dms_instance":        []string{"name", "engine_version", "resource_spec_code", "connect_address", "port"},
	"sys_dms_instance_broker": []string{"name", "engine_version", "resource_spec_code", "connect_address", "port"},
	"sys_dms_instance_topics": []string{"name", "engine_version", "resource_spec_code", "connect_address", "port"},
	"sys_vpc_bandwidth":       []string{"name", "size", "share_type", "bandwidth_type", "charge_mode"},
	"sys_vpc_eip":             []string{"name", "public_ip_address", "type"},
	"sys_evs":                 []string{"name", "server_id", "device"},
	"sys_ecs":                 []string{"hostname"},
	"sys_as":                  []string{"name", "status"},
	"sys_functiongraph":       []string{"func_urn"},
}

const TTL = time.Hour * 3

var (
	elbInfo serversInfo
	natInfo serversInfo
	rdsInfo serversInfo
	dmsInfo serversInfo
	dcsInfo serversInfo
	vpcInfo serversInfo
	evsInfo serversInfo
	ecsInfo serversInfo
	asInfo  serversInfo
	fgsInfo serversInfo
)

type serversInfo struct {
	TTL           int64
	LenMetric     int
	Info          map[string][]string
	FilterMetrics []metrics.Metric
	sync.Mutex
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

func (c *CloudEyeCollector) getElbResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	elbInfo.Lock()
	defer elbInfo.Unlock()
	if elbInfo.Info == nil || time.Now().Unix() > elbInfo.TTL {
		allELBs, err := getAllLoadBalancer(c.ClientConfig)
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

		allListeners, err := getAllListener(c.ClientConfig)
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

func buildListenerMetrics(metricNames []string, elb *loadbalancers.LoadBalancer) []metrics.Metric {
	filterMetrics := make([]metrics.Metric, 0)
	for listenerIndex := range elb.Listeners {
		for index := range metricNames {
			filterMetrics = append(filterMetrics, metrics.Metric{
				Namespace:  "SYS.ELB",
				MetricName: metricNames[index],
				Dimensions: []metrics.Dimension{
					{
						Name:  "lbaas_instance_id",
						Value: elb.ID,
					},
					{
						Name:  "lbaas_listener_id",
						Value: elb.Listeners[listenerIndex].ID,
					},
				},
			})
		}
	}
	return filterMetrics
}

func buildPoolMetrics(metricNames []string, elb *loadbalancers.LoadBalancer) []metrics.Metric {
	filterMetrics := make([]metrics.Metric, 0)
	for poolIndex := range elb.Pools {
		for index := range metricNames {
			filterMetrics = append(filterMetrics, metrics.Metric{
				Namespace:  "SYS.ELB",
				MetricName: metricNames[index],
				Dimensions: []metrics.Dimension{
					{
						Name:  "lbaas_instance_id",
						Value: elb.ID,
					},
					{
						Name:  "lbaas_pool_id",
						Value: elb.Pools[poolIndex].ID,
					},
				},
			})
		}
	}
	return filterMetrics
}

func (c *CloudEyeCollector) getNatResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	natInfo.Lock()
	defer natInfo.Unlock()
	if natInfo.Info == nil || time.Now().Unix() > natInfo.TTL {
		allnat, err := getAllNat(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Nat error: %s", err.Error()))
			return natInfo.Info, &natInfo.FilterMetrics
		}
		if allnat == nil {
			return natInfo.Info, &natInfo.FilterMetrics
		}
		configMap := config.GetMetricFilters("SYS.NAT")
		for _, nat := range *allnat {
			resourceInfos[nat.ID] = []string{nat.Name}
			if configMap == nil {
				continue
			}
			if metricNames, ok := configMap["nat_gateway_id"]; ok {
				filterMetrics = append(filterMetrics, buildSingleDimensionMetrics(metricNames, "SYS.NAT", "nat_gateway_id", nat.ID)...)
			}
		}

		natInfo.Info = resourceInfos
		natInfo.FilterMetrics = filterMetrics
		natInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return natInfo.Info, &natInfo.FilterMetrics
}

func (c *CloudEyeCollector) getRdsResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	rdsInfo.Lock()
	defer rdsInfo.Unlock()
	if rdsInfo.Info == nil || time.Now().Unix() > rdsInfo.TTL {
		allrds, err := getAllRds(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Rds error: %s", err.Error()))
			return rdsInfo.Info, &rdsInfo.FilterMetrics
		}
		if allrds == nil {
			return rdsInfo.Info, &rdsInfo.FilterMetrics
		}
		configMap := config.GetMetricFilters("SYS.RDS")
		for _, rds := range allrds.Instances {
			resourceInfos[rds.Id] = []string{rds.Name}
			for _, node := range rds.Nodes {
				resourceInfos[node.Id] = []string{fmt.Sprintf("%d", rds.Port), node.Name, node.Role}
			}
			if configMap == nil {
				continue
			}
			var dimName string
			switch rds.DataStore.Type {
			case "MySQL":
				dimName = "rds_cluster_id"
			case "PostgreSQL":
				dimName = "postgresql_cluster_id"
			case "SQLServer":
				dimName = "rds_cluster_sqlserver_id"
			}
			if metricNames, ok := configMap[dimName]; ok {
				filterMetrics = append(filterMetrics, buildSingleDimensionMetrics(metricNames, "SYS.RDS", dimName, rds.Id)...)
			}
		}

		rdsInfo.Info = resourceInfos
		rdsInfo.FilterMetrics = filterMetrics
		rdsInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return rdsInfo.Info, &rdsInfo.FilterMetrics
}

func (c *CloudEyeCollector) getDmsResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	dmsInfo.Lock()
	defer dmsInfo.Unlock()
	if dmsInfo.Info == nil || time.Now().Unix() > dmsInfo.TTL {
		allDmsInstance, err := getAllDms(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Dms error: %s", err.Error()))
			return dmsInfo.Info, &dmsInfo.FilterMetrics
		}
		if allDmsInstance == nil {
			return dmsInfo.Info, &dmsInfo.FilterMetrics
		}

		for _, dms := range allDmsInstance.Instances {
			resourceInfos[dms.InstanceID] = []string{dms.Name, dms.EngineVersion, dms.ResourceSpecCode, dms.ConnectAddress,
				fmt.Sprintf("%d", dms.Port)}
		}

		allQueues, err := getAllDmsQueue(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Dms Queue error: %s", err.Error()))
		}
		if allQueues != nil {
			for _, queue := range *allQueues {
				resourceInfos[queue.ID] = []string{queue.Name}
			}
		}

		dmsInfo.Info = resourceInfos
		dmsInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return dmsInfo.Info, &dmsInfo.FilterMetrics
}

func (c *CloudEyeCollector) getDcsResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	dcsInfo.Lock()
	defer dcsInfo.Unlock()
	if dcsInfo.Info == nil || time.Now().Unix() > dcsInfo.TTL {
		allDcs, err := getAllDcs(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Dcs error: %s", err.Error()))
			return dcsInfo.Info, &dcsInfo.FilterMetrics
		}
		if allDcs == nil {
			return dcsInfo.Info, &dcsInfo.FilterMetrics
		}
		configMap := config.GetMetricFilters("SYS.DCS")
		for _, dcs := range allDcs.Instances {
			resourceInfos[dcs.InstanceID] = []string{dcs.IP, fmt.Sprintf("%d", dcs.Port), dcs.Name, dcs.Engine}
			if configMap == nil {
				continue
			}
			var dimName string
			switch dcs.Engine {
			case "Redis":
				dimName = "dcs_instance_id"
			case "Memcached":
				dimName = "dcs_memcached_instance_id"
			}
			if metricNames, ok := configMap[dimName]; ok {
				filterMetrics = append(filterMetrics, buildSingleDimensionMetrics(metricNames, "SYS.DCS", dimName, dcs.InstanceID)...)
			}
		}

		dcsInfo.Info = resourceInfos
		dcsInfo.FilterMetrics = filterMetrics
		dcsInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return dcsInfo.Info, &dcsInfo.FilterMetrics
}

func (c *CloudEyeCollector) getVpcResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	vpcInfo.Lock()
	defer vpcInfo.Unlock()
	if vpcInfo.Info == nil || time.Now().Unix() > vpcInfo.TTL {
		allPublicIps, err := getAllPublicIp(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all PublicIp error: %s", err.Error()))
		}
		if allPublicIps != nil {
			for _, publicIp := range *allPublicIps {
				resourceInfos[publicIp.ID] = []string{publicIp.BandwidthName, publicIp.PublicIpAddress, publicIp.Type}
			}
		}

		allBandwidth, err := getAllBandwidth(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Bandwidth error: %s", err.Error()))
			return resourceInfos, &vpcInfo.FilterMetrics
		}
		if allBandwidth != nil {
			for _, bandwidth := range *allBandwidth {
				resourceInfos[bandwidth.ID] = []string{bandwidth.Name, fmt.Sprintf("%d", bandwidth.Size), bandwidth.ShareType, bandwidth.BandwidthType, bandwidth.ChargeMode}
			}
		}

		vpcInfo.Info = resourceInfos
		vpcInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return vpcInfo.Info, &vpcInfo.FilterMetrics
}

func (c *CloudEyeCollector) getEvsResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	evsInfo.Lock()
	defer evsInfo.Unlock()
	if evsInfo.Info == nil || time.Now().Unix() > evsInfo.TTL {
		allVolumes, err := getAllVolume(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Volume error: %s", err.Error()))
			return evsInfo.Info, &evsInfo.FilterMetrics
		}
		if allVolumes == nil {
			return evsInfo.Info, &evsInfo.FilterMetrics
		}

		for _, volume := range *allVolumes {
			if len(volume.Attachments) > 0 {
				device := strings.Split(volume.Attachments[0].Device, "/")
				resourceInfos[fmt.Sprintf("%s-%s", volume.Attachments[0].ServerID, device[len(device)-1])] = []string{volume.Name, volume.Attachments[0].ServerID, volume.Attachments[0].Device}
			}
		}

		evsInfo.Info = resourceInfos
		evsInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return evsInfo.Info, &evsInfo.FilterMetrics
}

func (c *CloudEyeCollector) getEcsResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	ecsInfo.Lock()
	defer ecsInfo.Unlock()
	if ecsInfo.Info == nil || time.Now().Unix() > ecsInfo.TTL {
		allServers, err := getAllServer(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Server error: %s", err.Error()))
			return ecsInfo.Info, &ecsInfo.FilterMetrics
		}
		if allServers == nil {
			return ecsInfo.Info, &ecsInfo.FilterMetrics
		}

		for _, server := range *allServers {
			resourceInfos[server.ID] = []string{server.Name}
		}

		ecsInfo.Info = resourceInfos
		ecsInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return ecsInfo.Info, &ecsInfo.FilterMetrics
}

func (c *CloudEyeCollector) getAsResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	asInfo.Lock()
	defer asInfo.Unlock()
	if asInfo.Info == nil || time.Now().Unix() > asInfo.TTL {
		allGroups, err := getAllGroup(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Group error: %s", err.Error()))
			return asInfo.Info, &asInfo.FilterMetrics
		}
		if allGroups == nil {
			return asInfo.Info, &asInfo.FilterMetrics
		}

		for _, group := range *allGroups {
			resourceInfos[group.ID] = []string{group.Name, group.Status}
		}

		asInfo.Info = resourceInfos
		asInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return asInfo.Info, &asInfo.FilterMetrics
}

func (c *CloudEyeCollector) getFunctionGraphResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	fgsInfo.Lock()
	defer fgsInfo.Unlock()
	if fgsInfo.Info == nil || time.Now().Unix() > fgsInfo.TTL {
		functionList, err := getAllFunction(c.ClientConfig)
		if err != nil {
			slog.Error(fmt.Sprintf("Get all Function error: %s", err.Error()))
			return fgsInfo.Info, &fgsInfo.FilterMetrics
		}
		if functionList == nil {
			return fgsInfo.Info, &fgsInfo.FilterMetrics
		}

		for _, function := range functionList.Functions {
			resourceInfos[fmt.Sprintf("%s-%s", function.Package, function.FuncName)] = []string{function.FuncUrn}
		}

		fgsInfo.Info = resourceInfos
		fgsInfo.TTL = time.Now().Add(TTL).Unix()
	}
	return fgsInfo.Info, &fgsInfo.FilterMetrics
}

func (c *CloudEyeCollector) getAllResource(namespace string) (map[string][]string, *[]metrics.Metric) {
	switch namespace {
	case "SYS.ELB":
		return c.getElbResourceInfo()
	case "SYS.NAT":
		return c.getNatResourceInfo()
	case "SYS.RDS":
		return c.getRdsResourceInfo()
	case "SYS.DMS":
		return c.getDmsResourceInfo()
	case "SYS.DCS":
		return c.getDcsResourceInfo()
	case "SYS.VPC":
		return c.getVpcResourceInfo()
	case "SYS.EVS":
		return c.getEvsResourceInfo()
	case "SYS.ECS":
		return c.getEcsResourceInfo()
	case "SYS.AS":
		return c.getAsResourceInfo()
	case "SYS.FunctionGraph":
		return c.getFunctionGraphResourceInfo()
	default:
		return map[string][]string{}, &[]metrics.Metric{}
	}
}
