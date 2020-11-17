# Docker build related
DOCKER_REPO       ?= maerskao.azurecr.io
IMAGE             ?= maerskao.azurecr.io/goldilocks
APP_NAME          ?= goldilocks
VERSION           ?= v3.0.0
GIT_COMMIT_SHORT  ?= $(shell git log -1 --pretty=format:"%h")

# k8s deploy related
KUBECTL_VERSION=v1.19.3
VAULT_VERSION=1.5.0

.EXPORT_ALL_VARIABLES:

# docker build related
build:
	docker build --compress --pull --rm -f Dockerfile --build-arg "KUBECTL_VERSION=$(KUBECTL_VERSION)" --build-arg "VAULT_VERSION=$(VAULT_VERSION)" -t $(APP_NAME):latest .

# Docker publish
publish: publish-latest publish-version publish-git-hash ## Publish all tagged containers

publish-latest: tag-latest ## Publish the latest tagged
	docker push ${DOCKER_REPO}/${APP_NAME}:latest

publish-version: tag-version ## Publish the version tagged container
	docker push ${DOCKER_REPO}/${APP_NAME}:${VERSION}

publish-git-hash: tag-git-hash ## Publish the git hash tagged container
	docker push ${DOCKER_REPO}/${APP_NAME}:${GIT_COMMIT_SHORT}

# Docker tagging
tag: tag-latest tag-version tag-git-hash ## Generate container tags for all tags

tag-latest: ## Generate container latest tag
	docker tag ${APP_NAME}:latest ${DOCKER_REPO}/${APP_NAME}:latest

tag-version: ## Generate container version tag
	docker tag ${APP_NAME}:latest ${DOCKER_REPO}/${APP_NAME}:${VERSION}

tag-git-hash: ## Generate container git hash tag
	docker tag ${APP_NAME}:latest ${DOCKER_REPO}/${APP_NAME}:${GIT_COMMIT_SHORT}
