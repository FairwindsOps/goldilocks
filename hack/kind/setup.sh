#!/bin/bash

set -e

kind_required_version=0.8.1
kind_node_image="kindest/node:v1.17.5@sha256:ab3f9e6ec5ad8840eeb1f76c89bb7948c77bbf76bcebe1a8b59790b8ae9a283a"
vertical_pod_autoscaler_ref=e0f63c1caeec518f85c4347b673e4e99e4fb0059
install_vpa=${1:-true}
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

if $install_vpa; then
  ## Install VPA

  if [ ! -d "autoscaler" ] ; then
      git clone "https://github.com/kubernetes/autoscaler.git"
  fi

  cd autoscaler/vertical-pod-autoscaler
  git checkout "$vertical_pod_autoscaler_ref"
  ./hack/vpa-up.sh

  cd ../../
fi

## Reckoner

reckoner plot course.yml -a

if $install_goldilocks; then
  ## Install Goldilocks
  kubectl get ns goldilocks || kubectl create ns goldilocks
  kubectl -n goldilocks apply -f ../manifests/controller
  kubectl -n goldilocks apply -f ../manifests/dashboard
fi

echo "Your test environment should now be running."
