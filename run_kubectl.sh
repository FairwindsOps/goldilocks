#!/usr/bin/env bash

if [ -z ${VAULT_TOKEN} ]; then
	printf "error VAULT_TOKEN is missing!\n"
	exit 1
fi
if [[ ! -z ${DEBUG} ]]; then
	set -x
fi

KUBE_CONFIGS_FOLDER=/tmp/configs
mkdir -p $KUBE_CONFIGS_FOLDER
#mkdir -p /$HOME/.kube
mkdir -p /.kube

API_ADDR=https://platform-api.maersk-digital.net
VAULT_ADDR=https://vault.maersk-digital.net
VAULT_RESOURCE=https://vault.maersk-digital.net/v1/platform-kv/data/readable/platform-api/prod/serviceprincipal
CURL_RETRY="--connect-timeout 5 --max-time 10 --retry 5 --retry-delay 0 --retry-max-time 40"


set_context() {
  local CLUSTER_NAME=$1
  local KUBECONFIG="$KUBE_CONFIGS_FOLDER/${CLUSTER_NAME}"
  export KUBECONFIG
  echo $KUBECONFIG
  # Get k8s secrets from vault
  KUBERNETES_SERVER=$(curl $CURL_RETRY --silent -H "X-Vault-Token: ${VAULT_TOKEN}" -X GET ${VAULT_ADDR}/v1/platform-kv/data/readable/orchestration/kubernetes/${CLUSTER_NAME}/server | jq -r ".data.data.server")
  curl $CURL_RETRY --silent -H "X-Vault-Token: ${VAULT_TOKEN}" -X GET ${VAULT_ADDR}/v1/platform-kv/data/readable/orchestration/kubernetes/${CLUSTER_NAME}/cert | jq -r ".data.data.cert" | base64 -d > /tmp/${CLUSTER_NAME}-ca.crt
  KUBERNETES_TOKEN=$(curl $CURL_RETRY --silent -H "X-Vault-Token: ${VAULT_TOKEN}" -X GET ${VAULT_ADDR}/v1/platform-kv/data/readable/orchestration/kubernetes/${CLUSTER_NAME}/token | jq -r ".data.data.token")
  kubectl config set-credentials ${CLUSTER_NAME} --token=${KUBERNETES_TOKEN}
  cp /tmp/${CLUSTER_NAME}-ca.crt /usr/local/share/ca-certificates/${CLUSTER_NAME}.crt
  kubectl config set-cluster ${CLUSTER_NAME} --server=${KUBERNETES_SERVER} --certificate-authority=/tmp/${CLUSTER_NAME}-ca.crt
  kubectl config set-context ${CLUSTER_NAME} --cluster=${CLUSTER_NAME} --user=${CLUSTER_NAME}
}


merge_kube_configs() {
  local CLUSTERS=$1
  for i in $CLUSTERS; do
          CONFIG_STRING+="${KUBE_CONFIGS_FOLDER}/$i:"
  done
	KUBECONFIG=$CONFIG_STRING kubectl config view --flatten > /.kube/config
}

get_k8s_cluster() {
  CLIENT_SECRET=$(curl -s -H "X-Vault-Token: ${VAULT_TOKEN}" -X GET ${VAULT_RESOURCE} | jq -r '.data.data.client_secret')
  CLIENT_ID=$(curl -s -H "X-Vault-Token: ${VAULT_TOKEN}" -X GET ${VAULT_RESOURCE} | jq -r '.data.data.client_id')
  SCOPE=$(curl -s -H "X-Vault-Token: ${VAULT_TOKEN}" -X GET ${VAULT_RESOURCE} | jq -r '.data.data.scope')
  BEARER_TOKEN=$(curl -s -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "client_id=${CLIENT_ID}&scope=${SCOPE}&client_secret=${CLIENT_SECRET}&grant_type=client_credentials" 'https://login.microsoftonline.com/maersk.onmicrosoft.com/oauth2/v2.0/token' | jq -r .access_token)
  CLUSTERS=$(curl -s -X GET "${API_ADDR}/clusters" -H "Authorization: Bearer ${BEARER_TOKEN}" -H "accept: application/json" -H "Content-Type: application/json"| jq -r .cluster[].cluster_name)
  export CLUSTERS
}

# Get k8s secrets from vault for tooling cluster
get_k8s_cluster
echo "Clusters: $CLUSTERS"
while read cluster; do
	set_context $cluster
done <<<"$CLUSTERS"
wait

merge_kube_configs "$CLUSTERS"

update-ca-certificates

exec "$@"

