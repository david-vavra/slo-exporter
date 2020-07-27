#!/usr/bin/make -f 
DOCKER_COMPOSE 			?= docker-compose
#DOCKER_IMAGE_REPO		 = seznam/slo-exporter
DOCKER_IMAGE_REPO		 = sevenood/slo-exporter
SLO_EXPORTER_VERSION 	?= test
OS				= linux
ARCH			= amd64
BINARY_PATH 	= build/$(OS)-$(ARCH)/slo_exporter
src_dir        := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all: lint build test-and-coverage

build:
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -a -ldflags "-X 'main.buildVersion=${SLO_EXPORTER_VERSION}' -X 'main.buildRevision=${CIRCLE_SHA1}' -X 'main.buildBranch=${CIRCLE_BRANCH}' -X 'main.buildTag=${CIRCLE_TAG}' -extldflags '-static'" -o $(BINARY_PATH) $(src_dir)/cmd/slo_exporter.go

docker-build:
docker-build:
	docker build -t $(DOCKER_IMAGE_REPO):$(SLO_EXPORTER_VERSION) .
	docker run --rm $(DOCKER_IMAGE_REPO):$(SLO_EXPORTER_VERSION) --help

docker-push:
	docker push $(DOCKER_IMAGE_REPO):$(SLO_EXPORTER_VERSION)

lint:
	go get github.com/mgechev/revive
	revive -formatter friendly -config .revive.toml $(shell find $(src_dir) -name "*.go" | grep -v "^$(src_dir)/vendor/")

e2e-test: build
	./test/run_tests.sh

test:
	go test -v --race -coverprofile=coverage.out $(shell go list ./... | grep -v /vendor/)

benchmark: clean
	./scripts/benchmark.sh

test-and-coverage: test
	go tool cover -func coverage.out

compose: build clean-compose
	$(DOCKER_COMPOSE) up --force-recreate --renew-anon-volumes --abort-on-container-exit --remove-orphans --exit-code-from slo-exporter

clean-compose:
	$(DOCKER_COMPOSE) rm --force --stop -v
	docker volume rm slo-exporter_log-volume || true

clean:
	rm -rf slo_exporter coverage.out profile
	find . -type f -name "*.pos" -prune -exec rm -f {} \;
	find . -type d -name "test_output" -prune -exec rm -rf {} \;


.PHONY: build lint test test-and-coverage compose clean-compose e2e-test benchmark docker
