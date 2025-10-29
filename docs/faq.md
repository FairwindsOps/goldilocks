---
meta:
  - name: description
    content: "We get a lot of questions about how Goldilocks works and where it gets the recomendations. Hopefully we can answer the most common ones here"
---
# Frequently Asked Questions

We get a lot of questions about how Goldilocks works and where it gets the recomendations. Hopefully we can answer the most common ones here.

## How does Goldilocks generate recommendations?

Goldilocks doesn't do any recommending of resource requests/limits by itself. It utilizes a Kubernetes project called [Vertical Pod Autoscaler (VPA)](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler). More specifically, it uses the [Recommender](https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/docs/components.md#recommender) portion of the VPA.

According to the VPA documentation:

```
After starting the binary, recommender reads the history of running pods and their usage from Prometheus into the model. It then runs in a loop and at each step performs the following actions:

update model with recent information on resources (using listers based on watch),
update model with fresh usage samples from Metrics API,
compute new recommendation for each VPA,
put any changed recommendations into the VPA resources.
```

This means that recommendations are generated based on historical usage of the pod over time.

## Which values from the VPA are used?

There are two types of recommendations that Goldilocks shows in the dashboard. They are based on Kubernetes [QoS Classes](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/)

A VPA recommendation section looks like this:

```
  recommendation:
    containerRecommendations:
    - containerName: basic-demo
      lowerBound:
        cpu: 10m
        memory: "26214400"
      target:
        cpu: 11m
        memory: "26214400"
      uncappedTarget:
        cpu: 11m
        memory: "26214400"
      upperBound:
        cpu: 12m
        memory: "26214400"
```

We generate two different QoS classes of recommendation from this

* For `Guaranteed`, we take the `target` field from the recommendation and set that as both the request and limit for that container
* For `Burstable`, we set the request as the `lowerBound` and the limit as the `upperBound` from the VPA object

## How Accurate is Goldilocks?

This is entirely based on the underlying VPA project. However, in our experience Goldilocks has usually been a good _starting point_ for setting your resource requests and limits. Every environment will be different, and Goldilocks is not a replacement for tuning your applications for your specific use-case.

## I see incoherent recommendations for my limits like 100T for memory or 100G for CPU, what gives?

This situation can happen if you look at the recommendations very shortly after starting your workload.
Indeed, the statistical model used in the VPA recommender needs 8 days of historic data to produce recommendations and upper/lower boundaries with maximum accuracy. In the time between starting a workload for the first time and these 8 days, the boundaries will become more and more accurate. The lowerBound converges much quicker to maximum accuracy than the upperBound: the idea is that upscaling can be done much more liberally than downscaling. 
If you see an upperBound value which is incredibly high, it is the maximum possible value for the VPA recommender's statistical model.
TL;DR: wait a little bit to have more accurate values.

## I don't see any VPA objects getting created, what gives?

There's two main possibilities here:

* You have not labelled any namespaces for use by goldilocks. Try `kubectl label ns <namespace-name> goldilocks.fairwinds.com/enabled=true`
* VPA is not installed. The goldilocks logs will indicate if this is the case.

## I am not getting any recommendations, what gives?

The first thing to do is wait a few minutes. The VPA recommender takes some time to populate data.

The next most common cause of this is that metrics server is not running, or the metrics api-service isn't working, so VPA cannot provide any recommendations. There are a couple of things you can check.

### Check that the metrics apiservice is available:

This indicates an issue:
```
▶ kubectl get apiservice v1beta1.metrics.k8s.io
NAME                     SERVICE                         AVAILABLE                  AGE
v1beta1.metrics.k8s.io   metrics-server/metrics-server   False (MissingEndpoints)   7s
```

This shows a healthy metrics service:
```
▶ kubectl get apiservice v1beta1.metrics.k8s.io
NAME                     SERVICE                         AVAILABLE   AGE
v1beta1.metrics.k8s.io   metrics-server/metrics-server   True        36s
```
