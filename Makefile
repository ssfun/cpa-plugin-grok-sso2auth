PLUGIN_ID := grok-sso2auth
VERSION   ?= 0.1.0
DIST      := dist
CMD       := ./cmd/$(PLUGIN_ID)

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(OS),Windows_NT)
  PLUGIN_EXT := dll
else ifeq ($(UNAME_S),Darwin)
  PLUGIN_EXT := dylib
else
  PLUGIN_EXT := so
endif

.PHONY: all build test vet tidy clean package install help

all: build

help:
	@echo "Targets:"
	@echo "  make build     # build shared library into dist/"
	@echo "  make test      # go test ./..."
	@echo "  make package   # zip + sha256 for current platform"
	@echo "  make install   # copy into \$$HOME/.cli-proxy-api/plugins (or PLUGINS_DIR)"
	@echo "  make clean"

tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

build: tidy
	@mkdir -p $(DIST)
	CGO_ENABLED=1 go build -trimpath -buildmode=c-shared \
		-ldflags "-s -w -X main.pluginVersion=$(VERSION)" \
		-o $(DIST)/$(PLUGIN_ID).$(PLUGIN_EXT) $(CMD)
	@rm -f $(DIST)/$(PLUGIN_ID).h $(DIST)/$(PLUGIN_ID).$(PLUGIN_EXT).h
	@echo "built $(DIST)/$(PLUGIN_ID).$(PLUGIN_EXT)"

package: build
	@go run ./.github/scripts/package-release.go \
		-library "$(DIST)/$(PLUGIN_ID).$(PLUGIN_EXT)" \
		-archive "$(DIST)/$(PLUGIN_ID)_$(VERSION)_$$(go env GOOS)_$$(go env GOARCH).zip" \
		-checksum "$(DIST)/$(PLUGIN_ID)_$(VERSION)_$$(go env GOOS)_$$(go env GOARCH).zip.sha256"
	@echo "packaged under $(DIST)/"

PLUGINS_DIR ?= $(HOME)/.cli-proxy-api/plugins

install: build
	@mkdir -p "$(PLUGINS_DIR)"
	cp "$(DIST)/$(PLUGIN_ID).$(PLUGIN_EXT)" "$(PLUGINS_DIR)/"
	@echo "installed -> $(PLUGINS_DIR)/$(PLUGIN_ID).$(PLUGIN_EXT)"
	@echo "ensure config has:"
	@echo "  plugins:"
	@echo "    enabled: true"
	@echo "    dir: \"$(PLUGINS_DIR)\""
	@echo "    configs:"
	@echo "      $(PLUGIN_ID):"
	@echo "        enabled: true"

clean:
	rm -rf $(DIST) *.so *.dylib *.dll *.h
