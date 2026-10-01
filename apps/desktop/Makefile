SHELL := /bin/sh

ROOT_DIR      := $(shell git -C $(CURDIR) rev-parse --show-toplevel 2>/dev/null || (cd ../.. && pwd))
DESKTOP_DIR   := $(ROOT_DIR)/apps/desktop
APP_NAME      := Relay DB
WAILS_VERSION := v2.13.0
WAILS_CMD     := $(shell command -v wails 2>/dev/null || { [ -x "$$HOME/go/bin/wails" ] && echo "$$HOME/go/bin/wails"; } || printf '%s' 'go run github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)')
DEV_ENV       := RELAYDB_DISABLE_KEYCHAIN=1

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
ifeq ($(VERSION),)
VERSION := dev
endif
LDFLAGS := -X 'github.com/relay-client/relay-db/apps/desktop/internal/api.appVersion=$(VERSION)'

MAC_APP := $(DESKTOP_DIR)/build/bin/$(APP_NAME).app

.PHONY: help dev frontend install tidy check test build build-macos open clean

help:
	@printf '%s\n' 'Available targets:'
	@printf '  %-14s %s\n' 'make dev' 'Run Wails dev mode (hot reload, keychain disabled)'
	@printf '  %-14s %s\n' 'make frontend' 'Run only the Svelte/Vite frontend'
	@printf '  %-14s %s\n' 'make install' 'Install npm deps and tidy Go deps'
	@printf '  %-14s %s\n' 'make check' 'Typecheck the frontend and run all tests'
	@printf '  %-14s %s\n' 'make build' 'Build the desktop app for this platform'

dev:
	cd $(DESKTOP_DIR) && $(DEV_ENV) $(WAILS_CMD) dev

frontend:
	cd $(ROOT_DIR) && npm run frontend:dev

install:
	cd $(ROOT_DIR) && npm install
	cd $(DESKTOP_DIR) && go mod tidy

tidy:
	cd $(DESKTOP_DIR) && go mod tidy

check:
	cd $(ROOT_DIR) && npm run frontend:check
	cd $(ROOT_DIR) && npm run frontend:test
	cd $(DESKTOP_DIR) && go vet ./... && go test ./...

test:
	cd $(DESKTOP_DIR) && go test ./...

build:
	cd $(DESKTOP_DIR) && $(WAILS_CMD) build -ldflags "$(LDFLAGS)"

build-macos:
	cd $(DESKTOP_DIR) && $(WAILS_CMD) build -platform darwin/universal -ldflags "$(LDFLAGS)"

open:
	open "$(MAC_APP)"

clean:
	rm -rf "$(DESKTOP_DIR)/build/bin" "$(DESKTOP_DIR)/frontend/dist/assets" "$(DESKTOP_DIR)/frontend/dist/index.html"
