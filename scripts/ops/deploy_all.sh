#!/bin/bash

# 全局部署脚本 - 支持 Helm 一键部署
NAMESPACE=${1:-default}
ACTION=${2:-upgrade --install}

echo "Deploying Financial Trading Platform to namespace: ${NAMESPACE}"

services=(
  "auth"
  "order"
  "marketdata"
  "matchingengine"
  "execution"
  "risk"
  "clearing"
  "connectivity"
  "monitoringanalytics"
)

for svc in "${services[@]}"; do
  echo ">>> Deploying ${svc}..."
  helm ${ACTION} ${svc} ./deployments/${svc}/helm -n ${NAMESPACE} --wait
done

echo ">>> Applying Istio Global Gateway..."
kubectl apply -f ./deployments/global/istio-gateway.yaml -n ${NAMESPACE}

echo ">>> Applying Resilience Policies..."
kubectl apply -f ./deployments/resilience/ -n ${NAMESPACE}

echo "Deployment completed successfully."
