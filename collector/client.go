package collector

import (
	"errors"
	"fmt"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack"
	"github.com/huaweicloud/golangsdk/openstack/autoscaling/v1/groups"
	"github.com/huaweicloud/golangsdk/openstack/blockstorage/v2/volumes"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/servers"
	dcs "github.com/huaweicloud/golangsdk/openstack/dcs/v1/instances"
	dms "github.com/huaweicloud/golangsdk/openstack/dms/v1/instances"
	"github.com/huaweicloud/golangsdk/openstack/dms/v1/queues"
	"github.com/huaweicloud/golangsdk/openstack/fgs/v2/function"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/natgateways"
	rds "github.com/huaweicloud/golangsdk/openstack/rds/v3/instances"
	"github.com/huaweicloud/golangsdk/openstack/vpc/v1/bandwidths"
	"github.com/huaweicloud/golangsdk/openstack/vpc/v1/publicips"
	"log/slog"
	"net/http"
)

type ClientConfig struct {
	AccessKey        string
	SecretKey        string
	DomainID         string
	DomainName       string
	EndpointType     string
	IdentityEndpoint string
	Insecure         bool
	Password         string
	Region           string
	TenantID         string
	TenantName       string
	Token            string
	Username         string
	UserID           string
}

type OpenTelekomCloudClient struct {
	HwClient *golangsdk.ProviderClient
	Config   ClientConfig
}

func NewOpenTelekomCloudClient(config *config.CloudConfig) (*OpenTelekomCloudClient, error) {
	auth := config.Auth
	clientConfig := ClientConfig{
		IdentityEndpoint: auth.AuthURL,
		TenantName:       auth.ProjectName,
		AccessKey:        auth.AccessKey,
		SecretKey:        auth.SecretKey,
		DomainName:       auth.DomainName,
		Username:         auth.UserName,
		Region:           auth.Region,
		Password:         auth.Password,
		Insecure:         true,
	}

	client, err := buildClient(&clientConfig)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to build client: %s", err.Error()))
		return nil, err
	}

	return client, err
}

func buildClient(c *ClientConfig) (*OpenTelekomCloudClient, error) {
	if c.AccessKey != "" && c.SecretKey != "" {
		return buildClientByAKSK(c)
	} else if c.Password != "" && (c.Username != "" || c.UserID != "") {
		return buildClientByPassword(c)
	}

	return nil, errors.New("Must config token or aksk or username password to be authorized")
}

func buildClientByPassword(c *ClientConfig) (*OpenTelekomCloudClient, error) {
	var pao, dao golangsdk.AuthOptions

	pao = golangsdk.AuthOptions{
		DomainID:   c.DomainID,
		DomainName: c.DomainName,
		TenantID:   c.TenantID,
		TenantName: c.TenantName,
	}

	dao = golangsdk.AuthOptions{
		DomainID:   c.DomainID,
		DomainName: c.DomainName,
	}

	for _, ao := range []*golangsdk.AuthOptions{&pao, &dao} {
		ao.IdentityEndpoint = c.IdentityEndpoint
		ao.Password = c.Password
		ao.Username = c.Username
		ao.UserID = c.UserID
	}

	return newOpenTelekomCloudClient(c, pao, dao)
}

func buildClientByAKSK(c *ClientConfig) (*OpenTelekomCloudClient, error) {
	var pao, dao golangsdk.AKSKAuthOptions

	pao = golangsdk.AKSKAuthOptions{
		ProjectName: c.TenantName,
		ProjectId:   c.TenantID,
	}

	dao = golangsdk.AKSKAuthOptions{
		DomainID: c.DomainID,
		Domain:   c.DomainName,
	}

	for _, ao := range []*golangsdk.AKSKAuthOptions{&pao, &dao} {
		ao.IdentityEndpoint = c.IdentityEndpoint
		ao.AccessKey = c.AccessKey
		ao.SecretKey = c.SecretKey
	}
	return newOpenTelekomCloudClient(c, pao, dao)
}

func newOpenTelekomCloudClient(c *ClientConfig, pao, dao golangsdk.AuthOptionsProvider) (*OpenTelekomCloudClient, error) {
	openstackClient, err := newOpenStackClient(c, pao)
	if err != nil {
		return nil, err
	}

	client := &OpenTelekomCloudClient{
		HwClient: openstackClient,
		Config:   *c,
	}

	return client, err
}

func newOpenStackClient(c *ClientConfig, ao golangsdk.AuthOptionsProvider) (*golangsdk.ProviderClient, error) {
	client, err := openstack.NewClient(ao.GetIdentityEndpoint())
	if err != nil {
		return nil, err
	}

	client.HTTPClient = http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if client.AKSKAuthOptions.AccessKey != "" {
				golangsdk.ReSign(req, golangsdk.SignOptions{
					AccessKey: client.AKSKAuthOptions.AccessKey,
					SecretKey: client.AKSKAuthOptions.SecretKey,
				})
			}
			return nil
		},
	}

	err = openstack.Authenticate(client, ao)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetServiceEndpoint(namespace string) (*golangsdk.ServiceClient, error) {
	switch namespace {
	case "SYS.CES":
		return c.GetCESClient()
	case "SYS.ELB":
		return c.GetELBClient()
	case "SYS.NAT":
		return c.GetNATClient()
	case "SYS.RDS":
		return c.GetRDSClient()
	case "SYS.DCS":
		return c.GetDCSClient()
	case "SYS.DMS":
		return c.GetDMSClient()
	case "SYS.VPC":
		return c.GetVPCClient()
	case "SYS.EVS":
		return c.GetEVSClient()
	case "SYS.ECS":
		return c.GetECSClient()
	case "SYS.AS":
		return c.GetASClient()
	case "SYS.FGS":
		return c.GetFGSClient()
	default:
		return nil, fmt.Errorf("could not provide a service endpoint for namespace: %s", namespace)
	}
}

func (c *OpenTelekomCloudClient) getAllLoadBalancers() (*[]loadbalancers.LoadBalancer, error) {
	client, err := c.GetELBClient()
	if err != nil {
		return nil, err
	}

	allPages, err := loadbalancers.List(client, loadbalancers.ListOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List load balancer error: %s", err.Error()))
		return nil, err
	}

	allLoadBalancers, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract load balancer pages error: %s", err.Error()))
		return nil, err
	}

	return &allLoadBalancers, nil
}

func (c *OpenTelekomCloudClient) getAllListeners() (*[]listeners.Listener, error) {
	client, err := c.GetELBClient()
	if err != nil {
		return nil, err
	}

	allPages, err := listeners.List(client, listeners.ListOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List listener all pages error: %s", err.Error()))
		return nil, err
	}

	allListeners, err := listeners.ExtractListeners(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract listener pages error: %s", err.Error()))
		return nil, err
	}

	return &allListeners, nil
}

func (c *OpenTelekomCloudClient) getAllNatGateways() (*[]natgateways.NatGateway, error) {
	client, err := c.GetNATClient()
	if err != nil {
		return nil, err
	}

	allPages, err := natgateways.List(client, natgateways.ListOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List nat gateways error: %s", err.Error()))
		return nil, err
	}

	allNatGateways, err := natgateways.ExtractNatGateways(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract nat gateway pages error: %s", err.Error()))
		return nil, err
	}

	return &allNatGateways, nil
}

func (c *OpenTelekomCloudClient) getAllRDSs() (*rds.ListRdsResponse, error) {
	client, err := c.GetRDSClient()
	if err != nil {
		return nil, err
	}

	// HACK: Changed to ListOps
	allPages, err := rds.List(client, rds.ListOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List rds error: %s", err.Error()))
		return nil, err
	}

	allRds, err := rds.ExtractRdsInstances(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract rds pages error: %s", err.Error()))
		return nil, err
	}

	return &allRds, nil
}

func (c *OpenTelekomCloudClient) getAllDCSs() (*dcs.ListDcsResponse, error) {
	client, err := c.GetDCSClient()
	if err != nil {
		return nil, err
	}

	allPages, err := dcs.List(client, dcs.ListDcsInstanceOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List dcs error: %s", err.Error()))
		return nil, err
	}

	allDcs, err := dcs.ExtractDcsInstances(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract dcs pages error: %s", err.Error()))
		return nil, err
	}

	return &allDcs, nil
}

func (c *OpenTelekomCloudClient) getAllDMSs() (*dms.ListDmsResponse, error) {
	client, err := c.GetDMSClient()
	if err != nil {
		return nil, err
	}

	allPages, err := dms.List(client, dms.ListDmsInstanceOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List dms instances error: %s", err.Error()))
		return nil, err
	}

	allDms, err := dms.ExtractDmsInstances(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract dms instances pages error: %s", err.Error()))
		return nil, err
	}

	return &allDms, nil
}

func (c *OpenTelekomCloudClient) getAllDMSQueues() (*[]queues.Queue, error) {
	client, err := c.GetDMSClient()
	if err != nil {
		return nil, err
	}

	allPages, err := queues.List(client, false).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List dms queues error: %s", err.Error()))
		return nil, err
	}

	allQueues, err := queues.ExtractQueues(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract dms queues pages error: %s", err.Error()))
		return nil, err
	}

	return &allQueues, nil
}

func (c *OpenTelekomCloudClient) getAllPublicIPs() (*[]publicips.PublicIP, error) {
	client, err := c.GetVPCClient()
	if err != nil {
		return nil, err
	}

	allPages, err := publicips.List(client, publicips.ListOpts{
		Limit: 1000,
	}).AllPages()

	if err != nil {
		slog.Error(fmt.Sprintf("List public ips error: %s", err.Error()))
		return nil, err
	}
	publicipList, err1 := publicips.ExtractPublicIPs(allPages)

	if err1 != nil {
		slog.Error(fmt.Sprintf("Extract public ips pages error: %s", err.Error()))
		return nil, err
	}

	return &publicipList, nil
}

func (c *OpenTelekomCloudClient) getAllBandwidth() (*[]bandwidths.BandWidth, error) {
	client, err := c.GetVPCClient()
	if err != nil {
		return nil, err
	}

	allPages, err := bandwidths.List(client, bandwidths.ListOpts{
		Limit: 1000,
	}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List bandwidths error: %s", err.Error()))
		return nil, err
	}

	result, err := bandwidths.ExtractBandWidths(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract bandwidths all pages error: %s", err.Error()))
		return nil, err
	}

	return &result, nil
}

func (c *OpenTelekomCloudClient) getAllVolumes() (*[]volumes.Volume, error) {
	client, err := c.GetEVSClient()
	if err != nil {
		return nil, err
	}

	allPages, err := volumes.List(client, volumes.ListOpts{
		Limit: 1000,
	}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List volumes error: %s", err.Error()))
		return nil, err
	}

	result, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract volumes all pages error: %s", err.Error()))
		return nil, err
	}

	return &result, nil
}

func (c *OpenTelekomCloudClient) getAllServers() (*[]servers.Server, error) {
	client, err := c.GetECSClient()
	if err != nil {
		return nil, err
	}

	allPages, err := servers.List(client, servers.ListOpts{
		Limit: 1000,
	}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List servers error: %s", err.Error()))
		return nil, err
	}

	result, err := servers.ExtractServers(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract servers all pages error: %s", err.Error()))
		return nil, err
	}

	return &result, nil
}

func (c *OpenTelekomCloudClient) getAllAutoscalingGroups() (*[]groups.Group, error) {
	client, err := c.GetASClient()
	if err != nil {
		return nil, err
	}

	allPages, err := groups.List(client, groups.ListOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List groups error: %s", err.Error()))
		return nil, err
	}

	result, err := (allPages.(groups.GroupPage)).Extract()
	if err != nil {
		slog.Error(fmt.Sprintf("Extract groups all pages error: %s", err.Error()))
		return nil, err
	}

	return &result, nil
}

func (c *OpenTelekomCloudClient) getAllFunctions() (*function.FunctionList, error) {
	client, err := c.GetFGSClient()
	if err != nil {
		return nil, err
	}

	allPages, err := function.List(client, function.ListOpts{}).AllPages()
	if err != nil {
		slog.Error(fmt.Sprintf("List function error: %s", err.Error()))
		return nil, err
	}

	result, err := function.ExtractList(allPages)
	if err != nil {
		slog.Error(fmt.Sprintf("Extract function all pages error: %s", err.Error()))
		return nil, err
	}

	return &result, nil
}
