#!/bin/bash

## This file is intended for use by CircleCI Only!!!
## I do not recommend running it yourself. If you want to run the e2e tests,
## just use the Makefile or read the CONTRIBUTING.md

printf "\n\n"
echo "**************************"
echo "** Begin E2E Test Setup **"
echo "**************************"
printf "\n\n"

set -e

printf "\n\n"
echo "**************************"
echo "** Install Dependencies **"
echo "**************************"
printf "\n\n"

wget -O /usr/local/bin/yq "https://github.com/mikefarah/yq/releases/download/2.4.0/yq_linux_amd64"
chmod +x /usr/local/bin/yq

printf "\n\n"
echo "***************************"
echo "** Install and Run Venom **"
echo "***************************"
printf "\n\n"

curl -LO https://github.com/ovh/venom/releases/download/v0.27.0/venom.linux-amd64
mv venom.linux-amd64 /usr/local/bin/venom
chmod +x /usr/local/bin/venom

cd /goldilocks/e2e
mkdir -p /tmp/test-results
venom run tests/* --log debug --output-dir=/tmp/test-results --strict
exit $?
