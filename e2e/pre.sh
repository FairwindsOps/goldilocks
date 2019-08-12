#!/bin/bash

set -e

wget -O /usr/local/bin/yq "https://github.com/mikefarah/yq/releases/download/2.4.0/yq_linux_amd64"
chmod +x /usr/local/bin/yq

if [ -z "$CI_SHA1" ]; then
    echo "CI_SHA1 not set. Something is wrong"
    exit 1
else
    echo "CI_SHA1: $CI_SHA1"
fi

yq w -i hack/manifests/dashboard/deployment.yaml spec.template.spec.containers[0].image "quay.io/fairwinds/goldilocks:$CI_SHA1"
yq w -i hack/manifests/controller/deployment.yaml spec.template.spec.containers[0].image "quay.io/fairwinds/goldilocks:$CI_SHA1"

cat hack/manifests/dashboard/deployment.yaml
cat hack/manifests/controller/deployment.yaml

docker cp hack e2e-command-runner:/
