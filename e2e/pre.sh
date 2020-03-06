#!/bin/bash

set -e

curl -LO https://github.com/kubernetes-sigs/kind/releases/download/v0.7.0/kind-linux-amd64
chmod +x kind-linux-amd64
bindir=$(pwd)/bin-kind
mkdir -p "$bindir"
mv kind-linux-amd64 "$bindir/kind"
export PATH="$bindir:$PATH"

wget -O /usr/local/bin/yq "https://github.com/mikefarah/yq/releases/download/2.4.0/yq_linux_amd64"
chmod +x /usr/local/bin/yq

if [ -z "$CI_SHA1" ]; then
    echo "CI_SHA1 not set. Something is wrong"
    exit 1
else
    echo "CI_SHA1: $CI_SHA1"
fi

docker-build -f deploy/build.config
docker images
docker tag "quay.io/fairwinds/goldilocks:latest" "quay.io/fairwinds/goldilocks:$CI_SHA1"
docker images
kind load docker-image "quay.io/fairwinds/goldilocks:$CI_SHA1" --name "e2e"

yq w -i hack/manifests/dashboard/deployment.yaml spec.template.spec.containers[0].imagePullPolicy "Never"
yq w -i hack/manifests/controller/deployment.yaml spec.template.spec.containers[0].imagePullPolicy "Never"
yq w -i hack/manifests/dashboard/deployment.yaml spec.template.spec.containers[0].image "quay.io/fairwinds/goldilocks:$CI_SHA1"
yq w -i hack/manifests/controller/deployment.yaml spec.template.spec.containers[0].image "quay.io/fairwinds/goldilocks:$CI_SHA1"

cat hack/manifests/dashboard/deployment.yaml
cat hack/manifests/controller/deployment.yaml

docker cp . e2e-command-runner:/goldilocks
