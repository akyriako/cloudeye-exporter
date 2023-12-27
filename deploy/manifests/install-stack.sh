#!/bin/bash

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm upgrade --install --values prometheus-stack/override.yaml kube-prometheus-stack prometheus-community/kube-prometheus-stack -n monitoring --create-namespace

kubectl apply -f ./