# ============================================================================
# axt-plugin Makefile
# Builds Cloud Assistant plugin zips for linux-x64, linux-arm64, windows-x64.
# Each platform is staged in its own isolated workspace under build/<platform>/
# so platform builds cannot pollute each other or the source tree.
# ============================================================================

# ----------------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------------
NODE_VERSION := v20.19.3
NODE_BASE_URL := https://nodejs.org/dist/$(NODE_VERSION)

PLATFORMS := linux-x64 linux-arm64 windows-x64

# Per-platform metadata. Resolved in pattern rules with $(<VAR>_$*).
NODE_DIST_linux-x64    := node-$(NODE_VERSION)-linux-x64.tar.xz
NODE_DIST_linux-arm64  := node-$(NODE_VERSION)-linux-arm64.tar.xz
NODE_DIST_windows-x64  := node-$(NODE_VERSION)-win-x64.zip

# Path inside each Node distribution to the actual interpreter binary.
NODE_BIN_PATH_linux-x64    := node-$(NODE_VERSION)-linux-x64/bin/node
NODE_BIN_PATH_linux-arm64  := node-$(NODE_VERSION)-linux-arm64/bin/node
NODE_BIN_PATH_windows-x64  := node-$(NODE_VERSION)-win-x64/node.exe

# Cloud Assistant config.json fields.
OS_TYPE_linux-x64      := linux
OS_TYPE_linux-arm64    := linux
OS_TYPE_windows-x64    := windows
ARCH_linux-x64         := x64
ARCH_linux-arm64       := arm
ARCH_windows-x64       := x64

# Entrypoint file (lives in source tree) and zipped binary name.
ENTRYPOINT_linux-x64   := entrypoint.sh
ENTRYPOINT_linux-arm64 := entrypoint.sh
ENTRYPOINT_windows-x64 := entrypoint.bat
NODE_BIN_linux-x64     := node
NODE_BIN_linux-arm64   := node
NODE_BIN_windows-x64   := node.exe

# ----------------------------------------------------------------------------
# Directories
# ----------------------------------------------------------------------------
PLUGIN_DIR := $(abspath .)
ROOT_DIR   := $(abspath ..)
BUILD_DIR  := $(PLUGIN_DIR)/build
SHARED_DIR := $(BUILD_DIR)/_shared

CLI_PKG           := $(ROOT_DIR)/packages/cli/package.json
CLI_ESBUILD_CONFIG := $(ROOT_DIR)/packages/cli/esbuild.config.mjs
CLI_DIST          := $(ROOT_DIR)/packages/cli/dist

# Pull the agent version from the CLI package.json (single source of truth).
CLI_VERSION := $(shell node -p "require('$(CLI_PKG)').version")

# Output zips at repo root, one per platform.
ZIP_FILES := $(foreach p,$(PLATFORMS),$(ROOT_DIR)/axt-plugin-$(p).zip)

# ----------------------------------------------------------------------------
# Phony targets
# ----------------------------------------------------------------------------
.PHONY: all build clean summary
.SECONDARY: $(foreach p,$(PLATFORMS),$(BUILD_DIR)/$(p)/.staged)

# ----------------------------------------------------------------------------
# Default target: build all platform zips
# ----------------------------------------------------------------------------
all: $(ZIP_FILES)

# ----------------------------------------------------------------------------
# Build the CLI (turbo handles caching internally)
# ----------------------------------------------------------------------------
build:
	@echo ">>> Building CLI..."
	cd $(ROOT_DIR) && npm run build
	@echo "CLI build complete."

# ----------------------------------------------------------------------------
# Shared workspace (built once, reused by every platform staging)
# ----------------------------------------------------------------------------
$(SHARED_DIR)/package.json: package.json.in tools/render-package-json.mjs $(CLI_PKG) $(CLI_ESBUILD_CONFIG)
	@echo ">>> Generating $(SHARED_DIR)/package.json (agent $(CLI_VERSION))..."
	@mkdir -p $(SHARED_DIR)
	node tools/render-package-json.mjs $(SHARED_DIR)/package.json

$(SHARED_DIR)/node_modules: $(SHARED_DIR)/package.json
	@echo ">>> Installing production dependencies into $(SHARED_DIR)..."
	rm -rf $(SHARED_DIR)/node_modules $(SHARED_DIR)/package-lock.json
	cd $(SHARED_DIR) && npm install --omit=dev
	rm -f $(SHARED_DIR)/package-lock.json
	@echo "Production dependencies installed."

$(SHARED_DIR)/dist: | build
	@echo ">>> Copying dist/ into $(SHARED_DIR)..."
	rm -rf $(SHARED_DIR)/dist
	mkdir -p $(SHARED_DIR)/dist
	cp $(CLI_DIST)/cli.js $(SHARED_DIR)/dist/
	cp $(CLI_DIST)/cli.js.map $(SHARED_DIR)/dist/
	cp -R $(CLI_DIST)/skills $(SHARED_DIR)/dist/
	@echo "Copied dist/ successfully."

# ----------------------------------------------------------------------------
# Per-platform staging workspace
#
# A sentinel file `build/<platform>/.staged` represents a fully populated
# workspace ready to zip. The recipe rebuilds the workspace from scratch on
# every invocation so a re-run cannot leave stale platform-foreign files
# behind.
# ----------------------------------------------------------------------------
$(BUILD_DIR)/%/.staged: $(SHARED_DIR)/node_modules $(SHARED_DIR)/dist config.json.in $(ENTRYPOINT_linux-x64) $(ENTRYPOINT_windows-x64)
	@echo ">>> Staging $* workspace..."
	rm -rf $(BUILD_DIR)/$*
	mkdir -p $(BUILD_DIR)/$*/bin
	@echo ">>> Downloading $(NODE_DIST_$*)..."
	@if [ "$(suffix $(NODE_DIST_$*))" = ".zip" ]; then \
		tmpzip=$$(mktemp -t node-XXXXXX.zip); \
		curl -fsSL "$(NODE_BASE_URL)/$(NODE_DIST_$*)" -o $$tmpzip; \
		unzip -j -o $$tmpzip "$(NODE_BIN_PATH_$*)" -d $(BUILD_DIR)/$*/bin; \
		rm -f $$tmpzip; \
	else \
		curl -fsSL "$(NODE_BASE_URL)/$(NODE_DIST_$*)" \
			| tar xJ -C $(BUILD_DIR)/$*/bin --strip-components=2 "$(NODE_BIN_PATH_$*)"; \
	fi
	chmod +x $(BUILD_DIR)/$*/bin/$(NODE_BIN_$*)
	@echo ">>> Rendering $(BUILD_DIR)/$*/config.json (osType=$(OS_TYPE_$*), arch=$(ARCH_$*))..."
	sed -e 's/{{ VERSION }}/$(CLI_VERSION)/g' \
	    -e 's/{{ OS_TYPE }}/$(OS_TYPE_$*)/g' \
	    -e 's/{{ ARCH }}/$(ARCH_$*)/g' \
	    -e 's/{{ RUN_PATH }}/$(ENTRYPOINT_$*)/g' \
	    config.json.in > $(BUILD_DIR)/$*/config.json
	@echo ">>> Materialising shared dist/ and node_modules/..."
	cp -R $(SHARED_DIR)/dist $(BUILD_DIR)/$*/dist
	cp -R $(SHARED_DIR)/node_modules $(BUILD_DIR)/$*/node_modules
	cp $(ENTRYPOINT_$*) $(BUILD_DIR)/$*/$(ENTRYPOINT_$*)
	touch $@
	@echo "Staged $*."

# ----------------------------------------------------------------------------
# Per-platform zip
# ----------------------------------------------------------------------------
$(ROOT_DIR)/axt-plugin-%.zip: $(BUILD_DIR)/%/.staged
	@echo ">>> Creating $@..."
	rm -f $@
	cd $(BUILD_DIR)/$* && zip -r $@ . -x ".staged"
	@echo "Created $@."

# ----------------------------------------------------------------------------
# Summary
# ----------------------------------------------------------------------------
summary: $(ZIP_FILES)
	@echo "=== Package Summary ==="
	@for p in $(PLATFORMS); do \
		zip="$(ROOT_DIR)/axt-plugin-$$p.zip"; \
		files=$$(unzip -l $$zip | tail -1 | awk '{print $$2}'); \
		size=$$(du -h $$zip | cut -f1); \
		echo "  $$p: $$files files, $$size"; \
	done
	@echo ""
	@echo "=== Packaging complete! ==="

# ----------------------------------------------------------------------------
# Clean
# ----------------------------------------------------------------------------
clean:
	@echo ">>> Cleaning generated files..."
	rm -rf $(BUILD_DIR)
	rm -f $(ROOT_DIR)/axt-plugin-*.zip
	@echo "Clean complete."
