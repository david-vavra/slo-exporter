#!/usr/bin/make -f 
DOCKER_COMPOSE ?= docker-compose
binary_path 	= build/$(OS)-$(ARCH)/slo_exporter
src_dir        := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all: OS=linux
all: ARCH=amd64
all: lint build test-and-coverage e2e-test

# FIXME variables no longer applicable to non gitlab-ci world
build:
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -a -ldflags "-X 'main.buildVersion=${SLO_EXPORTER_VERSION}' -X 'main.buildRevision=${CI_COMMIT_SHA}' -X 'main.buildBranch=${CI_COMMIT_BRANCH}' -X 'main.buildTag=${CI_COMMIT_TAG}' -extldflags '-static'" -o $(binary_path) $(src_dir)/cmd/slo_exporter.go

lint:
	go get github.com/mgechev/revive
	revive -formatter friendly -config .revive.toml $(shell find $(src_dir) -name "*.go" | grep -v "^$(src_dir)/vendor/")

e2e-test: OS=linux
e2e-test: ARCH=amd64
e2e-test: build
	./test/run_tests.sh $(binary_path)

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


.PHONY: build lint test test-and-coverage compose clean-compose e2e-test benchmark
