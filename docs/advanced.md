---
meta:
  - name: description
    content: "Goldilocks is a utility that can help you identify a starting point for resource requests and limits. Here are some advanced usages."
---
# Advanced Usage

## CLI Usage (not recommended)

The CLI was originally developed to test the features of Goldilocks. While the CLI is still present and somewhat functional, we do not generally recommend using it. The CLI is scoped to a single namespace per run (meaning you would have to run it multiple times), and it will not automatically clean up any VPA objects that are created. The controller is a much better option, as it will monitor for changes and keep the list of VPA objects up-to-date. In addition, the controller will allow keeping multiple namespaces up to date.

The CLI summary function should still be useful. It runs the same summary output that the dashboard uses, and can generate a JSON object that can be used elsewhere.

```
A tool for analysis of kubernetes workload resource usage.

Usage:
  goldilocks [flags]
  goldilocks [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  controller  Run goldilocks as a controller inside a kubernetes cluster.
  create-vpas Create VPAs
  dashboard   Run the goldilocks dashboard that will show recommendations.
  delete-vpas Delete VPAs
  help        Help about any command
  summary     Generate a summary of vpa recommendations.
  version     Prints the current version of the tool.

Flags:
  -h, --help                help for goldilocks
      --kubeconfig string   Kubeconfig location. [KUBECONFIG] (default "$HOME/.kube/config")
  -v, --v Level             number for the log level verbosity

Use "goldilocks [command] --help" for more information about a command.
```

### Installation

Visit the [releases page](https://github.com/FairwindsOps/goldilocks/releases) to find the release
that's right for your environment. For example, on Linux:
```
curl -L "https://github.com/FairwindsOps/goldilocks/releases/download/v4.0.0/goldilocks_4.0.0_linux_amd64.tar.gz" > goldilocks.tar.gz
tar -xvf goldilocks.tar.gz
sudo mv goldilocks /usr/local/bin/
```

### controller

This starts the goldilocks controller. Used by the Docker container, it will create vpas for properly labelled namespaces.

#### Flags
You can set the default behavior for VPA creation using some flags. When specified, labels will always take precedence over the command line flags.

* `--on-by-default` - create VPAs in all namespaces
* `--include-namespaces` - create VPAs in these namespaces, in addition to any that are labeled
* `--exclude-namespaces` - when `--on-by-default` is set, exclude this comma-separated list of namespaces
* `--ignore-controller-kind` - comma-separated list of controller kinds to ignore from automatic VPA creation. For example: `--ignore-controller-kind=Job,CronJob`

#### Enable Namespaces

Namespaces are considered enabled or managed by goldilocks when the Namespace
has the enabled label set to "true", for example:

```
kubectl label ns goldilocks goldilocks.fairwinds.com/enabled=true
```

#### VPA Update Mode

> Note: This feature is for advanced usage only and is not recommended nor the default!

VPAs created for workloads in a Namespace have an update mode of "off" by
default, meaning the VPAs only report recommendations and do not actually
auto-scale the Pods.

The update mode can be changed for a namespace by labels as well, for example:

```
kubectl label ns goldilocks goldilocks.fairwinds.com/vpa-update-mode="auto"
```

#### VPA Resource Policy

> Note: This feature is for advanced usage only and is not recommended nor the default!

The `resourcePolicy` section of a VPA allows detailed specifications per container.  This provides the ability to configure `minAllowed` and `maxAllowed` `cpu` and `memory` values, override the `updateMode` by specifying the `mode` and setting `controlledValues` with can be either `RequestsAndLimits` (default) or `RequestsOnly` which allows the VPA to only adjust Requests and/or Limits depending on the this setting.

Example of a `resourcePolicy` section

```
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: nginx-vpa
spec:
  targetRef:
    apiVersion: "apps/v1"
    kind:       Deployment
    name:       nginx
  updatePolicy:
    updateMode: "On"
  resourcePolicy:
    containerPolicies:
    - containerName: "nginx"
      minAllowed:
        cpu: "250m"
        memory: "100Mi"
      maxAllowed:
        cpu: "2000m"
        memory: "2048Mi"
    - containerName: "istio-proxy"
      mode: "Off"
```

The `resourcePolicy` section can be defined by using the following annotation on a namespace `goldilocks.fairwinds.com/vpa-resource-policy=<json formatted section>`

Example of an annotation

```
  annotations:
    goldilocks.fairwinds.com/vpa-resource-policy: >
      { "containerPolicies": [ { "containerName": "nginx", "minAllowed": {
      "cpu": "250m", "memory": "100Mi" }, "maxAllowed": { "cpu": "2000m",
      "memory": "2048Mi" } }, { "containerName": "istio-proxy", "mode": "Off" }
      ] }
```

#### Workload Specifications

If you want a specific workload to have a VPA in a specific update mode,
then you can annotate the workload with `goldilocks.fairwinds.com/vpa-update-mode=<mode>`
to control the update mode for a specific workload in a Namespace (regardless of labeling on the Namespace).

### create-vpas

`goldilocks create-vpas -n some-namespace`

This will search for any workloads in the given namespace and generate a VPA for each of them.  Each vpa will be labelled for use by this tool.

### delete-vpas

This will delete all vpa objects in a namespace that are labelled for use by this tool.

### dashboard

`goldilocks dashboard`

Runs the goldilocks dashboard server that will display recommendations. Listens on port `8080` by default.

### summary

`goldilocks summary`

Queries all the VPA objects that are labelled for this tool across all namespaces and summarizes their suggestions into a JSON object.

### Container Exclusions

The `dashboard` and `summary` commands can exclude recommendations for a list of comma separated container names using the `--exclude-containers` argument. This option can be useful for hiding recommendations for sidecar containers for things like Linkerd and Istio.

Containers can be excluded for individual workloads by applying a label to any of the workload controller resources (`Deployment`, `StatefulSet`, `DaemonSet`, etc). The label value should be a list of comma separated container names. The label value will be combined with any values provided through the `--exclude-containers` argument.

Example label:

`kubectl label deployment myapp goldilocks.fairwinds.com/exclude-containers=linkerd-proxy,istio-proxy`


## API Usage

Goldilocks has an API endpoint that returns the VPA Summary.

You can access the API at `/api/:namespace`.
