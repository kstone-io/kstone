.PHONY: all
all: lint build

# ==============================================================================
# Build options

ROOT_PACKAGE=tkestack.io/kstone
VERSION_PACKAGE=tkestack.io/kstone/pkg/app/version

# ==============================================================================
# Includes

include build/lib/common.mk
include build/lib/golang.mk
include build/lib/image.mk
include build/lib/gen.mk

# ==============================================================================
# Usage

define USAGE_OPTIONS

Options:
  DEBUG        Whether to generate debug symbols. Default is 0.
  BINS         The binaries to build. Default is all of cmd.
               This option is available when using: make build/build.multiarch
               Example: make build BINS="kstone-api kstone-controller"
  IMAGES       Backend images to make. Default is all of cmd starting with kstone-.
               This option is available when using: make image/image.multiarch/push/push.multiarch
               Example: make image.multiarch IMAGES="kstone-api kstone-controller"
  PLATFORMS    The multiple platforms to build. Default is linux_amd64 and linux_arm64.
               This option is available when using: make build.multiarch/image.multiarch/push.multiarch
               Example: make image.multiarch IMAGES="kstone-api kstone-controller" PLATFORMS="linux_amd64 linux_arm64"
  VERSION      The version information compiled into binaries.
               The default is obtained from git.
  V            Set to 1 enable verbose build. Default is 0.
endef
export USAGE_OPTIONS

# ==============================================================================
# Targets

## gen: Generate codes for API definitions.
.PHONY: gen
gen:
	@$(MAKE) gen.run

## build: Build source code for host platform.
.PHONY: build
build:
	@$(MAKE) go.build

## build.multiarch: Build source code for multiple platforms. See option PLATFORMS.
.PHONY: build.multiarch
build.multiarch:
	@$(MAKE) go.build.multiarch

## image: Build docker images for host arch.
.PHONY: image
image:
	@$(MAKE) image.build

## image.multiarch: Build docker images for multiple platforms. See option PLATFORMS.
.PHONY: image.multiarch
image.multiarch:
	@$(MAKE) image.build.multiarch

## push: Build docker images for host arch and push images to registry.
.PHONY: push
push:
	@$(MAKE) image.push

## push.multiarch: Build docker images for multiple platforms and push images to registry.
.PHONY: push.multiarch
push.multiarch:
	@$(MAKE) image.push.multiarch

## manifest: Build docker images for host arch and push manifest list to registry.
.PHONY: manifest
manifest:
	@$(MAKE) image.manifest.push

## manifest.multiarch: Build docker images for multiple platforms and push manifest lists to registry.
.PHONY: manifest.multiarch
manifest.multiarch:
	@$(MAKE) image.manifest.push.multiarch

## clean: Remove all files that are created by building.
.PHONY: clean
clean:
	@$(MAKE) go.clean

## lint: Check syntax and styling of go sources.
.PHONY: lint
lint:
	@$(MAKE) go.lint

## test: Run unit test.
.PHONY: test
test:
	@$(MAKE) go.test

## e2e: Run e2e test.
.PHONY: e2e
e2e:
	@$(MAKE) go.e2e

## release-test: test release
.PHONY: release-test
release-test:
	@go test -v -timeout=1200m tkestack.io/kstone/test/e2e

## help: Show this help info.
.PHONY: help
help: Makefile
	@echo -e "\nUsage: make <TARGETS> <OPTIONS> ...\n\nTargets:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo "$$USAGE_OPTIONS"
