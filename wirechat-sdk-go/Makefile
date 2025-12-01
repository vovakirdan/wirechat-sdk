.PHONY: all test build lint fmt tag-version tag-list tag-create tag-push tag-delete
GO ?= go

GOBIN := $(shell $(GO) env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell $(GO) env GOPATH)/bin
endif

GOLANGCI_LINT := $(GOBIN)/golangci-lint
GOLANGCI_LINT_VERSION := v1.62.2

# Version can be overridden: make tag-create VERSION=v0.2.0
VERSION ?= v0.1.1
TAG_NAME := wirechat-sdk-go/$(VERSION)

all: build

build:
	@echo ">> Building wirechat-sdk-go"

test:
	@echo ">> Running tests"
	$(GO) test ./...

fmt:
	@echo ">> Formatting code"
	$(GO) fmt ./...

$(GOLANGCI_LINT):
	@echo ">> Installing golangci-lint $(GOLANGCI_LINT_VERSION)"
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT)
	@echo ">> Running linters"
	$(GOLANGCI_LINT) run --config .golangci.yaml

# Tag management
tag-version:
	@echo ">> Current version:"
	@git describe --tags --abbrev=0 --match "wirechat-sdk-go/v*" 2>/dev/null || echo "No tags found"

tag-list:
	@echo ">> All Go SDK tags:"
	@git tag -l "wirechat-sdk-go/*" || echo "No tags found"

tag-create:
	@echo ">> Creating tag: $(TAG_NAME)"
	@git tag -a $(TAG_NAME) -m "Go SDK $(VERSION)\n\nWireChat Go SDK release $(VERSION)"
	@echo ">> Tag created successfully!"
	@echo ">> Push with: make tag-push VERSION=$(VERSION)"

tag-push:
	@echo ">> Pushing tag: $(TAG_NAME)"
	@git push origin $(TAG_NAME)
	@echo ">> Tag pushed successfully!"
	@echo ">> Users can now install with:"
	@echo "   go get github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go@$(TAG_NAME)"

tag-delete:
	@echo ">> Deleting tag: $(TAG_NAME)"
	@git tag -d $(TAG_NAME)
	@git push origin :refs/tags/$(TAG_NAME)
	@echo ">> Tag deleted!"

# Helper to create and push tag in one command
tag: tag-create tag-push