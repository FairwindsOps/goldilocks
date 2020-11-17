FROM golang:1.15.4 AS build-env

RUN go get -u github.com/gobuffalo/packr/v2/packr2

WORKDIR /go/src/github.com/fairwindsops/goldilocks/
COPY . .
ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 packr2 build -a -o goldilocks *.go

FROM alpine:3.12.1 as alpine

ARG KUBECTL_VERSION
ENV KUBECTL_VERSION=$KUBECTL_VERSION
ARG VAULT_VERSION
ENV VAULT_VERSION=$VAULT_VERSION

RUN apk --no-cache --update add curl bash jq ca-certificates tzdata && update-ca-certificates

# Install kubectl
RUN \
  curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
  chmod +x ./kubectl && \
  mv ./kubectl /usr/bin/kubectl

COPY run_kubectl.sh /run_kubectl.sh
RUN chmod 755 /run_kubectl.sh

# Install Vault CLI
RUN \
  curl https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip -o vault_${VAULT_VERSION}_linux_amd64.zip && \
  unzip -d /bin vault_${VAULT_VERSION}_linux_amd64 && \
  rm vault_${VAULT_VERSION}_linux_amd64.zip

ENV PATH /usr/local/bin:$PATH

COPY --from=build-env /go/src/github.com/fairwindsops/goldilocks /

WORKDIR /opt/app

ENTRYPOINT ["/bin/bash", "/run_kubectl.sh"]
CMD ["/goldilocks"]
