#!/bin/bash

kubectl create namespace applications
kubectl apply -f deploy/manifests/nginx-deployment.yaml
kubectl apply -f deploy/manifests/nginx-ingress.yaml