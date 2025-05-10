package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OllamaClient provides an interface to the Ollama API
type OllamaClient struct {
	BaseURL    string
	ModelName  string
	MaxTokens  int
	HttpClient *http.Client
}

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model     string                 `json:"model"`
	Prompt    string                 `json:"prompt"`
	System    string                 `json:"system,omitempty"`
	Context   []int                  `json:"context,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	Stream    bool                   `json:"stream"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Context  []int  `json:"context,omitempty"`
	Done     bool   `json:"done"`
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL string, modelName string) *OllamaClient {
	return &OllamaClient{
		BaseURL:   baseURL,
		ModelName: modelName,
		MaxTokens: 2048,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsAvailable checks if the Ollama server is available
func (c *OllamaClient) IsAvailable() bool {
	_, err := c.HttpClient.Get(c.BaseURL + "/api/tags")
	return err == nil
}

// Generate sends a prompt to Ollama and returns the response
func (c *OllamaClient) Generate(prompt string, systemPrompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  c.ModelName,
		Prompt: prompt,
		System: systemPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.1,
			"num_predict": 64,
		},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	
	resp, err := c.HttpClient.Post(
		c.BaseURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	// Remove extra whitespace and make sure the response is a single line
	response := strings.TrimSpace(result.Response)
	response = strings.ReplaceAll(response, "\n", " ")
	
	return response, nil
}

// CheckModelAvailability checks if the specified model is available
func (c *OllamaClient) CheckModelAvailability() (bool, error) {
	resp, err := c.HttpClient.Get(c.BaseURL + "/api/tags")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	
	for _, model := range result.Models {
		if model.Name == c.ModelName {
			return true, nil
		}
	}
	
	return false, nil
}

// DownloadModel initiates the download of the specified model
func (c *OllamaClient) DownloadModel() error {
	downloadReq := struct {
		Name string `json:"name"`
	}{Name: c.ModelName}
	
	jsonData, err := json.Marshal(downloadReq)
	if err != nil {
		return err
	}
	
	resp, err := c.HttpClient.Post(
		c.BaseURL+"/api/pull",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Check if the response indicates an error
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error != "" {
			return fmt.Errorf("ollama error: %s", errorResp.Error)
		}
		return fmt.Errorf("ollama error: status code %d", resp.StatusCode)
	}
	
	return nil
}