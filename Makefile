.PHONY: dev run build cli test gui-build linux-binaries package-linux package-linux-no-cache

ARGS ?=
ROOT_DIR := $(CURDIR)
GUI_DIR := $(ROOT_DIR)/packages/gui
GO_CACHE ?= /tmp/smokelab-go-build
BUILD_BIN_DIR := $(ROOT_DIR)/build/bin
PACKAGE_OUTPUT_DIR := $(ROOT_DIR)/build/packages
TOOLS_DIR := $(ROOT_DIR)/build/tools
NFPM_VERSION ?= v2.45.0
NFPM_BIN := $(TOOLS_DIR)/nfpm
PACKAGE_VERSION ?= 0.1.0
PACKAGE_ARCH ?= $(shell go env GOARCH)
WAILS_BUILD_FLAGS ?=

install:


dev:
	cd $(GUI_DIR) && GOCACHE=$(GO_CACHE) wails dev $(ARGS)

run: dev

build:
	cd $(GUI_DIR) && GOCACHE=$(GO_CACHE) wails build $(ARGS)

cli:
	GOCACHE=$(GO_CACHE) go run ./packages/cli $(ARGS)

test:
	GOCACHE=$(GO_CACHE) go test ./... $(ARGS)

gui-build:
	cd packages/gui && npm run build

$(NFPM_BIN):
	mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) GOCACHE=$(GO_CACHE) go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(NFPM_VERSION)

linux-binaries:
	mkdir -p $(BUILD_BIN_DIR)
	cd $(GUI_DIR) && GOCACHE=$(GO_CACHE) wails build $(WAILS_BUILD_FLAGS) -m -nopackage -trimpath -platform linux/$(PACKAGE_ARCH) -o SmokeLab
	GOCACHE=$(GO_CACHE) GOOS=linux GOARCH=$(PACKAGE_ARCH) go build -trimpath -ldflags="-s -w" -o $(BUILD_BIN_DIR)/smokelab ./packages/cli

package-linux: linux-binaries $(NFPM_BIN)
	mkdir -p $(PACKAGE_OUTPUT_DIR)
	PACKAGE_VERSION=$(PACKAGE_VERSION) PACKAGE_ARCH=$(PACKAGE_ARCH) $(NFPM_BIN) package \
		--config build/linux/nfpm.yaml \
		--packager deb \
		--target $(PACKAGE_OUTPUT_DIR)/smokelab_$(PACKAGE_VERSION)_$(PACKAGE_ARCH).deb

package-linux-no-cache:
	@package_cache="$$(mktemp -d /tmp/smokelab-package-cache.XXXXXX)"; \
	trap 'rm -rf "$$package_cache"' EXIT INT TERM; \
	$(MAKE) package-linux GO_CACHE="$$package_cache" WAILS_BUILD_FLAGS="-clean -f"
