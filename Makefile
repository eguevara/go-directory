VERSION := $(shell cat VERSION)

# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

.PHONY: dep compile lint test fmt vet version
.DEFAULT: default

BUILDTAGS=

all: compile test 

dep:
	@echo "+ $@"

compile: dep
	@echo "+ $@"
	@go build -tags "$(BUILDTAGS) cgo"  ./directory/...

lint:
	@echo "+ $@"
	@golint -set_exit_status $(go list ./...)

test: fmt lint vet
	@echo "+ $@"
	@go test -cover ./...

fmt:
	@echo "+ $@"
	@gofmt -s -l . | grep -v vendor | tee /dev/stderr

vet:
	@echo "+ $@"
	@go vet ./...

version:
	@echo $(VERSION)