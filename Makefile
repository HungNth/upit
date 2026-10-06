BINARY_NAME := upit
CMD_DIR := ./cmd/upit
DESKTOP_CMD_DIR := ./cmd/upit-desktop
DESKTOP_FRONTEND_DIR := $(DESKTOP_CMD_DIR)/frontend
BUILD_DIR := bin
VERSION ?= $(strip $(file < VERSION))
ifeq ($(VERSION),)
    VERSION := 0.9.0
endif

ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    BINARY_EXT := .exe
    SHELL := cmd.exe
    .SHELLFLAGS := /C
    MKDIR_CMD := if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
    CLEAN_CMD := powershell -NoProfile -Command "$$ErrorActionPreference = 'Stop'; foreach ($$path in @('$(BUILD_DIR)', '.build', 'dist')) { if (Test-Path -LiteralPath $$path) { Remove-Item -LiteralPath $$path -Recurse -Force } }"
else
    DETECTED_OS := $(shell uname -s)
    BINARY_EXT :=
    MKDIR_CMD := mkdir -p "$(BUILD_DIR)"
    CLEAN_CMD := rm -rf "$(BUILD_DIR)" ".build" "dist"
endif

TARGET := $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT)
ifeq ($(OS),Windows_NT)
    RUN_CMD := "$(TARGET)"
else
    RUN_CMD := ./$(TARGET)
endif

export WINDOWS_CERTIFICATE_PATH WINDOWS_CERTIFICATE_PASSWORD WINDOWS_TIMESTAMP_SERVER WINDOWS_PROTECTED_TAG UPIT_PROTECTED_RELEASE

.DEFAULT_GOAL := build
.PHONY: build package generate-desktop-bindings run test test-race vet fmt clean help

## build: Build only the native standalone CLI
build: $(BUILD_DIR)
	go build -ldflags "-X github.com/HungNth/upit/internal/version.version=$(VERSION)" -o "$(TARGET)" $(CMD_DIR)
	@echo Built $(TARGET) for $(DETECTED_OS)

$(BUILD_DIR):
	@$(MKDIR_CMD)

## package: Build the complete native Desktop installer using temporary private staging
package:
ifeq ($(DETECTED_OS),Darwin)
	bash packaging/macos/build.sh --version "$(VERSION)" $(if $(MACOS_SIGNING_IDENTITY),--signing-identity "$(MACOS_SIGNING_IDENTITY)",) $(if $(MACOS_NOTARY_PROFILE),--notary-profile "$(MACOS_NOTARY_PROFILE)",) $(if $(MACOS_PROTECTED_TAG),--protected-tag "$(MACOS_PROTECTED_TAG)",)
else ifeq ($(DETECTED_OS),Windows)
	powershell -NoProfile -ExecutionPolicy Bypass -File packaging/windows/build.ps1 -Version "$(VERSION)"
else
	@echo Desktop packaging is not supported on $(DETECTED_OS). Use make build for the native CLI.
	@exit 1
endif

## generate-desktop-bindings: Explicit maintenance after exported Go interfaces change
generate-desktop-bindings:
	wails3 generate bindings $(DESKTOP_CMD_DIR) -i -d "$(DESKTOP_FRONTEND_DIR)/bindings"

## run: Build and run the CLI
run: build
	@$(RUN_CMD) $(ARGS)

## test: Run all Go tests
test:
	go test -v ./...

## test-race: Run all Go tests with the race detector
test-race:
	go test -v -race ./...

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format Go source
fmt:
	go fmt ./...

## clean: Remove the CLI, owned package staging, and distribution output
clean:
	@$(CLEAN_CMD)
	@echo Cleaned $(BUILD_DIR), .build, and dist for $(DETECTED_OS)

## help: Show the public command contract
help:
	@echo Usage: make [target]
	@echo "  build                     Native standalone CLI ($(TARGET))"
	@echo "  package                   Native Desktop installer (VERSION=$(VERSION))"
	@echo "  run ARGS=...              Build and run the CLI"
	@echo "  generate-desktop-bindings Regenerate committed Wails bindings"
	@echo "  test                      Run Go tests"
	@echo "  test-race                 Run Go tests with the race detector"
	@echo "  vet                       Run go vet"
	@echo "  fmt                       Format Go source"
	@echo "  clean                     Remove bin, .build, and dist"
