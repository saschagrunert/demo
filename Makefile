GO := go

BUILD_PATH := $(shell pwd)/build
GOLANGCI_LINT := ${BUILD_PATH}/golangci-lint
GINKGO := $(BUILD_PATH)/ginkgo

COVERAGE_PATH := $(BUILD_PATH)/coverage
JUNIT_PATH := $(BUILD_PATH)/junit

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
	rm -rf $(JUNIT_PATH) && mkdir -p $(JUNIT_PATH)
	$(GO) run github.com/onsi/ginkgo/ginkgo $(TESTFLAGS) \
		-r -p \
		--cover \
		--randomizeAllSpecs \
		--randomizeSuites \
		--covermode atomic \
		--outputdir $(COVERAGE_PATH) \
		--coverprofile coverprofile \
		--slowSpecThreshold 60 \
		--succinct
	$(GO) tool cover -html=$(COVERAGE_PATH)/coverprofile -o $(COVERAGE_PATH)/coverage.html
	find . -name '*_junit.xml' -exec mv -t $(JUNIT_PATH) {} +
	rm coverprofile

${GOLANGCI_LINT}:
	export \
		VERSION=v1.40.1 \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		BINDIR=${BUILD_PATH} && \
	curl -sfL $$URL/$$VERSION/install.sh | sh -s $$VERSION

.PHONY: lint
lint: ${GOLANGCI_LINT}
	${GOLANGCI_LINT} run
