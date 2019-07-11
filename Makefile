# Go parameters
GOCMD=go
GOBUILD=GO111MODULE=on $(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=GO111MODULE=on $(GOCMD) test
BINARY_NAME=vpa-analysis
COMMIT := $(shell git rev-parse HEAD)
VERSION := "dev"

all: test build
build:
	$(GOBUILD) -o $(BINARY_NAME) -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -s -w" -v
test:
	printf "Linter:\n"
	GO111MODULE=on $(GOCMD) list ./... | xargs -L1 golint | tee golint-report.out
	printf "\n\nTests:\n\n"
	GO111MODULE=on $(GOCMD) test -v --bench --benchmem -coverprofile cover-report.out ./...
	GO111MODULE=on $(GOCMD) vet 2> govet-report.out
	GO111MODULE=on $(GOCMD) tool cover -html=cover-report.out -o cover-report.html
	printf "\nCoverage report available at cover-report.html\n\n"

clean:
	$(GOCLEAN)
	$(GOCMD) fmt ./...
	rm -f $(BINARY_NAME)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME) -ldflags "-X main.VERSION=$(VERSION)" -v

build-docker:
	docker build -t vpa-analysis:dev .
