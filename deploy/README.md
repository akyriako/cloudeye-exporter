# Kubernetes Installation

These are the instructions of installing and configuring cloudeye-exporter on an Open Telekom Cloud CCE cluster.

## Encode clouds.yaml in base64 and insert value in Secret

Fill in the `cloud.tpl` template with your own values, and the encode it in base64 with the following command:

```shell
 base64 -i clouds.tpl -o clouds.yaml
```

Take the encoded contents and replace the value of `clouds.yaml` in `deploy/manifests/cloudeye-exporter-clouds-secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cloudeye-exporter-clouds
  namespace: default
type: Opaque
data:
  clouds.yaml: Z2xvYmFsOg************************************************************
  
```

## Install kube-prometheus-stack & cloudeye-exporter artefacts

We are going to install Prometheus/Grafana stack via the kube-prometheus-stack chart. The configuration values used 
can be found at `deploy/manifests/prometheus-stack/override.yaml`. You could diff them with the default values `default.yaml`
to figure out the changes.

Run `./deploy/manifests/install.sh`. This script will deploy, besides the kube-prometheus-stack, all the cloudeye-exporter 
related artefacts.

## Install nginx demo workload

We are going to need a workload to test HPA and the autoscaling via our custom CloudEye derived metrics. For that matter
we will deploy a dummy nginx deployment and service:

`kubectl apply -f deploy/manifests/nginx-deployment.yaml`

## Install prometheus-adapter

Next, and last step, of the installation sequence is the deployment of prometheus-adapter that will give an additional 
custom metrics api endpoint that will bind our custom CloudEye metrics with HPA. Before installing the chart, you need to
get the Elastic Load Balancer Listener's ID from your Open Telekom Cloud Console and replace the value in `deploy/manifests/install-adapter.sh`:

```shell
#!/bin/bash

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

export $ELB_LISTENER_ID = "94424*****"

helm upgrade --install --values prometheus-adapter/override.yaml prometheus-adapter prometheus-community/prometheus-adapter -n monitoring
```

The configuration values used for the prometheus-adapter chart can be found at `deploy/manifests/prometheus-adapter/override.yaml`.
You could diff them with the default values `default.yaml` to figure out the changes.

## Stress-test nginx workload

**Please fill in**




