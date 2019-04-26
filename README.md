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

### create-vpas

```vpa-analysis create-vpas -n some-namespace```

This will search for any deployments in the given namespace and generate a VPA for each of them.  Each vpa will be labelled for use by this tool.

### summary

```vpa-analysis summary -n some-namespace```

Queries all the VPA objects that are labelled for this tool and summarizes their suggestions.

### Development

Look at the Makefile.  It works sometimes.
