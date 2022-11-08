GO := go

BUILD_PATH := $(shell pwd)/build
COVERAGE_PATH := $(BUILD_PATH)/coverage

GOLANGCI_LINT := ${BUILD_PATH}/golangci-lint

define go-build
	cd `pwd` && $(GO) build -ldflags '-s -w $(2)' \
		-o $(BUILD_PATH)/$(shell basename $(1)) $(1)
	@echo > /dev/null
endef

all:
	$(call go-build,./cmd)

.PHONY: clean
clean:
	rm -rf $(BUILD_PATH)

.PHONY: codecov
codecov: SHELL := $(shell which bash)
codecov:
	bash <(curl -s https://codecov.io/bash) -f $(COVERAGE_PATH)/coverprofile

.PHONY: test
test:
	rm -rf $(COVERAGE_PATH) && mkdir -p $(COVERAGE_PATH)
	$(GO) run github.com/onsi/ginkgo/v2/ginkgo run $(TESTFLAGS) \
		-r -p \
		--cover \
		--randomize-all \
		--randomize-suites \
		--covermode atomic \
		--output-dir $(COVERAGE_PATH) \
		--coverprofile coverprofile \
		--junit-report coverage.junit \
		--succinct
	$(GO) tool cover -html=$(COVERAGE_PATH)/coverprofile -o $(COVERAGE_PATH)/coverage.html

${GOLANGCI_LINT}:
	export \
		VERSION=v1.50.1 \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		BINDIR=${BUILD_PATH} && \
	curl -sfL $$URL/$$VERSION/install.sh | sh -s $$VERSION

.PHONY: lint
lint: ${GOLANGCI_LINT}
	GL_DEBUG=gocritic ${GOLANGCI_LINT} linters
	${GOLANGCI_LINT} run
