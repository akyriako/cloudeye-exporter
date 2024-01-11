#!/bin/bash

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

envsubst < prometheus-adapter/override.tpl > prometheus-adapter/override.yaml
sleep 15

helm upgrade --install --values prometheus-adapter/override.yaml prometheus-adapter prometheus-community/prometheus-adapter -n monitoring
