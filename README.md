# VPA Analysis

A tool to quickly summarize suggestions for resource requests in your Kubernetes cluster.

## How?

By using the kubernetes vertical-pod-autoscaler in recommendation mode, we can se a suggestion for resource requests on each of our apps.  This tool just creates a bunch of VPAs and then queries them for information.

## Requirements

* kubectl
* vertical-pod-autoscaler configured in the cluster
* some deployments with pods
* metrics-server (a requirement of vpa)
* golang 1.11+

## Usage

```
Usage:
  vpa-analysis [flags]
  vpa-analysis [command]

Available Commands:
  create-vpas Create VPAs
  help        Help about any command
  summary     Genarate a summary of the vpa recommendations in a namespace.
  version     Prints the current version of the tool.

Flags:
      --alsologtostderr                  log to standard error as well as files
  -h, --help                             help for vpa-analysis
      --kubeconfig string                Kubeconfig location. [KUBECONFIG] (default "$HOME/.kube/config")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default false)
  -n, --namespace string                 Namespace to install the VPA objects in. (default "default")
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging

Use "vpa-analysis [command] --help" for more information about a command.
```

### create-vpas

```vpa-analysis create-vpas -n some-namespace```

This will search for any deployments in the given namespace and generate a VPA for each of them.  Each vpa will be labelled for use by this tool.

### summary

```vpa-analysis summary -n some-namespace```

Queries all the VPA objects that are labelled for this tool and summarizes their suggestions.

### Development

Look at the Makefile.  It works sometimes.
