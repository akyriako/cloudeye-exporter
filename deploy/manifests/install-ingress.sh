#!/bin/bash

envsubst < nginx-ingress-controller/override.tpl > nginx-ingress-controller/override.yaml
sleep 15

helm upgrade --install -f nginx-ingress-controller/override.yaml --install ingress-nginx ingress-nginx --repo https://kubernetes.github.io/ingress-nginx --namespace ingress-nginx --create-namespace

