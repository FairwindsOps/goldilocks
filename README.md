# goldilocks [![CircleCI](https://circleci.com/gh/FairwindsOps/goldilocks.svg?style=svg&circle-token=affdde2880ec2669f26be783f3f9e412b0d2fb62)](https://circleci.com/gh/FairwindsOps/goldilocks) [![codecov](https://codecov.io/gh/FairwindsOps/goldilocks/branch/master/graph/badge.svg?token=jkXRJcqr49)](https://codecov.io/gh/FairwindsOps/goldilocks) [![Apache 2.0 license](https://img.shields.io/badge/licence-Apache2-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)

Get your resource requests "Just Right"

## How?

By using the kubernetes [vertical-pod-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) in recommendation mode, we can se a suggestion for resource requests on each of our apps. This tool creates a VPA for each deployment in a namespace and then queries them for information.

## Requirements

* kubectl
* [vertical-pod-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) configured in the cluster
* some deployments with pods
* metrics-server (a requirement of vpa)
* golang 1.11+

### GKE

Google has provided the vertical pod autoscaler as a beta feature in GKE. You can see the docs [here](https://cloud.google.com/kubernetes-engine/docs/how-to/vertical-pod-autoscaling), or just enable it like so:

```
gcloud beta container clusters update [CLUSTER-NAME] --enable-vertical-pod-autoscaling
```

NOTE: This does not support using prometheus as a data backend.

## Recommended Requirements

[VPA](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) does not require the use of prometheus, but it is supported. In order to take long-term data into account, we recommend that you install prometheus and configure your vertical pod autoscaler install to use it.

## Installation (Quickstart)

First, make sure you satisfy the requirements above.

The [hack/manifests](hack/manifests) directory contains collections of Kubernetes YAML definitions for installing the controller and dashboard components in cluster.

```
kubectl create namespace goldilocks
kubectl apply -f hack/manifests/controller
kubectl apply -f hack/manifests/dashboard
```

Now pick an application namespace and label it like so:

```
kubectl label ns goldilocks fairwinds.com/goldilocks=true
```

You should see start to see VPA objects in that namespace. Then you can checkout the dashboard:

```
kubectl -n goldilocks port-forward svc/goldilocks-dashboard 8080:80
```

Then open your browser to [http://localhost:8080](http://localhost:8080)

## Usage

```
A tool for analysis of kubernetes deployment resource usage.

Usage:
  goldilocks [flags]
  goldilocks [command]

Available Commands:
  controller  Run goldilocks as a controller inside a kubernetes cluster.
  create-vpas Create VPAs
  dashboard   Run the goldilocks dashboard that will show recommendations.
  delete-vpas Delete VPAs
  help        Help about any command
  summary     Genarate a summary of the vpa recommendations in a namespace.
  version     Prints the current version of the tool.

Flags:
      --alsologtostderr                  log to standard error as well as files
  -h, --help                             help for goldilocks
      --kubeconfig string                Kubeconfig location. [KUBECONFIG] (default "$HOME/.kube/config")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --log_file string                  If non-empty, use this log file
      --log_file_max_size uint           Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --master string                    The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging

Use "goldilocks [command] --help" for more information about a command.
```

### create-vpas

`goldilocks create-vpas -n some-namespace`

This will search for any deployments in the given namespace and generate a VPA for each of them.  Each vpa will be labelled for use by this tool.

### dashboard

`goldilocks dashboard`

Runs the goldilocks dashboard web server that will display recommendations. Listens on port `8080` by default.

### summary

`goldilocks summary`

Queries all the VPA objects that are labelled for this tool and summarizes their suggestions into a JSON object.

### Container Exclusions

The `dashboard` and `summary` commands can exclude recommendations for a list of comma separate container names using the `--excludes-containers` argument. This option can be useful for hiding recommendations for sidecar containers for things like Linkerd and Istio.

Containers can be excluded for individual deployments by applying an annotation to any deployment. The annotation value should be a list of comma separated container names. The annotation value will be combined with any values provided through the `--excludes-containers` argument.

`goldilocks.fairwinds.com/exclude-container: linkerd-proxy`
