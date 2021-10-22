# Advanced Usage

## CLI Usage (not recommended)

The CLI was originally developed to test the features of Goldilocks. While the CLI is still present and somewhat functional, we do not generally recommend using it. The CLI is scoped to a single namespace per run (meaning you would have to run it multiple times), and it will not automatically clean up any VPA objects that are created. The controller is a much better option, as it will monitor for changes and keep the list of VPA objects up-to-date. In addition, the controller will allow keeping multiple namespaces up to date.

The CLI summary function should still be useful. It runs the same summary output that the dashboard uses, and can generate a JSON object that can be used elsewhere.

```
A tool for analysis of kubernetes Deployment resource usage.

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
### controller

This starts the goldilocks controller. Used by the Docker container, it will create vpas for properly labelled namespaces.

#### Flags
You can set the default behavior for VPA creation using some flags. When specified, labels will always take precedence over the command line flags.

* `--on-by-default` - create VPAs in all namespaces
* `--include-namespaces` - create VPAs in these namespaces, in addition to any that are labeled
* `--exclude-namespaces` - when `--on-by-default` is set, exclude this comma-separated list of namespaces

#### Enable Namespaces

Namespaces are considered enabled or managed by goldilocks when the Namespace
has the enabled label set to "true", for example:

```
kubectl label ns goldilocks goldilocks.fairwinds.com/enabled=true
```

#### VPA Update Mode

> Note: This feature is for advanced usage only and is not recommended nor the default!

VPAs created for Deployments in a Namespace have an update mode of "off" by
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

The `resourcePolicy` section can be defined by using the following annotation on a namespace `goldilocks.fairwinds.com/vpa-update-mode=vpa-resource-policy=<json formatted section>`

Example of an annotation

```
  annotations:
    goldilocks.fairwinds.com/vpa-resource-policy: >
      { "containerPolicies": [ { "containerName": "nginx", "minAllowed": {
      "cpu": "250m", "memory": "100Mi" }, "maxAllowed": { "cpu": "2000m",
      "memory": "2048Mi" } }, { "containerName": "istio-proxy", "mode": "Off" }
      ] }
```

#### Deployment Specifications

If you want a specific Deployment to have a VPA in a specific update mode,
then you can annotate the Deployment with `goldilocks.fairwinds.com/vpa-update-mode=<mode>`
to control the update mode for a specific Deployment in a Namespace (regardless of labeling on the Namespace).

### create-vpas

`goldilocks create-vpas -n some-namespace`

This will search for any deployments in the given namespace and generate a VPA for each of them.  Each vpa will be labelled for use by this tool.

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

Containers can be excluded for individual deployments by applying a label to any deployment. The label value should be a list of comma separated container names. The label value will be combined with any values provided through the `--exclude-containers` argument.

Example label:

`kubectl label deployment myapp goldilocks.fairwinds.com/exclude-containers=linkerd-proxy,istio-proxy`
