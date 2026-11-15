VER ?= $(VERSION)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
APP_NAME := DV-Warp-Scanner
GO := GO111MODULE=on CGO_ENABLED=0 go
LDFLAGS = -w -s \
    -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
    -X "main.VERSION=$(VER)" \
    -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

OUT_DIR := bin
DIST_DIR := dist
CORE_DIR := core
TEMP_DIR := temp
XRAY_VERSION ?= $(shell cat XRAY_VERSION)
XRAY_PLATFORM := $(if $(filter darwin,$(GOOS)),macos,$(GOOS))
XRAY_ARCH := $(shell case "$(GOOS)/$(GOARCH)" in \
    linux/amd64|windows/amd64|darwin/amd64) echo 64 ;; \
    linux/386|windows/386) echo 32 ;; \
    linux/arm64|windows/arm64|darwin/arm64) echo arm64-v8a ;; \
    linux/arm) echo arm32-v5 ;; \
    *) echo unsupported ;; \
esac)
XRAY_ASSET := Xray-$(XRAY_PLATFORM)-$(XRAY_ARCH).zip
XRAY_URL := https://github.com/XTLS/Xray-core/releases/download/$(XRAY_VERSION)/$(XRAY_ASSET)
XRAY_ZIP := $(TEMP_DIR)/$(XRAY_ASSET)
XRAY_DIGEST := $(TEMP_DIR)/$(XRAY_ASSET).dgst

.PHONY: build clean pre-build

pre-build:
	@mkdir -p $(OUT_DIR) $(DIST_DIR) $(TEMP_DIR)

build: pre-build
	@if [ "$(XRAY_ARCH)" = "unsupported" ]; then \
		echo "Unsupported Xray target: $(GOOS)-$(GOARCH)"; exit 1; \
	fi
	@echo "Downloading official Xray-core $(XRAY_VERSION) ($(XRAY_ASSET))..."
	@rm -rf "$(CORE_DIR)" && mkdir -p "$(CORE_DIR)"
	@curl -fL --retry 3 --retry-delay 2 -o "$(XRAY_ZIP)" "$(XRAY_URL)"
	@curl -fL --retry 3 --retry-delay 2 -o "$(XRAY_DIGEST)" "$(XRAY_URL).dgst"
	@expected="$$(grep -Eo '[[:xdigit:]]{64}' "$(XRAY_DIGEST)" | head -n 1 | tr '[:upper:]' '[:lower:]')"; \
	if command -v sha256sum >/dev/null 2>&1; then \
		actual="$(sha256sum "$(XRAY_ZIP)" | awk '{print $1}')"; \
	elif command -v shasum >/dev/null 2>&1; then \
		actual="$(shasum -a 256 "$(XRAY_ZIP)" | awk '{print $1}')"; \
	else \
		echo "No SHA-256 utility found."; exit 1; \
	fi; \
	if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then \
		echo "Xray archive SHA-256 verification failed."; exit 1; \
	fi
	@unzip -oq "$(XRAY_ZIP)" -d "$(CORE_DIR)"
	@printf '%s\n' "$(XRAY_VERSION)" > "$(CORE_DIR)/version.txt"
	@printf '%s\n' "$(XRAY_VERSION)" > "$(CORE_DIR)/bundled-version.txt"
	@rm -f "$(XRAY_ZIP)" "$(XRAY_DIGEST)"
	@set -e; \
	if [ "$(GOOS)" = "windows" ]; then \
		test -f "$(CORE_DIR)/xray.exe" || { echo "Xray executable not found in archive."; exit 1; }; \
		ext=".exe"; \
	else \
		test -f "$(CORE_DIR)/xray" || { echo "Xray executable not found in archive."; exit 1; }; \
		ext=""; \
	fi; \
	echo "Building $(APP_NAME) for $(GOOS)-$(GOARCH)..."; \
	outdir="$(OUT_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH)"; \
	mkdir -p "$$outdir"; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -buildvcs=false -ldflags '$(LDFLAGS)' -o "$$outdir/$(APP_NAME)$$ext"; \
	cp LICENSE "$$outdir/"; \
	cp -r "$(CORE_DIR)" "$$outdir/"; \
	archive="$(APP_NAME)-$(GOOS)-$(GOARCH)"; \
	if [ "$(GOOS)" = "windows" ]; then \
		(cd "$$outdir" && zip -9vr -q "../../$(DIST_DIR)/$$archive.zip" .); \
	else \
		tar -czf "$(DIST_DIR)/$$archive.tar.gz" -C "$$outdir/" .; \
	fi
	@rm -rf "$(CORE_DIR)"

clean:
	@rm -rf $(OUT_DIR) $(DIST_DIR) $(TEMP_DIR) $(CORE_DIR)
