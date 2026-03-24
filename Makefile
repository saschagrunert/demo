GO := go

BUILD_PATH := $(shell pwd)/build
COVERAGE_PATH := $(BUILD_PATH)/coverage

GOLANGCI_LINT_VERSION := v2.11.4
GOLANGCI_LINT := ${BUILD_PATH}/golangci-lint

COLOR := \\033[36m
NOCOLOR := \\033[0m
WIDTH := 25

# Build flags: strip symbols only in release mode
# For development builds, omit -s -w to preserve debug symbols
LDFLAGS ?= -s -w

define go-build
	cd `pwd` && $(GO) build -ldflags '$(LDFLAGS) $(2)' \
		-o $(BUILD_PATH)/$(shell basename $(1)) $(1)
	@echo > /dev/null
endef

##@ General:

.PHONY: help
help: ## Display this help.
	@awk \
		-v "col=${COLOR}" -v "nocol=${NOCOLOR}" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "Usage:\n  make %s<target>%s\n", col, nocol \
			} \
			/^[./a-zA-Z_-]+:.*?##/ { \
				printf "  %s%-${WIDTH}s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s\n", substr($$0, 5) \
			} \
		' $(MAKEFILE_LIST)

##@ Build targets:

all: ## Build the demo binary.
	$(call go-build,./cmd)

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BUILD_PATH)

##@ Testing:

.PHONY: codecov
codecov: SHELL := $(shell which bash)
codecov: ## Upload coverage to codecov.
	bash <(curl -s https://codecov.io/bash) -f $(COVERAGE_PATH)/coverprofile

.PHONY: test
test: ## Run tests with coverage.
	rm -rf $(COVERAGE_PATH) && mkdir -p $(COVERAGE_PATH)
	$(GO) run github.com/onsi/ginkgo/v2/ginkgo run $(TESTFLAGS) \
		-r \
		--cover \
		--randomize-all \
		--randomize-suites \
		--covermode atomic \
		--output-dir $(COVERAGE_PATH) \
		--coverprofile coverprofile \
		--junit-report coverage.junit \
		--succinct
	$(GO) tool cover -html=$(COVERAGE_PATH)/coverprofile -o $(COVERAGE_PATH)/coverage.html

##@ Linting:

${GOLANGCI_LINT}:
	export \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		VERSION=${GOLANGCI_LINT_VERSION} \
		BINDIR=${BUILD_PATH} && \
	curl -sfL $$URL/refs/heads/main/install.sh | sh -s $$VERSION

.PHONY: lint
lint: ${GOLANGCI_LINT} ## Run golangci-lint.
	GL_DEBUG=gocritic ${GOLANGCI_LINT} linters
	${GOLANGCI_LINT} run
