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
</div>

Goldilocks is a utility that can help you identify a starting point for resource requests and limits.

# Documentation
Check out the [documentation at docs.fairwinds.com](https://goldilocks.docs.fairwinds.com/)

## How can this help with my resource settings?

By using the kubernetes [vertical-pod-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) in recommendation mode, we can see a suggestion for resource requests on each of our apps. This tool creates a VPA for each workload in a namespace and then queries them for information.

Once your VPAs are in place, you'll see recommendations appear in the Goldilocks dashboard:
<div align="center">
<img src="/img/screenshot.png" alt="Goldilocks Screenshot" />
</div>


<!-- Begin boilerplate -->
## Join the Fairwinds Open Source Community

The goal of the Fairwinds Community is to exchange ideas, influence the open source roadmap,
and network with fellow Kubernetes users.
[Chat with us on Slack](https://join.slack.com/t/fairwindscommunity/shared_invite/zt-e3c6vj4l-3lIH6dvKqzWII5fSSFDi1g)
or
[join the user group](https://www.fairwinds.com/open-source-software-user-group) to get involved!

<a href="https://www.fairwinds.com/t-shirt-offer?utm_source=goldilocks&utm_medium=goldilocks&utm_campaign=goldilocks-tshirt">
  <img src="https://www.fairwinds.com/hubfs/Doc_Banners/Fairwinds_OSS_User_Group_740x125_v6.png" alt="Love Fairwinds Open Source? Share your business email and job title and we'll send you a free Fairwinds t-shirt!" />
</a>

## Other Projects from Fairwinds

Enjoying Goldilocks? Check out some of our other projects:
* [Polaris](https://github.com/FairwindsOps/Polaris) - Audit, enforce, and build policies for Kubernetes resources, including over 20 built-in checks for best practices
* [Pluto](https://github.com/FairwindsOps/Pluto) - Detect Kubernetes resources that have been deprecated or removed in future versions
* [Nova](https://github.com/FairwindsOps/Nova) - Check to see if any of your Helm charts have updates available
* [rbac-manager](https://github.com/FairwindsOps/rbac-manager) - Simplify the management of RBAC in your Kubernetes clusters

Or [check out the full list](https://www.fairwinds.com/open-source-software?utm_source=goldilocks&utm_medium=goldilocks&utm_campaign=goldilocks)
## Fairwinds Insights
If you're interested in running Goldilocks in multiple clusters,
tracking the results over time, integrating with Slack, Datadog, and Jira,
or unlocking other functionality, check out
[Fairwinds Insights](https://www.fairwinds.com/goldilocks-user-insights-demo?utm_source=goldilocks&utm_medium=goldilocks&utm_campaign=goldilocks),
a platform for auditing and enforcing policy in Kubernetes clusters.

<a href="https://www.fairwinds.com/goldilocks-user-insights-demo?utm_source=goldilocks&utm_medium=ad&utm_campaign=goldilocksad">
  <img src="https://www.fairwinds.com/hubfs/Doc_Banners/Fairwinds_Goldilocks_Ad.png" alt="Fairwinds Insights" />
</a>
