SHELL := /bin/bash

REGISTRY ?= "gpay-gacha"
CLOUD_RUN_REGION ?= "us-central1"

TOOLS=$(shell cat tools/tools.go | egrep '^\s_ '  | awk '{ print $$2 }')

.PHONY: proto
proto:
	protoc --go_opt=paths=source_relative -I proto -I $(GOPATH)/src/github.com/protocolbuffers/protobuf/src/ -I $(GOPATH)/src/github.com/googleapis/googleapis --go_out=plugins=grpc:api --gohttp_out=api proto/api.proto
	protoc --go_opt=paths=source_relative -I proto -I $(GOPATH)/src/github.com/protocolbuffers/protobuf/src/ -I $(GOPATH)/src/github.com/googleapis/googleapis --go_out=resource proto/resource.proto

.PHONY: graphql
graphql:
	gqlgenc

.PHONY: tools
tools:
	GOBIN=$(PWD)/bin go install -mod readonly $(TOOLS)

.PHONY: build/server
build/server:
	go build -o bin/server ./cmd/server/*.go

.PHONY: build/image
build/image: build/web build/server
	@echo "building image..."
	@echo "registry: $(REGISTRY)"
	docker build -t $(REGISTRY) .

.PHONY: push/image
push/image: build/image
	docker push $(REGISTRY)

.PHONY: build/web
build/web:
	cd ../animekai-web; yarn run build
	statik -src ../animekai-web/dist

.PHONY: deploy
deploy: push/image
	@gcloud beta run deploy \
		--allow-unauthenticated \
		--platform managed \
		--region $(CLOUD_RUN_REGION) $(CLOUD_RUN_SERVICE_NAME) \
		--image $(REGISTRY)

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test: build lint
	go test ./...
