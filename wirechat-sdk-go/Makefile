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
get-tag:
	@echo "Актуальный тег Go-SDK"
	@git describe --tags --abbrev=0 --match "wirechat-sdk-go/*" 2>/dev/null || echo "No tags found"

get-tag-version:
	@echo "Версия Go-SDK"
	@git describe --tags --abbrev=0 --match "wirechat-sdk-go/*" 2>/dev/null | sed 's|wirechat-sdk-go/||' || echo "No tags found"

tag-patch:
	@echo "Создание нового тега с инкрементом patch-версии"
	@CURRENT_TAG=$$(git describe --tags --abbrev=0 --match "wirechat-sdk-go/*" 2>/dev/null || echo "wirechat-sdk-go/v0.0.0");\
	CURRENT_VERSION=$$(echo $$CURRENT_TAG | sed 's|wirechat-sdk-go/v||');\
	MAJOR=$$(echo $$CURRENT_VERSION | cut -d. -f1);\
	MINOR=$$(echo $$CURRENT_VERSION | cut -d. -f2);\
	PATCH=$$(echo $$CURRENT_VERSION | cut -d. -f3);\
	NEW_PATCH=$$((PATCH + 1));\
	NEW_VERSION="$$MAJOR.$$MINOR.$$NEW_PATCH";\
	NEW_TAG="wirechat-sdk-go/v$$NEW_VERSION";\
	echo "Текущий тег: $$CURRENT_TAG";\
	echo "Новый тег: $$NEW_TAG";\
	read -p "Создать тег? (y/n): " confirm && [ "$$confirm" == "y" ] || exit 1; \
	git tag -a $$NEW_TAG -m "Go SDK $$NEW_VERSION\n\nWireChat Go SDK release $$NEW_VERSION";\
	echo "Тег $$NEW_TAG создан";\
	echo "Для отправки тега выполните: git push origin $$NEW_TAG"

tag-create:
	@if [ -z "$(VERSION)" ]; then \
		echo "Версия не указана"; \
		echo "Укажите версию через VERSION=<MAJOR>.<MINOR>.<PATCH>";\
		exit 1;\
	fi;\
	NEW_TAG="wirechat-sdk-go/v$(VERSION)";\
	echo "Создание тега: $$NEW_TAG";\
	read -p "Создать тег? (y/n): " confirm && [ "$$confirm" == "y" ] || exit 1; \
	git tag -a $$NEW_TAG -m "Go SDK $(VERSION)\n\nWireChat Go SDK release $(VERSION)";\
	echo "Тег $$NEW_TAG создан";\
	echo "Для отправки тега выполните: git push origin $$NEW_TAG"

# Helper to create and push tag in one command
tag: tag-create tag-push