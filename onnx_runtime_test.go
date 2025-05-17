package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestONNXRuntimeIntegration(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "onnx-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test model and vocab files
	modelPath := filepath.Join(tempDir, "model.onnx")
	vocabPath := filepath.Join(tempDir, "vocab.txt")

	// Create a simple model file
	modelContent := "ONNX model placeholder content for testing"
	err = os.WriteFile(modelPath, []byte(modelContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test model file: %v", err)
	}

	// Create a simple vocabulary file
	vocabContent := "<unk>\n<s>\n</s>\n<pad>\nthe\na\nto\nand\nin\n"
	err = os.WriteFile(vocabPath, []byte(vocabContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test vocab file: %v", err)
	}

	// Create ONNX runtime configuration
	config := ONNXRuntimeConfig{
		ModelPath:   modelPath,
		VocabPath:   vocabPath,
		Dimension:   384,
		MaxLength:   128,
		BatchSize:   4,
		NumThreads:  4,
		UseGPU:      false,
		UseFP16:     false,
		GPUDeviceID: 0,
	}

	// Create and initialize the ONNX runtime
	runtime, err := NewONNXRuntime(config)
	if err != nil {
		t.Fatalf("Failed to create ONNX runtime: %v", err)
	}

	// Initialize the runtime
	err = runtime.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize ONNX runtime: %v", err)
	}

	// Test generating an embedding
	text := "This is a test sentence for embedding generation"
	embedding, err := runtime.GenerateEmbedding(text)
	if err != nil {
		t.Fatalf("Failed to generate embedding: %v", err)
	}

	// Verify embedding dimensions
	if len(embedding) != config.Dimension {
		t.Errorf("Expected embedding dimension %d, got %d", config.Dimension, len(embedding))
	}

	// Verify embedding normalization (L2 norm should be close to 1)
	var sumSquared float32
	for _, v := range embedding {
		sumSquared += v * v
	}
	if sumSquared < 0.99 || sumSquared > 1.01 {
		t.Errorf("Embedding not properly normalized, L2 norm: %f", sumSquared)
	}

	// Test vocabulary loading
	if len(runtime.vocabulary) != 9 {
		t.Errorf("Expected 9 vocabulary tokens, got %d", len(runtime.vocabulary))
	}

	// Verify specific tokens exist
	requiredTokens := []string{"<unk>", "<s>", "</s>", "<pad>", "the"}
	for _, token := range requiredTokens {
		if _, ok := runtime.vocabulary[token]; !ok {
			t.Errorf("Expected token '%s' not found in vocabulary", token)
		}
	}

	// Test tokenization
	tokens := runtime.tokenizeText("the cat and the dog")
	if len(tokens) != config.MaxLength {
		t.Errorf("Expected token length to be MaxLength (%d), got %d", config.MaxLength, len(tokens))
	}

	// Close the runtime
	err = runtime.Close()
	if err != nil {
		t.Errorf("Failed to close ONNX runtime: %v", err)
	}
}

func TestONNXModelDownloader(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "onnx-downloader-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test the model downloader
	modelPath := filepath.Join(tempDir, "model.onnx")
	modelURL := "https://example.com/model.onnx"

	err = DownloadONNXEmbeddingModel(modelPath, modelURL)
	if err != nil {
		t.Fatalf("Failed to download model: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Errorf("Model file was not created")
	}

	// Test vocab downloader
	vocabPath := filepath.Join(tempDir, "vocab.txt")
	vocabURL := "https://example.com/vocab.txt"

	err = DownloadONNXEmbeddingModel(vocabPath, vocabURL)
	if err != nil {
		t.Fatalf("Failed to download vocabulary: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(vocabPath); os.IsNotExist(err) {
		t.Errorf("Vocabulary file was not created")
	}

	// Read the vocab file to verify it contains expected content
	vocabData, err := os.ReadFile(vocabPath)
	if err != nil {
		t.Fatalf("Failed to read vocabulary file: %v", err)
	}

	vocabContent := string(vocabData)
	if vocabContent == "" {
		t.Errorf("Vocabulary file is empty")
	}
}
