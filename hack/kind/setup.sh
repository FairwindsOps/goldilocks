#!/bin/bash

set -e

kind_required_version=0.9.0
kind_node_image="kindest/node:v1.18.8@sha256:f4bcc97a0ad6e7abaf3f643d890add7efe6ee4ab90baeb374b4f41a4c95567eb"
install_goldilocks=${2:-true}

## Test Infra Setup
## This will use Kind, Reckoner, and Helm to setup a test infrastructure locally for goldilocks

function version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1";
}

cd "$( cd "$(dirname "$0")" ; pwd -P )"

required_clis="reckoner helm kind"
for cli in $required_clis; do
  command -v "$cli" >/dev/null 2>&1 || { echo >&2 "I require $cli but it's not installed.  Aborting."; exit 1; }
done

kind_version=$(kind version | cut -c2-)

if version_gt "$kind_required_version" "$kind_version"; then
     echo "This script requires kind version greater than or equal to $kind_required_version!"
     exit 1
fi

## Create the kind cluster

kind create cluster \
  --name test-infra \
  --image="$kind_node_image" || true

# shellcheck disable=SC2034
until kubectl cluster-info; do
    echo "Waiting for cluster to become available...."
    sleep 3
done

## Reckoner
## Installs all dependencies such as metrics-server and vpa

reckoner plot course.yml -a

if $install_goldilocks; then
  ## Install Goldilocks
  kubectl get ns goldilocks || kubectl create ns goldilocks
  kubectl -n goldilocks apply -f ../manifests/controller
  kubectl -n goldilocks apply -f ../manifests/dashboard
fi

echo "Your test environment should now be running."
