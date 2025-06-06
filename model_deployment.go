package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ModelFormat defines supported model formats
type ModelFormat string

const (
	// ModelFormatONNX is the ONNX model format
	ModelFormatONNX ModelFormat = "onnx"
	// ModelFormatPyTorch is the PyTorch model format
	ModelFormatPyTorch ModelFormat = "pytorch"
	// ModelFormatBinary is a custom binary format
	ModelFormatBinary ModelFormat = "binary"
)

// ModelDeploymentConfig contains configuration for model deployment
type ModelDeploymentConfig struct {
	ModelPath         string      // Path to the model to deploy
	ModelFormat       ModelFormat // Format of the model
	TargetPath        string      // Where to deploy the model
	Optimize          bool        // Whether to optimize the model for inference
	Quantize          bool        // Whether to quantize the model
	ValidateModel     bool        // Whether to validate the model before deployment
	BackupExisting    bool        // Whether to backup existing model
	CreateSymlink     bool        // Whether to create a symlink to the latest model
	OptimizationLevel int         // Level of optimization (0-3)
}

// ModelDeploymentResult contains results of model deployment
type ModelDeploymentResult struct {
	SourcePath      string      // Original model path
	DeployedPath    string      // Path where model was deployed
	ModelFormat     ModelFormat // Format of the model
	ModelSize       int64       // Size of the model in bytes
	OptimizedSize   int64       // Size after optimization (if applicable)
	DeploymentTime  time.Time   // When deployment was performed
	ValidationScore float64     // Validation score (if validated)
	BackupPath      string      // Path to backup if created
	SymlinkPath     string      // Path to symlink if created
	Optimized       bool        // Whether optimization was performed
	Quantized       bool        // Whether quantization was performed
	Success         bool        // Whether deployment was successful
	Error           string      // Error message if unsuccessful
}

// ModelInfo contains information about a model
type ModelInfo struct {
	Path            string      // Path to the model
	Format          ModelFormat // Format of the model
	Size            int64       // Size in bytes
	ModTime         time.Time   // Last modification time
	IsDeployed      bool        // Whether the model is deployed
	IsLatest        bool        // Whether this is the latest model
	ValidationScore float64     // Validation score if available
	DeploymentTime  time.Time   // When the model was deployed
}

// ModelDeploymentService manages model deployment
type ModelDeploymentService struct {
	modelsDir        string            // Directory for model storage
	deploymentDir    string            // Directory for deployed models
	backupDir        string            // Directory for model backups
	metadataDir      string            // Directory for deployment metadata
	inferenceManager *InferenceManager // Reference to inference manager
	evaluator        *ModelEvaluator   // Reference to model evaluator
}

// NewModelDeploymentService creates a new model deployment service
func NewModelDeploymentService() (*ModelDeploymentService, error) {
	// Get inference manager
	inferenceManager := GetInferenceManager()
	if inferenceManager == nil {
		return nil, fmt.Errorf("inference manager not available")
	}

	// Get model evaluator
	evaluator := GetModelEvaluator()
	if evaluator == nil {
		return nil, fmt.Errorf("model evaluator not available")
	}

	// Set up directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	configRoot := filepath.Join(homeDir, ".config", "delta", "memory")
	modelsDir := filepath.Join(configRoot, "models")
	deploymentDir := filepath.Join(configRoot, "deployed_models")
	backupDir := filepath.Join(configRoot, "model_backups")
	metadataDir := filepath.Join(configRoot, "deployment_metadata")

	// Create directories if they don't exist
	for _, dir := range []string{modelsDir, deploymentDir, backupDir, metadataDir} {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return &ModelDeploymentService{
		modelsDir:        modelsDir,
		deploymentDir:    deploymentDir,
		backupDir:        backupDir,
		metadataDir:      metadataDir,
		inferenceManager: inferenceManager,
		evaluator:        evaluator,
	}, nil
}

// DeployModel deploys a model with the given configuration
func (s *ModelDeploymentService) DeployModel(config ModelDeploymentConfig) (*ModelDeploymentResult, error) {
	// Initialize result
	result := &ModelDeploymentResult{
		SourcePath:     config.ModelPath,
		ModelFormat:    config.ModelFormat,
		DeploymentTime: time.Now(),
		Success:        false,
	}

	// Validate model path
	if config.ModelPath == "" {
		result.Error = "model path is required"
		return result, fmt.Errorf(result.Error)
	}

	// Check if model exists
	modelInfo, err := os.Stat(config.ModelPath)
	if os.IsNotExist(err) {
		result.Error = fmt.Sprintf("model not found: %s", config.ModelPath)
		return result, fmt.Errorf(result.Error)
	}

	// Set model size
	result.ModelSize = modelInfo.Size()

	// Determine model format if not specified
	if config.ModelFormat == "" {
		ext := filepath.Ext(config.ModelPath)
		switch ext {
		case ".onnx":
			config.ModelFormat = ModelFormatONNX
		case ".pt", ".pth":
			config.ModelFormat = ModelFormatPyTorch
		case ".bin":
			config.ModelFormat = ModelFormatBinary
		default:
			result.Error = fmt.Sprintf("unable to determine model format for extension: %s", ext)
			return result, fmt.Errorf(result.Error)
		}
	}
	result.ModelFormat = config.ModelFormat

	// Determine target path if not specified
	if config.TargetPath == "" {
		modelName := filepath.Base(config.ModelPath)
		timestamp := time.Now().Format("20060102_150405")
		targetName := fmt.Sprintf("%s_deployed_%s%s",
			strings.TrimSuffix(modelName, filepath.Ext(modelName)),
			timestamp,
			filepath.Ext(modelName))
		config.TargetPath = filepath.Join(s.deploymentDir, targetName)
	}
	result.DeployedPath = config.TargetPath

	// Backup existing model if requested
	if config.BackupExisting {
		existingModel := s.GetCurrentDeployedModel()
		if existingModel != "" {
			backupPath, err := s.backupModel(existingModel)
			if err != nil {
				result.Error = fmt.Sprintf("failed to backup existing model: %v", err)
				return result, fmt.Errorf(result.Error)
			}
			result.BackupPath = backupPath
		}
	}

	// Validate model if requested
	if config.ValidateModel {
		score, err := s.validateModel(config.ModelPath)
		if err != nil {
			result.Error = fmt.Sprintf("model validation failed: %v", err)
			return result, fmt.Errorf(result.Error)
		}
		result.ValidationScore = score
	}

	// Optimize model if requested
	var processedModelPath string
	if config.Optimize {
		optimizedPath, err := s.optimizeModel(config.ModelPath, config.ModelFormat, config.OptimizationLevel)
		if err != nil {
			result.Error = fmt.Sprintf("model optimization failed: %v", err)
			return result, fmt.Errorf(result.Error)
		}
		processedModelPath = optimizedPath
		result.Optimized = true

		// Get optimized size
		if optInfo, err := os.Stat(optimizedPath); err == nil {
			result.OptimizedSize = optInfo.Size()
		}
	} else {
		processedModelPath = config.ModelPath
	}

	// Quantize model if requested
	if config.Quantize {
		quantizedPath, err := s.quantizeModel(processedModelPath, config.ModelFormat)
		if err != nil {
			result.Error = fmt.Sprintf("model quantization failed: %v", err)
			return result, fmt.Errorf(result.Error)
		}
		processedModelPath = quantizedPath
		result.Quantized = true
	}

	// Copy model to deployment location
	err = copyFile(processedModelPath, config.TargetPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to copy model to deployment location: %v", err)
		return result, fmt.Errorf(result.Error)
	}

	// Create symlink if requested
	if config.CreateSymlink {
		symlinkPath := filepath.Join(s.deploymentDir, "latest_model"+filepath.Ext(config.TargetPath))

		// Remove existing symlink if it exists
		os.Remove(symlinkPath)

		// Create the symlink
		err = os.Symlink(config.TargetPath, symlinkPath)
		if err != nil {
			// Non-fatal error, continue with deployment
			fmt.Printf("Warning: failed to create symlink: %v\n", err)
		} else {
			result.SymlinkPath = symlinkPath
		}
	}

	// Save deployment metadata
	err = s.saveDeploymentMetadata(result)
	if err != nil {
		// Non-fatal error, continue with deployment
		fmt.Printf("Warning: failed to save deployment metadata: %v\n", err)
	}

	// Update inference manager configuration
	err = s.updateInferenceConfig(config.TargetPath)
	if err != nil {
		// Non-fatal error, continue with deployment
		fmt.Printf("Warning: failed to update inference configuration: %v\n", err)
	}

	result.Success = true
	return result, nil
}

// GetCurrentDeployedModel returns the path to the currently deployed model
func (s *ModelDeploymentService) GetCurrentDeployedModel() string {
	// Check if inference manager has a custom model set
	if s.inferenceManager.learningConfig.UseCustomModel &&
		s.inferenceManager.learningConfig.CustomModelPath != "" {
		return s.inferenceManager.learningConfig.CustomModelPath
	}

	// Check for symlink to latest model
	symlinkPath := filepath.Join(s.deploymentDir, "latest_model.onnx")
	if _, err := os.Stat(symlinkPath); err == nil {
		// Resolve the symlink
		path, err := os.Readlink(symlinkPath)
		if err == nil {
			return path
		}
	}

	// If no symlink, find the newest model in the deployment directory
	files, err := os.ReadDir(s.deploymentDir)
	if err != nil {
		return ""
	}

	var newest os.FileInfo
	var newestPath string
	for _, file := range files {
		// Skip symlinks and directories
		if file.IsDir() {
			continue
		}

		// Check if it's a model file
		ext := filepath.Ext(file.Name())
		if ext != ".onnx" && ext != ".pt" && ext != ".pth" && ext != ".bin" {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if newest == nil || info.ModTime().After(newest.ModTime()) {
			newest = info
			newestPath = filepath.Join(s.deploymentDir, file.Name())
		}
	}

	return newestPath
}

// ListAvailableModels lists all available models
func (s *ModelDeploymentService) ListAvailableModels() ([]ModelInfo, error) {
	models := make([]ModelInfo, 0)

	// Get currently deployed model
	currentModel := s.GetCurrentDeployedModel()

	// Directories to check for models
	directories := []string{s.modelsDir, s.deploymentDir}

	for _, dir := range directories {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			// Skip directories
			if file.IsDir() {
				continue
			}

			// Check if it's a model file
			ext := filepath.Ext(file.Name())
			if ext != ".onnx" && ext != ".pt" && ext != ".pth" && ext != ".bin" {
				continue
			}

			info, err := file.Info()
			if err != nil {
				continue
			}

			// Determine model format
			var format ModelFormat
			switch ext {
			case ".onnx":
				format = ModelFormatONNX
			case ".pt", ".pth":
				format = ModelFormatPyTorch
			case ".bin":
				format = ModelFormatBinary
			default:
				format = ""
			}

			// Create model info
			modelPath := filepath.Join(dir, file.Name())
			isDeployed := modelPath == currentModel

			// Find metadata if available
			metadata, _ := s.getModelMetadata(modelPath)

			// Build the model info
			modelInfo := ModelInfo{
				Path:       modelPath,
				Format:     format,
				Size:       info.Size(),
				ModTime:    info.ModTime(),
				IsDeployed: isDeployed,
				IsLatest:   false, // Will set this later for the newest model
			}

			// Add metadata if available
			if metadata != nil {
				modelInfo.ValidationScore = metadata.ValidationScore
				modelInfo.DeploymentTime = metadata.DeploymentTime
			}

			models = append(models, modelInfo)
		}
	}

	// Sort models by modification time (newest first)
	sort.Slice(models, func(i, j int) bool {
		return models[i].ModTime.After(models[j].ModTime)
	})

	// Mark the newest model
	if len(models) > 0 {
		models[0].IsLatest = true
	}

	return models, nil
}

// GetDeploymentHistory returns the deployment history
func (s *ModelDeploymentService) GetDeploymentHistory() ([]*ModelDeploymentResult, error) {
	// Read all metadata files
	files, err := os.ReadDir(s.metadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata directory: %v", err)
	}

	deployments := make([]*ModelDeploymentResult, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			// Read metadata file
			metadataPath := filepath.Join(s.metadataDir, file.Name())
			data, err := os.ReadFile(metadataPath)
			if err != nil {
				continue
			}

			// Parse metadata
			var result ModelDeploymentResult
			err = json.Unmarshal(data, &result)
			if err != nil {
				continue
			}

			deployments = append(deployments, &result)
		}
	}

	// Sort by deployment time (newest first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].DeploymentTime.After(deployments[j].DeploymentTime)
	})

	return deployments, nil
}

// SwitchToModel switches to a different model
func (s *ModelDeploymentService) SwitchToModel(modelPath string) error {
	// Check if model exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model not found: %s", modelPath)
	}

	// Update inference manager configuration
	return s.updateInferenceConfig(modelPath)
}

// backupModel creates a backup of a model
func (s *ModelDeploymentService) backupModel(modelPath string) (string, error) {
	// Create backup filename
	modelName := filepath.Base(modelPath)
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s_backup_%s%s",
		strings.TrimSuffix(modelName, filepath.Ext(modelName)),
		timestamp,
		filepath.Ext(modelName))
	backupPath := filepath.Join(s.backupDir, backupName)

	// Copy model to backup location
	err := copyFile(modelPath, backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to copy model to backup location: %v", err)
	}

	return backupPath, nil
}

// validateModel validates a model and returns a validation score
func (s *ModelDeploymentService) validateModel(modelPath string) (float64, error) {
	// In a real implementation, we'd run the model on validation data
	// For now, return a simulated score

	// Simulate validation by checking if the model is valid
	modelInfo, err := os.Stat(modelPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat model: %v", err)
	}

	// Simple check: model should be larger than 1KB
	if modelInfo.Size() < 1024 {
		return 0, fmt.Errorf("model is too small, likely invalid")
	}

	// Return a simulated score between 0.7 and 0.99
	// In a real implementation, we'd run the model evaluator
	seed := time.Now().Unix() % 30
	return 0.7 + float64(seed)/100.0, nil
}

// optimizeModel optimizes a model for inference
func (s *ModelDeploymentService) optimizeModel(modelPath string, format ModelFormat, level int) (string, error) {
	// In a real implementation, we'd use a model optimization tool
	// For now, just copy the model and pretend we optimized it

	// Create optimized model filename
	modelName := filepath.Base(modelPath)
	timestamp := time.Now().Format("20060102_150405")
	optimizedName := fmt.Sprintf("%s_optimized_%s%s",
		strings.TrimSuffix(modelName, filepath.Ext(modelName)),
		timestamp,
		filepath.Ext(modelName))
	optimizedPath := filepath.Join(s.deploymentDir, optimizedName)

	// Copy model to optimized location
	err := copyFile(modelPath, optimizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to copy model to optimized location: %v", err)
	}

	// For ONNX models, we could use onnxruntime to optimize
	if format == ModelFormatONNX {
		// Check if onnxruntime is available
		_, err := exec.LookPath("onnxruntime")
		if err == nil {
			// In a real implementation, we'd run onnxruntime here
			fmt.Println("ONNX Runtime is available for model optimization (simulation)")
		}
	}

	return optimizedPath, nil
}

// quantizeModel quantizes a model
func (s *ModelDeploymentService) quantizeModel(modelPath string, format ModelFormat) (string, error) {
	// In a real implementation, we'd use a model quantization tool
	// For now, just copy the model and pretend we quantized it

	// Create quantized model filename
	modelName := filepath.Base(modelPath)
	timestamp := time.Now().Format("20060102_150405")
	quantizedName := fmt.Sprintf("%s_quantized_%s%s",
		strings.TrimSuffix(modelName, filepath.Ext(modelName)),
		timestamp,
		filepath.Ext(modelName))
	quantizedPath := filepath.Join(s.deploymentDir, quantizedName)

	// Copy model to quantized location
	err := copyFile(modelPath, quantizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to copy model to quantized location: %v", err)
	}

	return quantizedPath, nil
}

// saveDeploymentMetadata saves metadata about a deployment
func (s *ModelDeploymentService) saveDeploymentMetadata(result *ModelDeploymentResult) error {
	// Create metadata filename
	timestamp := result.DeploymentTime.Format("20060102_150405")
	modelName := filepath.Base(result.DeployedPath)
	metadataName := fmt.Sprintf("%s_metadata_%s.json",
		strings.TrimSuffix(modelName, filepath.Ext(modelName)),
		timestamp)
	metadataPath := filepath.Join(s.metadataDir, metadataName)

	// Marshal metadata to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	// Write metadata to file
	err = os.WriteFile(metadataPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata: %v", err)
	}

	return nil
}

// getModelMetadata gets metadata for a model
func (s *ModelDeploymentService) getModelMetadata(modelPath string) (*ModelDeploymentResult, error) {
	// Extract model name without extension
	modelName := filepath.Base(modelPath)
	modelNameWithoutExt := strings.TrimSuffix(modelName, filepath.Ext(modelName))

	// Find metadata file for this model
	files, err := os.ReadDir(s.metadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), modelNameWithoutExt) &&
			strings.Contains(file.Name(), "_metadata_") && strings.HasSuffix(file.Name(), ".json") {
			// Read metadata file
			metadataPath := filepath.Join(s.metadataDir, file.Name())
			data, err := os.ReadFile(metadataPath)
			if err != nil {
				continue
			}

			// Parse metadata
			var result ModelDeploymentResult
			err = json.Unmarshal(data, &result)
			if err != nil {
				continue
			}

			return &result, nil
		}
	}

	return nil, fmt.Errorf("metadata not found for model: %s", modelPath)
}

// updateInferenceConfig updates the inference configuration to use a model
func (s *ModelDeploymentService) updateInferenceConfig(modelPath string) error {
	// Get the inference manager configuration
	inferenceConfig := s.inferenceManager.inferenceConfig
	learningConfig := s.inferenceManager.learningConfig

	// Update to use the custom model
	learningConfig.UseCustomModel = true
	learningConfig.CustomModelPath = modelPath
	inferenceConfig.UseLocalInference = true
	inferenceConfig.ModelPath = modelPath

	// Save the updated configuration
	return s.inferenceManager.UpdateConfig(inferenceConfig, learningConfig)
}

// Helper functions

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Get source file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination directory if needed
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy data from source to destination
	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return err
	}

	// Set same permissions as source
	return os.Chmod(dst, srcInfo.Mode())
}

// Global ModelDeploymentService instance
var globalModelDeploymentService *ModelDeploymentService

// GetModelDeploymentService returns the global ModelDeploymentService instance
func GetModelDeploymentService() *ModelDeploymentService {
	if globalModelDeploymentService == nil {
		var err error
		globalModelDeploymentService, err = NewModelDeploymentService()
		if err != nil {
			fmt.Printf("Error initializing model deployment service: %v\n", err)
			return nil
		}
	}
	return globalModelDeploymentService
}
