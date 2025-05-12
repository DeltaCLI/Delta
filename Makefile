# Define the name of the binary
BINARY_NAME = delta

# Define the Go compiler flags
GOFLAGS := -v

# Define the directory for compiled binaries (optional)
BUILD_DIR = build

# Define the target architecture and OS (e.g., linux/amd64, windows/amd64, darwin/amd64)
TARGET_ARCH ?= amd64
TARGET_OS ?= $(shell go env GOOS)

# Define the full target
TARGET := $(TARGET_OS)/$(TARGET_ARCH)

# Define the output binary path
OUTPUT_BINARY = $(BUILD_DIR)/$(TARGET)/$(BINARY_NAME)

all: build

build: $(OUTPUT_BINARY)
	@echo "Building $(BINARY_NAME) for $(TARGET)"
	@mkdir -p $(dir $(OUTPUT_BINARY))
	CGO_ENABLED=0 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) go build $(GOFLAGS) -o $(OUTPUT_BINARY) \
		ai.go ai_manager.go \
		jump_manager.go jump_helper.go \
		cli.go help.go \
		memory_manager.go memory_commands.go \
		tokenizer.go tokenizer_commands.go \
		inference.go inference_commands.go \
		vector_db.go vector_commands.go \
		embedding_manager.go embedding_commands.go \
		speculative_decoding.go speculative_commands.go \
		knowledge_extractor.go knowledge_commands.go \
		agent_manager.go agent_commands.go \
		config_manager.go config_commands.go \
		spellcheck.go spellcheck_commands.go \
		history_analysis.go history_commands.go
	@echo "Successfully built $(BINARY_NAME) for $(TARGET)"

clean:
	@echo "Cleaning up build directory"
	@rm -rf $(BUILD_DIR)

run: build
	@echo "Running $(BINARY_NAME)"
	@./$(OUTPUT_BINARY)

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)"
	@sudo cp $(OUTPUT_BINARY) /usr/local/bin/$(BINARY_NAME)
	@chmod +x /usr/local/bin/$(BINARY_NAME)

.PHONY: all clean run install
