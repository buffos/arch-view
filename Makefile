ROOT_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
GO ?= go
CGO_CC ?= $(shell "$(GO)" env CC)
CGO_CXX ?= $(shell "$(GO)" env CXX)
HOST_GOOS := $(shell "$(GO)" env GOOS)
HOST_GOARCH := $(shell "$(GO)" env GOARCH)
PLATFORM ?= $(HOST_GOOS)-$(HOST_GOARCH)
VERSION ?=
COMMIT ?=
BUILD_DATE ?=
BUILD_ID ?= local
ANALYZERS ?= org.archview.clojure,org.archview.go,org.archview.python,org.archview.rust,org.archview.typescript
ANALYZER_ROOT ?= $(ROOT_DIR)/dist/analyzers
RELEASE_ROOT ?= $(ROOT_DIR)/dist/release

.PHONY: analyzers release

analyzers:
	"$(GO)" -C "$(ROOT_DIR)" run ./cmd/analyzer-distribution -repository-root "$(ROOT_DIR)" -output-root "$(ANALYZER_ROOT)" -platform "$(PLATFORM)" -build-id "$(BUILD_ID)" -go-command "$(GO)" -cc-command "$(CGO_CC)" -cxx-command "$(CGO_CXX)" -analyzers "$(ANALYZERS)"

release:
	"$(GO)" -C "$(ROOT_DIR)" run ./cmd/release-assembly -repository-root "$(ROOT_DIR)" -release-root "$(RELEASE_ROOT)" -platform "$(PLATFORM)" -version "$(VERSION)" -commit "$(COMMIT)" -build-date "$(BUILD_DATE)" -build-id "$(BUILD_ID)" -go-command "$(GO)" -cc-command "$(CGO_CC)" -cxx-command "$(CGO_CXX)" -analyzers "$(ANALYZERS)"
