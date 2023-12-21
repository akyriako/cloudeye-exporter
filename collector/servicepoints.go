package collector

import (
	"fmt"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack"
	"log/slog"
)

func (c *OpenTelekomCloudClient) GetCESClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewCESClient(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the CES client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetELBClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewNetworkV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the NetworkV2 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetNATClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewNatV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the NatV2 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetRDSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewRDSV3(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the RDSV3 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetDCSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewDCSServiceV1(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the DCSV1 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetDMSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewDMSServiceV1(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the DMSV1 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetVPCClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewVPCV1(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the VPCV1 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetEVSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewBlockStorageV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the BlockStorageVS client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetECSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewComputeV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the ECS client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetASClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewAutoScalingService(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the AS client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetFGSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewFGSV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get the FGSV2 client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}
