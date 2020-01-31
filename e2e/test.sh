#!/bin/bash

vertical_pod_autoscaler_ref=e16a0adef6c7d79a23d57f9bbbef26fc9da59378
timeout=60s
reinstall_wait=30s

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

apk --update add openssl
wget -O /usr/local/bin/yq "https://github.com/mikefarah/yq/releases/download/2.4.0/yq_linux_amd64"
chmod +x /usr/local/bin/yq

printf "\n\n"
echo "****************************"
echo "** Install Metrics-Server **"
echo "****************************"
printf "\n\n"

helm repo add stable https://kubernetes-charts.storage.googleapis.com
helm repo update
kubectl create namespace metrics-server
helm install metrics-server stable/metrics-server --version=2.9.0 --set args[0]=--kubelet-insecure-tls --set args[1]=--kubelet-preferred-address-types=InternalIP --namespace metrics-server

printf "\n\n"
echo "****************************"
echo "** Install VPA Controller **"
echo "****************************"
printf "\n\n"

if [ ! -d "autoscaler" ] ; then
    git clone "https://github.com/kubernetes/autoscaler.git"
fi

cd autoscaler/vertical-pod-autoscaler
git checkout "$vertical_pod_autoscaler_ref"
./hack/vpa-up.sh

cd ../../

echo "Ensure the CRD is available"
until kubectl get crd verticalpodautoscalers.autoscaling.k8s.io &> /dev/null ; do
    echo "Waiting for vpas to become available..."
    sleep 2
done

printf "\n\n"
echo "********************************************************************"
echo "** Install Goldilocks at $CI_SHA1 **"
echo "********************************************************************"
printf "\n\n"

kubectl create ns goldilocks

kubectl -n goldilocks apply -f /hack/manifests/dashboard/
kubectl -n goldilocks apply -f /hack/manifests/controller/

kubectl get all -n goldilocks

kubectl -n goldilocks wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=goldilocks,app.kubernetes.io/component=dashboard
kubectl -n goldilocks wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=goldilocks,app.kubernetes.io/component=controller

kubectl get po --all-namespaces

printf "\n\n"
echo "**********************"
echo "** Install Demo App **"
echo "**********************"
printf "\n\n"

helm repo add fairwinds-incubator https://charts.fairwinds.com/incubator
kubectl create ns demo
kubectl create ns demo-no-label
kubectl create ns demo-included
kubectl create ns demo-excluded
helm install basic-demo fairwinds-incubator/basic-demo --namespace demo --version=0.2.3
helm install basic-demo-no-label fairwinds-incubator/basic-demo --namespace demo-no-label --version=0.2.3
helm install basic-demo-included fairwinds-incubator/basic-demo --namespace demo-included --version=0.2.3
helm install basic-demo-excluded fairwinds-incubator/basic-demo --namespace demo-excluded --version=0.2.3

kubectl -n demo wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=basic-demo
kubectl -n demo-no-label wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=basic-demo
kubectl -n demo-included wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=basic-demo
kubectl -n demo-excluded wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=basic-demo

printf "\n\n"
echo "**********************"
echo "** Run a Basic Test **"
echo "**********************"
printf "\n\n"

echo "** Existing VPAs: "
kubectl get verticalpodautoscalers.autoscaling.k8s.io --all-namespaces -owide

echo
echo "** Label the Namespace"
kubectl label ns demo goldilocks.fairwinds.com/enabled=true --overwrite
sleep $reinstall_wait

echo
echo "** New VPAs: "
kubectl get verticalpodautoscalers.autoscaling.k8s.io -n demo basic-demo

printf "\n\n"
echo "****************************"
echo "** Run on-by-default test **"
echo "****************************"
printf "\n\n"

yq w -i /hack/manifests/controller/deployment.yaml -- spec.template.spec.containers[0].command[2] '--on-by-default'
kubectl -n goldilocks apply -f /hack/manifests/controller/
kubectl -n goldilocks wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=goldilocks,app.kubernetes.io/component=controller
sleep $reinstall_wait

echo "** No-label VPAs: "
kubectl get verticalpodautoscalers.autoscaling.k8s.io -n demo-no-label basic-demo-no-label

printf "\n\n"
echo "*********************************"
echo "** Run include-namespaces test **"
echo "*********************************"
printf "\n\n"

yq w -i /hack/manifests/controller/deployment.yaml -- spec.template.spec.containers[0].command[2] '--include-namespaces=demo-included'
kubectl -n goldilocks apply -f /hack/manifests/controller/
kubectl -n goldilocks wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=goldilocks,app.kubernetes.io/component=controller
sleep $reinstall_wait

echo "** Included VPAs: "
kubectl get verticalpodautoscalers.autoscaling.k8s.io -n demo-included basic-demo-included

printf "\n\n"
echo "*********************************"
echo "** Run exclude-namespaces test **"
echo "*********************************"
printf "\n\n"

yq w -i /hack/manifests/controller/deployment.yaml -- spec.template.spec.containers[0].command[2] '--on-by-default'
yq w -i /hack/manifests/controller/deployment.yaml -- spec.template.spec.containers[0].command[3] '--exclude-namespaces=demo-excluded'
kubectl -n goldilocks apply -f /hack/manifests/controller/
kubectl -n goldilocks wait deployment --timeout=$timeout --for condition=available -l app.kubernetes.io/name=goldilocks,app.kubernetes.io/component=controller
sleep $reinstall_wait

echo "** Excluded VPAs: "
kubectl get verticalpodautoscalers.autoscaling.k8s.io -n demo-excluded
if kubectl get verticalpodautoscalers.autoscaling.k8s.io -n demo-excluded basic-demo-excluded; then
  echo "Found VPA on demo-excluded when it should have been excluded"
  exit 1
fi
