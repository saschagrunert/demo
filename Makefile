GO := go

BUILD_PATH := $(shell pwd)/build
COVERAGE_PATH := $(BUILD_PATH)/coverage

GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT_PATH := $(BUILD_PATH)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT := $(GOLANGCI_LINT_PATH)/golangci-lint

COLOR := \\033[36m
NOCOLOR := \\033[0m
WIDTH := 25

# Build flags: symbols are stripped by default, use `make LDFLAGS=` to keep them
LDFLAGS ?= -s -w

define go-build
	$(GO) build -ldflags '$(LDFLAGS) $(2)' \
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

.PHONY: all
all: ## Build the demo binary.
	$(call go-build,./cmd)

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BUILD_PATH)

##@ Testing:

.PHONY: test
test: ## Run tests with coverage.
	rm -rf $(COVERAGE_PATH) && mkdir -p $(COVERAGE_PATH)
	$(GO) tool ginkgo run $(TESTFLAGS) \
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

##@ Documentation:

.PHONY: demo-svg
demo-svg: ## Record the README demo into .github/demo.svg (requires python3 and npx).
	$(GO) build -o $(BUILD_PATH)/demo ./cmd
	python3 .github/record-demo.py $(BUILD_PATH)/demo.cast \
		$(BUILD_PATH)/demo --demo-0 --auto --theme=dark
	npx -y svg-term-cli@2.1.1 --in $(BUILD_PATH)/demo.cast \
		--out .github/demo.svg --width 90 --height 20

##@ Linting:

${GOLANGCI_LINT}:
	export \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		VERSION=${GOLANGCI_LINT_VERSION} \
		BINDIR=$(GOLANGCI_LINT_PATH) && \
	mkdir -p $$BINDIR && \
	curl -sfL $$URL/refs/tags/$$VERSION/install.sh -o $$BINDIR/install.sh && \
	sh $$BINDIR/install.sh $$VERSION && \
	rm $$BINDIR/install.sh

.PHONY: lint
lint: ${GOLANGCI_LINT} ## Run golangci-lint.
	GL_DEBUG=gocritic ${GOLANGCI_LINT} linters
	${GOLANGCI_LINT} run
