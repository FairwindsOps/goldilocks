---
meta:
  - name: description
    content: "Issues are essential for keeping goldilocks great. There are a few guidelines that we need contributors to follow so that we can keep on top of things"
---
# Contributing

Issues, whether bugs, tasks, or feature requests are essential for keeping goldilocks great. We believe it should be as easy as possible to contribute changes that get things working in your environment. There are a few guidelines that we need contributors to follow so that we can keep on top of things.

## Code of Conduct

This project adheres to a [code of conduct](/contributing/code-of-conduct). Please review this document before contributing to this project.

## Sign the CLA
Before you can contribute, you will need to sign the [Contributor License Agreement](https://cla-assistant.io/fairwindsops/goldilocks).

## Project Structure

Goldilocks can run in 3 modes.  There is a CLI that allows the manipulation of VPA objects, a dashboard, and a controller that runs in-cluster and manages VPA objects based on namespace labels. The CLI uses the cobra package and the commands are in the `cmd` folder.

## Getting Started

We label issues with the ["good first issue" tag](https://github.com/FairwindsOps/goldilocks/labels/good%20first%20issue) if we believe they'll be a good starting point for new contributors. If you're interested in working on an issue, please start a conversation on that issue, and we can help answer any questions as they come up.

## Pre-commit

This repo contains a pre-commit file for use with [pre-commit](https://pre-commit.com/). Just run `pre-commit install` and you will have the hooks.

## Setting Up Your Development Environment

### Using Kind

Make sure you have the following installed:

* [kind 0.9.0](https://github.com/kubernetes-sigs/kind/releases) or higher
* [reckoner v1.4.0](https://github.com/FairwindsOps/reckoner/releases) or higher
* [helm 2.13.1](https://github.com/helm/helm/releases) or higher
* git
* kubectl

Go into the [/hack/kind](https://github.com/FairwindsOps/goldilocks/tree/master/hack/kind) directory and run `./setup.sh`

This will create a kind cluster, place a demo app, install VPA, and install the latest goldilocks. You can run your local development against this cluster.

### Using your own cluster

Prerequisites:

* A properly configured Golang environment with Go 1.11 or higher
* If you want to see the local changes you make on a dashboard, you will need access to a Kubernetes cluster defined in `~/.kube/config` or the KUBECONFIG variable.
* The [vertical pod autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) will need to be installed in the cluster.

### Installation
* Install the project with `go get github.com/fairwindsops/goldilocks`
* Change into the goldilocks directory which is installed at `$GOPATH/src/github.com/fairwindsops/goldilocks`
* Use `make tidy` or `make build` to ensure all dependencies are downloaded.
* See the dashboard with `go run main.go dashboard`, then open http://localhost:8080/.  This assumes that you have a working KUBECONFIG in place with access to a cluster.

### End-To-End Tests

The e2e tests run using [Venom](https://github.com/ovh/venom). You can run them yourself by:

- installing venom
- setting up a kind cluster `kind create cluster`
- running `make e2e-test`.

The tests are also run automatically by CI

You can add tests in the [e2e/tests](https://github.com/FairwindsOps/goldilocks/tree/master/e2e/tests) directory. See the Venom README for more info.

## Creating a New Issue

If you've encountered an issue that is not already reported, please create an issue that contains the following:

- Clear description of the issue
- Steps to reproduce it
- Appropriate labels

## Creating a Pull Request

Each new pull request should:

- Reference any related issues
- Add tests that show the issues have been solved
- Pass existing tests and linting
- Contain a clear indication of if they're ready for review or a work in progress
- Be up to date and/or rebased on the master branch

## Creating a new release

Push a new annotated tag.  This tag should contain a changelog of pertinent changes.
