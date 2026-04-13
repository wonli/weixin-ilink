.PHONY: opcpu-public release-upload-ref release-build

OPCPU_RELEASE_REF ?= master
OPCPU_RELEASE_PACKAGES ?= windows-amd64 darwin-arm64 darwin-amd64 linux-amd64


PACKAGE_VERSION ?= $(GIT_REVISION)

# 编译路径
BUILD_PATH := ./dist
PACKAGE_TMP_PATH := $(BUILD_PATH)/package-tmp
PACKAGE_OUTPUT_PATH := $(BUILD_PATH)/packages

PUBLIC_RELEASE_SRC_DIR ?= $(BUILD_PATH)/public-release-src
PUBLIC_RELEASE_ARCHIVE ?= $(BUILD_PATH)/public-release-src.tar
PUBLIC_RELEASE_PACKAGE_DIR ?= $(BUILD_PATH)/public-release-package

RELEASE_REF_VERSION = $(shell git rev-parse --short "$(OPCPU_RELEASE_REF)" 2>/dev/null)
RELEASE_REF_NOTES = $(shell git log -1 --pretty=%s "$(OPCPU_RELEASE_REF)" 2>/dev/null)
RELEASE_REF_COMMIT_COUNT = $(shell git rev-list --count "$(OPCPU_RELEASE_REF)" 2>/dev/null)
RELEASE_UPLOAD_TOKEN = $(or $(strip $(TOKEN)),$(strip $(OPCPU_AUTHOR_TOKEN)))
RELEASE_BUILD_OS = $(word 1,$(subst -, ,$(PKG)))
RELEASE_BUILD_ARCH = $(word 2,$(subst -, ,$(PKG)))
RELEASE_BUILD_EXT = $(if $(filter windows,$(RELEASE_BUILD_OS)),.exe,)

opcpu-public:
	@$(MAKE) release-upload-ref OPCPU_RELEASE_REF="$(OPCPU_RELEASE_REF)" TOKEN="$(RELEASE_UPLOAD_TOKEN)"

release-build:
	$(call go/build,$(RELEASE_BUILD_OS),$(RELEASE_BUILD_ARCH),$(if $(filter darwin,$(RELEASE_BUILD_OS)),1,0),$(if $(filter windows,$(RELEASE_BUILD_OS)),$(LDFLAGS_WINDOWS),$(if $(filter darwin,$(RELEASE_BUILD_OS)),$(LDFLAGS_DARWIN),$(LDFLAGS))),$(RELEASE_BUILD_EXT))

release-upload-ref:
	@if [ -z "$(OPCPU_RELEASE_REF)" ]; then echo "OPCPU_RELEASE_REF is required"; exit 1; fi
	@if [ -z "$(RELEASE_UPLOAD_TOKEN)" ]; then echo "TOKEN is required (set TOKEN or OPCPU_AUTHOR_TOKEN)"; exit 1; fi
	@if [ -z "$(RELEASE_REF_VERSION)" ]; then echo "failed to resolve ref: $(OPCPU_RELEASE_REF)"; exit 1; fi
	@set -e; \
	rm -rf "$(PUBLIC_RELEASE_SRC_DIR)" "$(PUBLIC_RELEASE_PACKAGE_DIR)"; \
	rm -f "$(PUBLIC_RELEASE_ARCHIVE)"; \
		mkdir -p "$(PUBLIC_RELEASE_SRC_DIR)" "$(PUBLIC_RELEASE_PACKAGE_DIR)"; \
		git archive --format=tar "$(OPCPU_RELEASE_REF)" -o "$(PUBLIC_RELEASE_ARCHIVE)"; \
		tar -xf "$(PUBLIC_RELEASE_ARCHIVE)" -C "$(PUBLIC_RELEASE_SRC_DIR)"; \
		cp "$(CURDIR)/Makefile" "$(PUBLIC_RELEASE_SRC_DIR)/Makefile"; \
		mkdir -p "$(PUBLIC_RELEASE_SRC_DIR)/make"; \
		cp "$(CURDIR)/make/release-upload.mk" "$(PUBLIC_RELEASE_SRC_DIR)/make/release-upload.mk"; \
	mkdir -p "$(PUBLIC_RELEASE_SRC_DIR)/dist/package-tmp" "$(PUBLIC_RELEASE_SRC_DIR)/dist/packages"; \
	for pkg in $(OPCPU_RELEASE_PACKAGES); do \
	  case "$$pkg" in \
	    windows-amd64) \
	      bin_name="$(APP_NAME)-windows-amd64-latest.exe"; \
	      zip_name="$(APP_NAME)-windows-amd64-$(RELEASE_REF_VERSION).exe"; \
	      asset_name="$(APP_NAME)-windows-amd64-$(RELEASE_REF_VERSION).zip"; \
	      os="windows"; arch="amd64"; label="Windows x64"; sort="10"; \
	      ;; \
	    darwin-arm64) \
	      bin_name="$(APP_NAME)-darwin-arm64-latest"; \
	      zip_name="$(APP_NAME)-darwin-arm64-$(RELEASE_REF_VERSION)"; \
	      asset_name="$(APP_NAME)-darwin-arm64-$(RELEASE_REF_VERSION).zip"; \
	      os="darwin"; arch="arm64"; label="macOS Apple Silicon"; sort="20"; \
	      ;; \
	    darwin-amd64) \
	      bin_name="$(APP_NAME)-darwin-amd64-latest"; \
	      zip_name="$(APP_NAME)-darwin-amd64-$(RELEASE_REF_VERSION)"; \
	      asset_name="$(APP_NAME)-darwin-amd64-$(RELEASE_REF_VERSION).zip"; \
	      os="darwin"; arch="amd64"; label="macOS Intel"; sort="30"; \
	      ;; \
	    linux-amd64) \
	      bin_name="$(APP_NAME)-linux-amd64-latest"; \
	      zip_name="$(APP_NAME)-linux-amd64-$(RELEASE_REF_VERSION)"; \
	      asset_name="$(APP_NAME)-linux-amd64-$(RELEASE_REF_VERSION).zip"; \
	      os="linux"; arch="amd64"; label="Linux x64"; sort="40"; \
	      ;; \
	    *) \
	      echo "unsupported package: $$pkg"; \
	      exit 1; \
	      ;; \
	  esac; \
	  $(MAKE) -C "$(PUBLIC_RELEASE_SRC_DIR)" \
	    -f Makefile -f make/release-upload.mk \
	    GOWORK=off \
	    APP_NAME="$(APP_NAME)" \
	    APP_PATH="$(APP_PATH)" \
	    BUILD_PATH=./dist \
	    PACKAGE_VERSION="$(RELEASE_REF_VERSION)" \
	    GIT_BRANCH="$(OPCPU_RELEASE_REF)" \
	    GIT_COMMIT="$(RELEASE_REF_COMMIT_COUNT)" \
	    GIT_REVISION="$(RELEASE_REF_VERSION)" \
	    BUILD_DATE="$(BUILD_DATE)" \
	    PKG="$$pkg" \
	    release-build; \
	  cp "$(PUBLIC_RELEASE_SRC_DIR)/dist/$$bin_name" "$(PUBLIC_RELEASE_SRC_DIR)/dist/package-tmp/$$zip_name"; \
	  (cd "$(PUBLIC_RELEASE_SRC_DIR)/dist/package-tmp" && zip -q -j "$(abspath $(PUBLIC_RELEASE_PACKAGE_DIR))/$$asset_name" "$$zip_name"); \
	  echo "Uploading $$label -> $(PUBLIC_RELEASE_PACKAGE_DIR)/$$asset_name"; \
	  curl --max-time 120 -X POST "https://www.opcpu.com/api/author/projects/weixin-ilink/release/assets/upload" \
	    -H "Authorization: Bearer $(RELEASE_UPLOAD_TOKEN)" \
	    -F "file=@$(abspath $(PUBLIC_RELEASE_PACKAGE_DIR))/$$asset_name" \
	    -F "version=$(RELEASE_REF_VERSION)" \
	    -F "release_notes=$(RELEASE_REF_NOTES)" \
	    -F "branch=$(OPCPU_RELEASE_REF)" \
	    -F "commit_hash=$(RELEASE_REF_VERSION)" \
	    -F "os=$$os" \
	    -F "arch=$$arch" \
	    -F "label=$$label" \
	    -F "sort=$$sort"; \
	done
