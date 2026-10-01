BINARY_NAME := upit
CMD_DIR := ./cmd/upit
DESKTOP_BINARY_NAME := upit-desktop
FILE_MANAGER_HELPER_NAME := upit-file-manager
DESKTOP_CMD_DIR := ./cmd/upit-desktop
FILE_MANAGER_HELPER_CMD_DIR := ./cmd/upit-file-manager
DESKTOP_FRONTEND_DIR := $(DESKTOP_CMD_DIR)/frontend
BUILD_DIR := bin
MACOS_VERSION ?= 0.7.0

# Detect OS and set OS-specific commands
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    BINARY_EXT := .exe

    # Force GNU Make to use Windows cmd.exe
    SHELL := cmd.exe
    .SHELLFLAGS := /C

    MKDIR_CMD := if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
    CLEAN_CMD := if exist "$(BUILD_DIR)" rmdir /S /Q "$(BUILD_DIR)"
else
    DETECTED_OS := $(shell uname -s)
    BINARY_EXT :=
    MKDIR_CMD := mkdir -p "$(BUILD_DIR)"
    CLEAN_CMD := rm -rf "$(BUILD_DIR)"
endif

TARGET := $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT)
DESKTOP_TARGET := $(BUILD_DIR)/$(DESKTOP_BINARY_NAME)$(BINARY_EXT)
FILE_MANAGER_HELPER_TARGET := $(BUILD_DIR)/$(FILE_MANAGER_HELPER_NAME)$(BINARY_EXT)

# Command used to run the binary
ifeq ($(OS),Windows_NT)
    RUN_CMD := "$(TARGET)"
else
    RUN_CMD := ./$(TARGET)
endif

.PHONY: all build setup-desktop build-desktop-frontend generate-desktop-bindings build-file-manager-helper build-desktop run test test-race vet fmt clean help build-all build-windows build-linux build-darwin build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-macos package-macos validate-macos

all: build

$(BUILD_DIR):
	@$(MKDIR_CMD)

## build: Build the binary for the current operating system
build: $(BUILD_DIR)
	go build -o "$(TARGET)" $(CMD_DIR)
	@echo Built $(TARGET) for $(DETECTED_OS)

## setup-desktop: Install desktop frontend dependencies from the lockfile
setup-desktop:
	npm --prefix "$(DESKTOP_FRONTEND_DIR)" ci

## build-desktop-frontend: Type-check and bundle the desktop frontend
build-desktop-frontend:
	npm --prefix "$(DESKTOP_FRONTEND_DIR)" run build

## generate-desktop-bindings: Regenerate Wails bindings after exported Go interfaces change
generate-desktop-bindings:
	wails3 generate bindings $(DESKTOP_CMD_DIR) -i -d "$(DESKTOP_FRONTEND_DIR)/bindings"

## build-file-manager-helper: Build the Wails-free File Manager Upload helper
build-file-manager-helper: $(BUILD_DIR)
	go build -o "$(FILE_MANAGER_HELPER_TARGET)" $(FILE_MANAGER_HELPER_CMD_DIR)
	@echo Built $(FILE_MANAGER_HELPER_TARGET) for $(DETECTED_OS)

## build-desktop: Rebuild frontend assets, helper, then build upit-desktop for the current operating system
build-desktop: build-desktop-frontend build-file-manager-helper $(BUILD_DIR)
	go build -o "$(DESKTOP_TARGET)" $(DESKTOP_CMD_DIR)
	@echo Built $(DESKTOP_TARGET) for $(DETECTED_OS)

## run: Build and run binary
run: build
	@$(RUN_CMD) $(ARGS)

## Cross-compilation targets (Go cross-compiles natively via exported GOOS/GOARCH)
build-windows: export GOOS := windows
build-windows: export GOARCH := amd64
build-windows: $(BUILD_DIR)
	go build -o "$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe" $(CMD_DIR)

build-linux-amd64: export GOOS := linux
build-linux-amd64: export GOARCH := amd64
build-linux-amd64: $(BUILD_DIR)
	go build -o "$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64" $(CMD_DIR)

build-linux-arm64: export GOOS := linux
build-linux-arm64: export GOARCH := arm64
build-linux-arm64: $(BUILD_DIR)
	go build -o "$(BUILD_DIR)/$(BINARY_NAME)-linux-arm64" $(CMD_DIR)

build-linux: build-linux-amd64 build-linux-arm64

build-darwin-amd64: export GOOS := darwin
build-darwin-amd64: export GOARCH := amd64
build-darwin-amd64: $(BUILD_DIR)
	go build -o "$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64" $(CMD_DIR)

build-darwin-arm64: export GOOS := darwin
build-darwin-arm64: export GOARCH := arm64
build-darwin-arm64: $(BUILD_DIR)
	go build -o "$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64" $(CMD_DIR)

build-darwin: build-darwin-amd64 build-darwin-arm64

## build-macos: Build the macOS 14+ Apple Silicon CLI, helper, and desktop binaries
build-macos: export GOOS := darwin
build-macos: export GOARCH := arm64
build-macos: export CGO_ENABLED := 1
build-macos: build-desktop
	go build -o "$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64" $(CMD_DIR)

## package-macos: Build an unsigned or protected signed/notarized macOS package
package-macos: build-macos
	packaging/macos/package.sh --version "$(MACOS_VERSION)" $(if $(MACOS_SIGNING_IDENTITY),--signing-identity "$(MACOS_SIGNING_IDENTITY)",) $(if $(MACOS_NOTARY_PROFILE),--notary-profile "$(MACOS_NOTARY_PROFILE)",) $(if $(MACOS_PROTECTED_TAG),--protected-tag "$(MACOS_PROTECTED_TAG)",)

## validate-macos: Validate an extracted Upit.app bundle
validate-macos:
	packaging/macos/validate.sh --app "$(MACOS_APP)" $(if $(MACOS_REQUIRE_SIGNATURE),--require-signature,)


build-all: build-windows build-linux build-darwin

## test: Run all test suites
test:
	go test -v ./...

## test-race: Run all test suites with the race detector
test-race:
	go test -v -race ./...

## vet: Run go vet to check the code
vet:
	go vet ./...

## fmt: Format all source code
fmt:
	go fmt ./...

## clean: Remove the build directory using the appropriate OS command
clean:
	@$(CLEAN_CMD)
	@echo Cleaned $(BUILD_DIR) for $(DETECTED_OS)

## help: Show usage information
help:
	@echo Usage: make [target]
	@echo Targets:
	@echo "  build                    Build binary for current host OS ($(TARGET))"
	@echo "  build-all                Cross-compile for Windows, Linux, and macOS"
	@echo "  setup-desktop            Install desktop frontend dependencies from package-lock.json"
	@echo "  build-desktop-frontend   Type-check and bundle desktop frontend assets"
	@echo "  build-file-manager-helper Build Wails-free File Manager Upload helper"
	@echo "  generate-desktop-bindings Regenerate Wails bindings after exported Go interfaces change"
	@echo "  build-desktop            Build frontend assets, helper, and upit-desktop ($(DESKTOP_TARGET))"
	@echo "  build-windows  Build binary for Windows (amd64)"
	@echo "  build-linux    Build binaries for Linux (amd64, arm64)"
	@echo "  build-darwin   Build binaries for macOS (amd64, arm64)"
	@echo "  build-macos   Build macOS 14+ Apple Silicon CLI, helper, and desktop"
	@echo "  package-macos Build unsigned or protected macOS DMG"
	@echo "  validate-macos Validate an extracted Upit.app bundle"
	@echo "  test           Run tests"
	@echo "  test-race      Run tests with -race"
	@echo "  vet            Run go vet"
	@echo "  fmt            Run go fmt"
	@echo "  clean          Remove build directory ($(BUILD_DIR))"