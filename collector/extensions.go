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
// 2. Added the new labels values to getAllResources
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

type serversInfo struct {
	TTL           int64
	LenMetric     int
	Info          map[string][]string
	FilterMetrics []metrics.Metric
	sync.Mutex
}

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

func (c *CloudEyeExporter) getELBResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	elbInfo.Lock()
	defer elbInfo.Unlock()
	if elbInfo.Info == nil || time.Now().Unix() > elbInfo.TTL {
		allELBs, err := c.Client.getAllLoadBalancers()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all load balancers failed: %s", err.Error()))
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
				filterMetrics = append(filterMetrics, c.buildELBListenerMetrics(metricNames, &elb)...)
			}
			if metricNames, ok := configMap["lbaas_instance_id,lbaas_pool_id"]; ok {
				filterMetrics = append(filterMetrics, c.buildELBPoolMetrics(metricNames, &elb)...)
			}
		}

		allListeners, err := c.Client.getAllListeners()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all listeners failed: %s", err.Error()))
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

func (c *CloudEyeExporter) buildELBListenerMetrics(metricNames []string, elb *loadbalancers.LoadBalancer) []metrics.Metric {
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

func (c *CloudEyeExporter) buildELBPoolMetrics(metricNames []string, elb *loadbalancers.LoadBalancer) []metrics.Metric {
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

func (c *CloudEyeExporter) getNATResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	natInfo.Lock()
	defer natInfo.Unlock()
	if natInfo.Info == nil || time.Now().Unix() > natInfo.TTL {
		allnat, err := c.Client.getAllNatGateways()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all NAT gateways failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getRDSResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	rdsInfo.Lock()
	defer rdsInfo.Unlock()
	if rdsInfo.Info == nil || time.Now().Unix() > rdsInfo.TTL {
		allrds, err := c.Client.getAllRDSs()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all RDS instances failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getDMSResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	dmsInfo.Lock()
	defer dmsInfo.Unlock()
	if dmsInfo.Info == nil || time.Now().Unix() > dmsInfo.TTL {
		allDmsInstance, err := c.Client.getAllDMSs()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all DMS instances failed: %s", err.Error()))
			return dmsInfo.Info, &dmsInfo.FilterMetrics
		}
		if allDmsInstance == nil {
			return dmsInfo.Info, &dmsInfo.FilterMetrics
		}

		for _, dms := range allDmsInstance.Instances {
			resourceInfos[dms.InstanceID] = []string{dms.Name, dms.EngineVersion, dms.ResourceSpecCode, dms.ConnectAddress,
				fmt.Sprintf("%d", dms.Port)}
		}

		allQueues, err := c.Client.getAllDMSQueues()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all DMS queues failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getDCSResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := make(map[string][]string)
	filterMetrics := make([]metrics.Metric, 0)
	dcsInfo.Lock()
	defer dcsInfo.Unlock()
	if dcsInfo.Info == nil || time.Now().Unix() > dcsInfo.TTL {
		allDcs, err := c.Client.getAllDCSs()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all DCS failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getVPCResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	vpcInfo.Lock()
	defer vpcInfo.Unlock()
	if vpcInfo.Info == nil || time.Now().Unix() > vpcInfo.TTL {
		allPublicIps, err := c.Client.getAllPublicIPs()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all public ips failed: %s", err.Error()))
		}
		if allPublicIps != nil {
			for _, publicIp := range *allPublicIps {
				resourceInfos[publicIp.ID] = []string{publicIp.BandwidthName, publicIp.PublicIpAddress, publicIp.Type}
			}
		}

		allBandwidth, err := c.Client.getAllBandwidth()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all bandwidth failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getEVSResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	evsInfo.Lock()
	defer evsInfo.Unlock()
	if evsInfo.Info == nil || time.Now().Unix() > evsInfo.TTL {
		allVolumes, err := c.Client.getAllVolumes()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all volumes failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getECSResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	ecsInfo.Lock()
	defer ecsInfo.Unlock()
	if ecsInfo.Info == nil || time.Now().Unix() > ecsInfo.TTL {
		allServers, err := c.Client.getAllServers()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all servers failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getASResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	asInfo.Lock()
	defer asInfo.Unlock()
	if asInfo.Info == nil || time.Now().Unix() > asInfo.TTL {
		allGroups, err := c.Client.getAllAutoscalingGroups()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all autoscaling groups failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getFGSResourceInfo() (map[string][]string, *[]metrics.Metric) {
	resourceInfos := map[string][]string{}
	fgsInfo.Lock()
	defer fgsInfo.Unlock()
	if fgsInfo.Info == nil || time.Now().Unix() > fgsInfo.TTL {
		functionList, err := c.Client.getAllFunctions()
		if err != nil {
			slog.Error(fmt.Sprintf("getting all functions failed: %s", err.Error()))
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

func (c *CloudEyeExporter) getAllResources(namespace string) (map[string][]string, *[]metrics.Metric) {
	switch namespace {
	case "SYS.ELB":
		return c.getELBResourceInfo()
	case "SYS.NAT":
		return c.getNATResourceInfo()
	case "SYS.RDS":
		return c.getRDSResourceInfo()
	case "SYS.DMS":
		return c.getDMSResourceInfo()
	case "SYS.DCS":
		return c.getDCSResourceInfo()
	case "SYS.VPC":
		return c.getVPCResourceInfo()
	case "SYS.EVS":
		return c.getEVSResourceInfo()
	case "SYS.ECS":
		return c.getECSResourceInfo()
	case "SYS.AS":
		return c.getASResourceInfo()
	case "SYS.FunctionGraph":
		return c.getFGSResourceInfo()
	default:
		return map[string][]string{}, &[]metrics.Metric{}
	}
}
