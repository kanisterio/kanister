#!/bin/bash

# Copyright 2019 The Kanister Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset

# -----------------------------
# Config (override via env)
# -----------------------------
NAMESPACE="${REGISTRY_NAMESPACE:-registry}"
RELEASE_NAME="${REGISTRY_RELEASE_NAME:-local-registry}"
CHART_PATH="${REGISTRY_CHART_PATH:-helm/local-registry}"

STORAGE_CLASS="${REGISTRY_STORAGE_CLASS:-csi-hostpath-sc}"
STORAGE_SIZE="${REGISTRY_STORAGE_SIZE:-8Gi}"

WAIT_TIMEOUT="${REGISTRY_WAIT_TIMEOUT:-120s}"

# -----------------------------
# Pre-flight checks
# -----------------------------
command -v kubectl >/dev/null || {
  echo "kubectl not found"
  exit 1
}

command -v helm >/dev/null || {
  echo "helm not found"
  exit 1
}

kubectl cluster-info >/dev/null

# -----------------------------
# Namespace
# -----------------------------
kubectl get ns "${NAMESPACE}" >/dev/null 2>&1 || \
  kubectl create ns "${NAMESPACE}"

# -----------------------------
# Deploy / upgrade registry
# -----------------------------
echo "Deploying local registry via Helm..."
helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
  --namespace "${NAMESPACE}" \
  --set persistence.storageClass="${STORAGE_CLASS}" \
  --set persistence.size="${STORAGE_SIZE}"

# -----------------------------
# Wait for readiness
# -----------------------------
echo "Waiting for registry deployment to be ready..."
kubectl -n "${NAMESPACE}" rollout status deploy/"${RELEASE_NAME}" \
  --timeout="${WAIT_TIMEOUT}"

# -----------------------------
# Output usage info
# -----------------------------
SERVICE_NAME="${RELEASE_NAME}"

echo ""
echo "Local registry deployed successfully."
echo ""
echo "Expose registry on localhost:"
echo "  kubectl -n ${NAMESPACE} port-forward svc/${SERVICE_NAME} 5000:5000"
echo ""
echo "Push images:"
echo "  docker build -t localhost:5000/<image>:<tag> ."
echo "  docker push localhost:5000/<image>:<tag>"
echo ""
echo "Pull images inside Kubernetes using:"
echo "  ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:5000/<image>:<tag>"
