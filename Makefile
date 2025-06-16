# Define the name of the binary
BINARY_NAME = delta

# Define the Go compiler flags
GOFLAGS := -v

# Version information (can be overridden)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.4.0-alpha")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
IS_DIRTY ?= $(shell git diff --quiet 2>/dev/null; if [ $$? -eq 1 ]; then echo "true"; else echo "false"; fi)

# Define ldflags for version injection
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -X main.BuildDate=$(BUILD_DATE) \
           -X main.IsDirty=$(IS_DIRTY)

# Define the directory for compiled binaries (optional)
BUILD_DIR = build

# Define the target architecture and OS (e.g., linux/amd64, windows/amd64, darwin/amd64)
TARGET_ARCH ?= amd64
TARGET_OS ?= $(shell go env GOOS)

# Define the full target
TARGET := $(TARGET_OS)/$(TARGET_ARCH)

# Define the output binary path (add .exe for Windows)
ifeq ($(TARGET_OS),windows)
	OUTPUT_BINARY = $(BUILD_DIR)/$(TARGET)/$(BINARY_NAME).exe
else
	OUTPUT_BINARY = $(BUILD_DIR)/$(TARGET)/$(BINARY_NAME)
endif

# Define all supported platforms
PLATFORMS = linux/amd64 darwin/amd64 darwin/arm64 windows/amd64

# SQLite vector extension version
SQLITE_VEC_VERSION = v0.1.6
SQLITE_VEC_URL = https://github.com/asg017/sqlite-vec/releases/download/$(SQLITE_VEC_VERSION)/sqlite-vec-0.1.6-loadable-linux-x86_64.tar.gz

# Define Go source files
GO_SOURCES = version.go \
	ai.go ai_manager.go \
	art2_manager.go art2_preprocessor.go art2_commands.go \
	jump_manager.go jump_helper.go \
	cli.go help.go \
	i18n_manager.go i18n_commands.go \
	memory_manager.go memory_commands.go \
	tokenizer.go tokenizer_commands.go \
	inference.go inference_commands.go \
	vector_db.go vector_commands.go \
	onnx_runtime.go onnx_runtime_test.go \
	embedding_manager.go embedding_commands.go \
	speculative_decoding.go speculative_commands.go \
	knowledge_extractor.go knowledge_commands.go knowledge_extractor_agent_command.go \
	agent_types.go agent_manager.go agent_commands.go \
	config_manager.go config_commands.go \
	version_manager.go update_manager.go update_commands.go github_client.go update_checker.go update_downloader.go update_installer.go update_ui.go update_scheduler.go update_history.go update_validation.go \
	spellcheck.go spellcheck_commands.go \
	history_analysis.go history_commands.go \
	pattern_update.go pattern_commands.go pattern_recognition.go \
	error_learning.go

all: deps build

deps: vec0.so

vec0.so:
	@if [ ! -f vec0.so ]; then \
		echo "Downloading SQLite vector extension $(SQLITE_VEC_VERSION)..."; \
		curl -L -o sqlite-vec.tar.gz $(SQLITE_VEC_URL); \
		tar -xzf sqlite-vec.tar.gz; \
		rm -f sqlite-vec.tar.gz; \
		echo "SQLite vector extension downloaded successfully"; \
	else \
		echo "SQLite vector extension already exists locally"; \
	fi

build:
	@echo "Building $(BINARY_NAME) for $(TARGET)"
	@echo "Version: $(VERSION), Commit: $(GIT_COMMIT), Date: $(BUILD_DATE), Dirty: $(IS_DIRTY)"
	@mkdir -p $(dir $(OUTPUT_BINARY))
	CGO_ENABLED=1 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(OUTPUT_BINARY) $(GO_SOURCES)
	@echo "Successfully built $(BINARY_NAME) for $(TARGET)"

# Cross-compilation targets
build-all: $(PLATFORMS)

$(PLATFORMS):
	@echo "Building $(BINARY_NAME) for $@"
	@mkdir -p $(BUILD_DIR)/$@
	$(eval OS := $(word 1, $(subst /, ,$@)))
	$(eval ARCH := $(word 2, $(subst /, ,$@)))
	$(eval BINARY_EXT := $(if $(filter $(OS),windows),.exe,))
	$(eval OUTPUT := $(BUILD_DIR)/$@/$(BINARY_NAME)$(BINARY_EXT))
	@if [ "$(OS)" = "linux" ]; then \
		CGO_ENABLED=1 GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(OUTPUT) $(GO_SOURCES); \
	else \
		CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(OUTPUT) $(GO_SOURCES); \
	fi
	@echo "Successfully built $(BINARY_NAME) for $@"

clean:
	@echo "Cleaning up build directory and dependencies"
	@rm -rf $(BUILD_DIR)
	@rm -f vec0.so

run: build
	@echo "Running $(BINARY_NAME)"
	@./$(OUTPUT_BINARY)

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)"
	@sudo cp $(OUTPUT_BINARY) /usr/local/bin/$(BINARY_NAME)
	@chmod +x /usr/local/bin/$(BINARY_NAME)

# Show version information that will be injected
version-info:
	@echo "Version Information:"
	@echo "  VERSION: $(VERSION)"
	@echo "  GIT_COMMIT: $(GIT_COMMIT)"
	@echo "  BUILD_DATE: $(BUILD_DATE)"
	@echo "  IS_DIRTY: $(IS_DIRTY)"
	@echo "  LDFLAGS: $(LDFLAGS)"

# Release target - creates a full release
release:
	@if [ -z "$(RELEASE_VERSION)" ]; then \
		echo "Error: RELEASE_VERSION not specified"; \
		echo "Usage: make release RELEASE_VERSION=v0.4.2-alpha"; \
		exit 1; \
	fi
	@echo "Creating release for version $(RELEASE_VERSION)"
	@echo "Step 1: Creating release notes..."
	@mkdir -p RELEASE_NOTES
	@echo "# Release Notes for $(RELEASE_VERSION)" > RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "## 🚀 Highlights" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "TODO: Add release highlights here" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "## 📦 What's New" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "TODO: Add new features here" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "" >> RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@echo "Release notes template created at: RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md"
	@echo "Please edit the release notes before continuing..."
	@echo ""
	@echo "When ready, run the following commands:"
	@echo "  1. git add RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md"
	@echo "  2. git commit -m 'feat: prepare $(RELEASE_VERSION) release'"
	@echo "  3. git tag -a $(RELEASE_VERSION) -m 'Release $(RELEASE_VERSION)'"
	@echo "  4. git push origin main"
	@echo "  5. git push origin $(RELEASE_VERSION)"
	@echo "  6. ./scripts/create-release.sh $(RELEASE_VERSION)"
	@echo ""
	@echo "Or use: make release-auto RELEASE_VERSION=$(RELEASE_VERSION)"

# Automated release process (use with caution)
release-auto:
	@if [ -z "$(RELEASE_VERSION)" ]; then \
		echo "Error: RELEASE_VERSION not specified"; \
		echo "Usage: make release-auto RELEASE_VERSION=v0.4.2-alpha"; \
		exit 1; \
	fi
	@echo "Automated release process for $(RELEASE_VERSION)"
	@echo "WARNING: This will create tags and push to GitHub!"
	@read -p "Continue? (y/N): " -n 1 -r; \
	echo; \
	if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Release cancelled"; \
		exit 1; \
	fi
	@echo "Creating and committing release notes..."
	@$(MAKE) release RELEASE_VERSION=$(RELEASE_VERSION)
	@echo "Edit the release notes file, then press Enter to continue..."
	@read -p "Press Enter when release notes are ready: "
	@git add RELEASE_NOTES/RELEASE_NOTES_$(RELEASE_VERSION).md
	@git commit -m "feat: prepare $(RELEASE_VERSION) release"
	@git tag -a $(RELEASE_VERSION) -m "Release $(RELEASE_VERSION)"
	@git push origin main
	@git push origin $(RELEASE_VERSION)
	@./scripts/create-release.sh $(RELEASE_VERSION)

.PHONY: all deps clean run install build build-all version-info release release-auto $(PLATFORMS)
