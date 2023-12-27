#!/bin/bash

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

export $ELB_LISTENER_ID = "94424*****"

helm upgrade --install --values prometheus-adapter/override.yaml prometheus-adapter prometheus-community/prometheus-adapter -n monitoring
