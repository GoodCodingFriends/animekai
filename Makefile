SHELL := /bin/bash

TOOLS=$(shell cat tools/tools.go | egrep '^\s_ '  | awk '{ print $$2 }')

.PHONY: proto
proto:
	protoc --go_opt=paths=source_relative -I proto -I $(GOPATH)/src/github.com/protocolbuffers/protobuf/src/ -I $(GOPATH)/src/github.com/googleapis/googleapis --go_out=plugins=grpc:api --gohttp_out=. proto/api.proto
	protoc --go_opt=paths=source_relative -I proto -I $(GOPATH)/src/github.com/protocolbuffers/protobuf/src/ -I $(GOPATH)/src/github.com/googleapis/googleapis --go_out=resource proto/resource.proto

.PHONY: tools
tools:
	GOBIN=$(PWD)/bin go install -mod readonly $(TOOLS)

.PHONY: build
build:
	go build -o bin/server ./cmd/server/*.go

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test: build lint
	go build ./...
