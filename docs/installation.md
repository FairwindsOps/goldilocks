# Installation

## Requirements

* kubectl
* [vertical-pod-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) configured in the cluster
* some workloads with pods (Goldilocks will monitor any workload controller that includes a PodSpec template (`spec.template.spec.containers[]` to be specific). This includes `Deployments`, `DaemonSets`, and `StatefulSets` among others.)
* metrics-server (a requirement of vpa)
* golang 1.17+

### Installing Vertical Pod Autoscaler

There are multiple ways to install VPA for use with Goldilocks:

* Install using the `hack/vpa-up.sh` script from the [vertical-pod-autoscaler repository](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
* Install using the [Fairwinds VPA Helm Chart](https://github.com/FairwindsOps/charts/tree/master/stable/vpa)

#### Important Note about VPA

The full VPA install includes the updater and the admission webhook for VPA. Goldilocks only requires the recommender. An admission webhook can introduce unexpected results in a cluster if not planned for properly. We recommend installing using the chart v2.0.0 or higher, which will only install the parts of VPA necessary for Goldilocks to function.

### Prometheus (optional)

[VPA](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) does not require the use of prometheus, but it is supported. The use of prometheus may provide more accurate results.

### GKE Notes

Google has provided the vertical pod autoscaler as a beta feature in GKE. You can see the docs [here](https://cloud.google.com/kubernetes-engine/docs/how-to/vertical-pod-autoscaling), or just enable it like so:

```
gcloud container clusters update [CLUSTER-NAME] --enable-vertical-pod-autoscaling {--region [REGION-NAME] | --zone [ZONE-NAME]}
```

NOTE: This does not support using prometheus as a data backend.

## Installation

First, make sure you satisfy the requirements above.

### Method 1 - Helm (preferred)

```
helm repo add fairwinds-stable https://charts.fairwinds.com/stable
Helm v2:
helm install --name goldilocks --namespace goldilocks fairwinds-stable/goldilocks
Helm v3:
helm install goldilocks --namespace goldilocks fairwinds-stable/goldilocks
```

### Method 2 - Manifests

The [hack/manifests](https://github.com/FairwindsOps/goldilocks/tree/master/hack/manifests) directory contains collections of Kubernetes YAML definitions for installing the controller and dashboard components in cluster.

```
git clone https://github.com/FairwindsOps/goldilocks.git
cd goldilocks
kubectl create namespace goldilocks
kubectl -n goldilocks apply -f hack/manifests/controller
kubectl -n goldilocks apply -f hack/manifests/dashboard
```

### Enable Namespace

Pick an application namespace and label it like so in order to see some data:

```
kubectl label ns goldilocks goldilocks.fairwinds.com/enabled=true
```

After that you should start to see VPA objects in that namespace.

### Viewing the Dashboard

The default installation creates a ClusterIP service for the dashboard. You can access via port forward:

```
kubectl -n goldilocks port-forward svc/goldilocks-dashboard 8080:80
```

Then open your browser to [http://localhost:8080](http://localhost:8080)
