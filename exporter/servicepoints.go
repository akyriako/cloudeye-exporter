package exporter

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
		slog.Error(fmt.Sprintf("acquiring a CES client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetELBClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewNetworkV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a NetworkV2 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetNATClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewNatV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a NatV2 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetRDSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewRDSV3(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a RDSV3 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetDCSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewDCSServiceV1(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a DCSV1 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetDMSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewDMSServiceV1(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a DMSV1 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetVPCClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewVPCV1(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a VPCV1 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetEVSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewBlockStorageV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a BlockStorageV2 client failed client: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetECSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewComputeV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring an ECS client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetASClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewAutoScalingService(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring an AS client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}

func (c *OpenTelekomCloudClient) GetFGSClient() (*golangsdk.ServiceClient, error) {
	client, err := openstack.NewFGSV2(c.HwClient, golangsdk.EndpointOpts{
		Region: c.Config.Region,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("acquiring a FGSV2 client failed: %s", err.Error()))
		return nil, err
	}

	return client, nil
}
