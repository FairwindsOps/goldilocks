<div align="center" class="no-border">
    <img src="/img/goldilocks.svg" height="150" alt="Goldilocks" style="padding-bottom: 20px" />
    <br>
    <h3>Get your resource requests "Just Right"</h3>
    <a href="https://github.com/FairwindsOps/goldilocks/releases">
        <img src="https://img.shields.io/github/v/release/FairwindsOps/goldilocks">
    </a>
    <a href="https://goreportcard.com/report/github.com/FairwindsOps/goldilocks">
        <img src="https://goreportcard.com/badge/github.com/FairwindsOps/goldilocks">
    </a>
    <a href="https://circleci.com/gh/FairwindsOps/goldilocks.svg">
        <img src="https://circleci.com/gh/FairwindsOps/goldilocks.svg?style=svg">
    </a>
    <a href="https://codecov.io/gh/FairwindsOps/goldilocks">
        <img src="https://codecov.io/gh/FairwindsOps/goldilocks/branch/master/graph/badge.svg">
    </a>
</div>

Goldilocks is a utility that can help you identify a starting point for resource requests and limits.

# Documentation
Check out the [documentation at docs.fairwinds.com](goldilocks.docs.fairwinds.com)

## How can this help with my resource settings?

By using the kubernetes [vertical-pod-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) in recommendation mode, we can see a suggestion for resource requests on each of our apps. This tool creates a VPA for each deployment in a namespace and then queries them for information.

Once your VPAs are in place, you'll see recommendations appear in the Goldilocks dashboard:
<div align="center">
<img src="/img/screenshot.png" alt="Goldilocks Screenshot" />
</div>
