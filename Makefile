APP_NAME = wexinboot
APP_PATH = ./cmd/weixin-echo

# 编译路径
BUILD_PATH := ./dist

# 编译时间
BUILD_DATE = $(shell date +'%F %T')

ifeq ($(shell test -d .git && echo yes),yes)
# git已初始化,可以安全执行git命令
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git rev-list --count HEAD)
GIT_REVISION = $(shell git rev-parse --short HEAD)
GIT_COMMITAT = $(shell git --no-pager log -1 --format="%at")
else
# git未初始化,相关变量设为默认值
GIT_BRANCH = -
GIT_COMMIT = -
GIT_REVISION = -
GIT_COMMITAT = -
endif

# support包名称
FLAGS_PKG = github.com/wonli/aqi
BASE_LDFLAGS = -X '$(FLAGS_PKG).BuildDate=$(BUILD_DATE)' \
               -X '$(FLAGS_PKG).Branch=$(GIT_BRANCH)' \
               -X '$(FLAGS_PKG).CommitVersion=$(GIT_COMMIT)' \
               -X '$(FLAGS_PKG).Revision=$(GIT_REVISION)'

STRIP_LDFLAGS = -s -w
LDFLAGS = $(BASE_LDFLAGS) $(STRIP_LDFLAGS) -extldflags '-static'
LDFLAGS_DARWIN = $(BASE_LDFLAGS) $(STRIP_LDFLAGS)
LDFLAGS_WINDOWS = $(BASE_LDFLAGS) $(STRIP_LDFLAGS) -H=windowsgui -extldflags '-static'

# Go编译命令 1-GOOS 2-GOARCH 3-CGO_ENABLED 4-LDFLAGS 5-FILE EXT
define go/build
	go mod tidy
	GOOS=$(1) GOARCH=$(2) CGO_ENABLED=$(3) go build -ldflags "$(4)" -trimpath -tags netgo -o $(BUILD_PATH)/$(APP_NAME)-$(1)-$(2)-latest$(5) ${APP_PATH}
endef

# Generate binaries
.PHONY: build darwin linux windows icons

darwin:
	$(call go/build,darwin,arm64,1,$(LDFLAGS_DARWIN),)

linux:
	$(call go/build,linux,amd64,0,$(LDFLAGS),)

windows:
	$(call go/build,windows,amd64,0,$(LDFLAGS_WINDOWS),.exe)

include make/release-upload.mk
