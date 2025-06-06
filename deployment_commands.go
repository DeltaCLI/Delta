package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HandleDeploymentCommand processes deployment-related commands
func HandleDeploymentCommand(args []string) bool {
	// Get the model deployment service
	deploymentService := GetModelDeploymentService()
	if deploymentService == nil {
		fmt.Println("Failed to initialize model deployment service")
		return true
	}

	// Handle commands
	if len(args) == 0 {
		// Show deployment status
		showDeploymentStatus(deploymentService)
		return true
	}

	// Handle subcommands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "deploy":
			// Deploy a model
			deployModel(deploymentService, args[1:])
			return true

		case "list":
			// List available models
			listAvailableModels(deploymentService)
			return true

		case "history":
			// Show deployment history
			showDeploymentHistory(deploymentService)
			return true

		case "switch":
			// Switch to a different model
			if len(args) >= 2 {
				switchToModel(deploymentService, args[1])
			} else {
				fmt.Println("Usage: :deployment switch <model_id>")
			}
			return true

		case "info":
			// Show model information
			if len(args) >= 2 {
				showModelInfo(deploymentService, args[1])
			} else {
				fmt.Println("Usage: :deployment info <model_id>")
			}
			return true

		case "optimize":
			// Optimize a model
			if len(args) >= 2 {
				optimizeModel(deploymentService, args[1], args[2:])
			} else {
				fmt.Println("Usage: :deployment optimize <model_id> [options]")
			}
			return true

		case "help":
			// Show help
			showDeploymentHelp()
			return true

		default:
			fmt.Printf("Unknown deployment command: %s\n", cmd)
			fmt.Println("Type :deployment help for a list of available commands")
			return true
		}
	}

	return true
}

// showDeploymentStatus displays the current status of model deployment
func showDeploymentStatus(deploymentService *ModelDeploymentService) {
	fmt.Println("Model Deployment Status")
	fmt.Println("======================")

	// Get current deployed model
	currentModel := deploymentService.GetCurrentDeployedModel()
	if currentModel == "" {
		fmt.Println("No model currently deployed")
		fmt.Println("Use ':deployment deploy' to deploy a model")
		return
	}

	// Get model info
	modelInfo, err := os.Stat(currentModel)
	if err != nil {
		fmt.Printf("Error getting model info: %v\n", err)
		return
	}

	// Display model info
	fmt.Println("Current Deployed Model:")
	fmt.Printf("  Path: %s\n", currentModel)
	fmt.Printf("  Size: %.2f MB\n", float64(modelInfo.Size())/(1024*1024))
	fmt.Printf("  Last Modified: %s\n", modelInfo.ModTime().Format(time.RFC1123))

	// Get model metadata if available
	metadata, err := deploymentService.getModelMetadata(currentModel)
	if err == nil && metadata != nil {
		fmt.Printf("  Deployed On: %s\n", metadata.DeploymentTime.Format(time.RFC1123))
		fmt.Printf("  Validation Score: %.4f\n", metadata.ValidationScore)
		fmt.Printf("  Model Format: %s\n", metadata.ModelFormat)

		if metadata.Optimized {
			fmt.Printf("  Optimized: Yes (%.2f%% size reduction)\n",
				100.0*(1.0-float64(metadata.OptimizedSize)/float64(metadata.ModelSize)))
		} else {
			fmt.Println("  Optimized: No")
		}

		fmt.Printf("  Quantized: %v\n", metadata.Quantized)
	}

	// Get inference manager status
	inferenceManager := GetInferenceManager()
	if inferenceManager != nil {
		fmt.Println("\nInference Status:")
		stats := inferenceManager.GetInferenceStats()
		fmt.Printf("  Learning System: %s\n", getBoolStatus(stats["learning_enabled"].(bool)))
		fmt.Printf("  Using Custom Model: %s\n", getBoolStatus(stats["custom_model_enabled"].(bool)))
		fmt.Printf("  Local Inference: %s\n", getBoolStatus(stats["local_inference_enabled"].(bool)))
	}

	// Show available models count
	models, _ := deploymentService.ListAvailableModels()
	fmt.Printf("\nAvailable Models: %d\n", len(models))
	fmt.Println("Use ':deployment list' to see all available models")
}

// deployModel deploys a model with specified options
func deployModel(deploymentService *ModelDeploymentService, args []string) {
	// Default deployment configuration
	config := ModelDeploymentConfig{
		ModelPath:         "",
		ModelFormat:       "",
		TargetPath:        "",
		Optimize:          true,
		Quantize:          false,
		ValidateModel:     true,
		BackupExisting:    true,
		CreateSymlink:     true,
		OptimizationLevel: 1,
	}

	// Parse model path or ID
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		modelPathOrId := args[0]

		// Check if it's a numeric ID
		if isNumeric(modelPathOrId) {
			// Get model by ID
			models, err := deploymentService.ListAvailableModels()
			if err != nil {
				fmt.Printf("Error listing models: %v\n", err)
				return
			}

			id := parseId(modelPathOrId)
			if id > 0 && id <= len(models) {
				config.ModelPath = models[id-1].Path
			} else {
				fmt.Printf("Invalid model ID: %s\n", modelPathOrId)
				fmt.Printf("Valid IDs are 1-%d\n", len(models))
				return
			}
		} else if modelPathOrId == "latest" {
			// Get latest model
			models, err := deploymentService.ListAvailableModels()
			if err != nil {
				fmt.Printf("Error listing models: %v\n", err)
				return
			}

			if len(models) > 0 {
				config.ModelPath = models[0].Path
			} else {
				fmt.Println("No models available")
				return
			}
		} else {
			// Assume it's a path
			config.ModelPath = modelPathOrId
		}
	}

	// Parse options
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			continue
		}

		option := args[i]
		switch option {
		case "--no-optimize":
			config.Optimize = false
		case "--quantize":
			config.Quantize = true
		case "--no-validate":
			config.ValidateModel = false
		case "--no-backup":
			config.BackupExisting = false
		case "--no-symlink":
			config.CreateSymlink = false
		case "--optimization-level":
			if i+1 < len(args) {
				level := parseId(args[i+1])
				if level >= 0 && level <= 3 {
					config.OptimizationLevel = level
				}
				i++ // Skip the next argument
			}
		case "--format":
			if i+1 < len(args) {
				format := args[i+1]
				switch format {
				case "onnx":
					config.ModelFormat = ModelFormatONNX
				case "pytorch":
					config.ModelFormat = ModelFormatPyTorch
				case "binary":
					config.ModelFormat = ModelFormatBinary
				}
				i++ // Skip the next argument
			}
		case "--target":
			if i+1 < len(args) {
				config.TargetPath = args[i+1]
				i++ // Skip the next argument
			}
		}
	}

	// If no model path specified, use latest trained model
	if config.ModelPath == "" {
		// Find the latest trained model
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			return
		}

		modelsDir := filepath.Join(homeDir, ".config", "delta", "memory", "models")
		files, err := os.ReadDir(modelsDir)
		if err != nil {
			fmt.Printf("Error reading models directory: %v\n", err)
			return
		}

		// Find most recent model file
		var newest os.FileInfo
		var newestPath string
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".onnx") || strings.HasSuffix(file.Name(), ".bin") {
				info, err := file.Info()
				if err != nil {
					continue
				}

				if newest == nil || info.ModTime().After(newest.ModTime()) {
					newest = info
					newestPath = filepath.Join(modelsDir, file.Name())
				}
			}
		}

		if newestPath != "" {
			config.ModelPath = newestPath
		} else {
			fmt.Println("No trained models found")
			fmt.Println("Train a model first with ':memory train start'")
			return
		}
	}

	// Display deployment plan
	fmt.Println("Model Deployment Plan")
	fmt.Println("====================")
	fmt.Printf("Model: %s\n", config.ModelPath)
	fmt.Printf("Format: %s\n", config.ModelFormat)
	fmt.Printf("Optimization: %v (level %d)\n", config.Optimize, config.OptimizationLevel)
	fmt.Printf("Quantization: %v\n", config.Quantize)
	fmt.Printf("Validation: %v\n", config.ValidateModel)
	fmt.Printf("Backup existing: %v\n", config.BackupExisting)
	fmt.Printf("Create symlink: %v\n", config.CreateSymlink)

	// Confirm deployment
	fmt.Print("\nProceed with deployment? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Deployment cancelled")
		return
	}

	// Run deployment
	fmt.Println("\nDeploying model...")
	result, err := deploymentService.DeployModel(config)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		return
	}

	// Display result
	fmt.Println("\nDeployment Result")
	fmt.Println("================")
	fmt.Printf("Success: %v\n", result.Success)
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
		return
	}

	fmt.Printf("Source: %s\n", result.SourcePath)
	fmt.Printf("Deployed To: %s\n", result.DeployedPath)
	fmt.Printf("Format: %s\n", result.ModelFormat)
	fmt.Printf("Size: %.2f MB\n", float64(result.ModelSize)/(1024*1024))

	if result.Optimized {
		fmt.Printf("Optimized: Yes (%.2f%% size reduction)\n",
			100.0*(1.0-float64(result.OptimizedSize)/float64(result.ModelSize)))
	} else {
		fmt.Println("Optimized: No")
	}

	fmt.Printf("Quantized: %v\n", result.Quantized)
	fmt.Printf("Validation Score: %.4f\n", result.ValidationScore)

	if result.BackupPath != "" {
		fmt.Printf("Backup Created: %s\n", result.BackupPath)
	}

	if result.SymlinkPath != "" {
		fmt.Printf("Symlink Created: %s\n", result.SymlinkPath)
	}

	fmt.Println("\nModel has been successfully deployed and is now active")
}

// listAvailableModels lists all available models
func listAvailableModels(deploymentService *ModelDeploymentService) {
	// Get all models
	models, err := deploymentService.ListAvailableModels()
	if err != nil {
		fmt.Printf("Error listing models: %v\n", err)
		return
	}

	if len(models) == 0 {
		fmt.Println("No models available")
		fmt.Println("Train a model first with ':memory train start'")
		return
	}

	fmt.Println("Available Models")
	fmt.Println("===============")

	for i, model := range models {
		fmt.Printf("%d. %s\n", i+1, filepath.Base(model.Path))
		fmt.Printf("   Path: %s\n", model.Path)
		fmt.Printf("   Format: %s\n", model.Format)
		fmt.Printf("   Size: %.2f MB\n", float64(model.Size)/(1024*1024))
		fmt.Printf("   Modified: %s\n", model.ModTime.Format(time.RFC1123))

		if model.ValidationScore > 0 {
			fmt.Printf("   Validation Score: %.4f\n", model.ValidationScore)
		}

		status := ""
		if model.IsDeployed {
			status += "✓ Currently Deployed"
		}
		if model.IsLatest {
			if status != "" {
				status += ", "
			}
			status += "✓ Latest Model"
		}
		if status != "" {
			fmt.Printf("   Status: %s\n", status)
		}

		if i < len(models)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\nUse ':deployment deploy <id>' to deploy a model")
	fmt.Println("Use ':deployment info <id>' for detailed model information")
}

// showDeploymentHistory displays the deployment history
func showDeploymentHistory(deploymentService *ModelDeploymentService) {
	// Get deployment history
	deployments, err := deploymentService.GetDeploymentHistory()
	if err != nil {
		fmt.Printf("Error getting deployment history: %v\n", err)
		return
	}

	if len(deployments) == 0 {
		fmt.Println("No deployment history available")
		fmt.Println("Deploy a model first with ':deployment deploy'")
		return
	}

	fmt.Println("Deployment History")
	fmt.Println("=================")

	for i, deployment := range deployments {
		fmt.Printf("%d. Deployment on %s\n", i+1, deployment.DeploymentTime.Format(time.RFC1123))
		fmt.Printf("   Model: %s\n", filepath.Base(deployment.SourcePath))
		fmt.Printf("   Deployed To: %s\n", filepath.Base(deployment.DeployedPath))
		fmt.Printf("   Format: %s\n", deployment.ModelFormat)
		fmt.Printf("   Size: %.2f MB\n", float64(deployment.ModelSize)/(1024*1024))

		if deployment.Optimized {
			fmt.Printf("   Optimized: Yes (%.2f%% size reduction)\n",
				100.0*(1.0-float64(deployment.OptimizedSize)/float64(deployment.ModelSize)))
		} else {
			fmt.Println("   Optimized: No")
		}

		fmt.Printf("   Quantized: %v\n", deployment.Quantized)
		fmt.Printf("   Validation Score: %.4f\n", deployment.ValidationScore)
		fmt.Printf("   Success: %v\n", deployment.Success)

		if !deployment.Success {
			fmt.Printf("   Error: %s\n", deployment.Error)
		}

		if i < len(deployments)-1 {
			fmt.Println()
		}
	}
}

// switchToModel switches to a different model
func switchToModel(deploymentService *ModelDeploymentService, modelIdOrPath string) {
	var modelPath string

	// Check if it's a numeric ID
	if isNumeric(modelIdOrPath) {
		// Get model by ID
		models, err := deploymentService.ListAvailableModels()
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		id := parseId(modelIdOrPath)
		if id > 0 && id <= len(models) {
			modelPath = models[id-1].Path
		} else {
			fmt.Printf("Invalid model ID: %s\n", modelIdOrPath)
			fmt.Printf("Valid IDs are 1-%d\n", len(models))
			return
		}
	} else if modelIdOrPath == "latest" {
		// Get latest model
		models, err := deploymentService.ListAvailableModels()
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		if len(models) > 0 {
			modelPath = models[0].Path
		} else {
			fmt.Println("No models available")
			return
		}
	} else {
		// Assume it's a path
		modelPath = modelIdOrPath
	}

	// Verify model exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Printf("Model not found: %s\n", modelPath)
		return
	}

	// Switch to model
	err := deploymentService.SwitchToModel(modelPath)
	if err != nil {
		fmt.Printf("Error switching to model: %v\n", err)
		return
	}

	fmt.Printf("Successfully switched to model: %s\n", modelPath)
}

// showModelInfo displays detailed information about a model
func showModelInfo(deploymentService *ModelDeploymentService, modelIdOrPath string) {
	var modelPath string

	// Check if it's a numeric ID
	if isNumeric(modelIdOrPath) {
		// Get model by ID
		models, err := deploymentService.ListAvailableModels()
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		id := parseId(modelIdOrPath)
		if id > 0 && id <= len(models) {
			modelPath = models[id-1].Path
		} else {
			fmt.Printf("Invalid model ID: %s\n", modelIdOrPath)
			fmt.Printf("Valid IDs are 1-%d\n", len(models))
			return
		}
	} else if modelIdOrPath == "latest" {
		// Get latest model
		models, err := deploymentService.ListAvailableModels()
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		if len(models) > 0 {
			modelPath = models[0].Path
		} else {
			fmt.Println("No models available")
			return
		}
	} else {
		// Assume it's a path
		modelPath = modelIdOrPath
	}

	// Verify model exists
	modelInfo, err := os.Stat(modelPath)
	if os.IsNotExist(err) {
		fmt.Printf("Model not found: %s\n", modelPath)
		return
	}

	fmt.Println("Model Information")
	fmt.Println("=================")
	fmt.Printf("Path: %s\n", modelPath)
	fmt.Printf("Size: %.2f MB\n", float64(modelInfo.Size())/(1024*1024))
	fmt.Printf("Modified: %s\n", modelInfo.ModTime().Format(time.RFC1123))

	// Determine model format
	ext := filepath.Ext(modelPath)
	var format string
	switch ext {
	case ".onnx":
		format = "ONNX"
	case ".pt", ".pth":
		format = "PyTorch"
	case ".bin":
		format = "Binary"
	default:
		format = "Unknown"
	}
	fmt.Printf("Format: %s\n", format)

	// Check if this is the current deployed model
	currentModel := deploymentService.GetCurrentDeployedModel()
	if modelPath == currentModel {
		fmt.Println("Status: ✓ Currently Deployed")
	}

	// Get model metadata if available
	metadata, _ := deploymentService.getModelMetadata(modelPath)
	if metadata != nil {
		fmt.Printf("\nDeployment Information:\n")
		fmt.Printf("  Deployed On: %s\n", metadata.DeploymentTime.Format(time.RFC1123))
		fmt.Printf("  Validation Score: %.4f\n", metadata.ValidationScore)
		fmt.Printf("  Deployed Path: %s\n", metadata.DeployedPath)

		if metadata.Optimized {
			fmt.Printf("  Optimized: Yes (%.2f%% size reduction)\n",
				100.0*(1.0-float64(metadata.OptimizedSize)/float64(metadata.ModelSize)))
		} else {
			fmt.Println("  Optimized: No")
		}

		fmt.Printf("  Quantized: %v\n", metadata.Quantized)

		if metadata.BackupPath != "" {
			fmt.Printf("  Backup Path: %s\n", metadata.BackupPath)
		}

		if metadata.SymlinkPath != "" {
			fmt.Printf("  Symlink Path: %s\n", metadata.SymlinkPath)
		}
	}
}

// optimizeModel optimizes a model
func optimizeModel(deploymentService *ModelDeploymentService, modelIdOrPath string, options []string) {
	var modelPath string

	// Check if it's a numeric ID
	if isNumeric(modelIdOrPath) {
		// Get model by ID
		models, err := deploymentService.ListAvailableModels()
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		id := parseId(modelIdOrPath)
		if id > 0 && id <= len(models) {
			modelPath = models[id-1].Path
		} else {
			fmt.Printf("Invalid model ID: %s\n", modelIdOrPath)
			fmt.Printf("Valid IDs are 1-%d\n", len(models))
			return
		}
	} else if modelIdOrPath == "latest" {
		// Get latest model
		models, err := deploymentService.ListAvailableModels()
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		if len(models) > 0 {
			modelPath = models[0].Path
		} else {
			fmt.Println("No models available")
			return
		}
	} else {
		// Assume it's a path
		modelPath = modelIdOrPath
	}

	// Verify model exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Printf("Model not found: %s\n", modelPath)
		return
	}

	// Create deployment config
	config := ModelDeploymentConfig{
		ModelPath:         modelPath,
		Optimize:          true,
		Quantize:          false,
		ValidateModel:     true,
		BackupExisting:    true,
		CreateSymlink:     true,
		OptimizationLevel: 2, // Higher level for explicit optimization
	}

	// Parse options
	for i := 0; i < len(options); i++ {
		option := options[i]
		switch option {
		case "--quantize":
			config.Quantize = true
		case "--no-validate":
			config.ValidateModel = false
		case "--no-backup":
			config.BackupExisting = false
		case "--no-symlink":
			config.CreateSymlink = false
		case "--level":
			if i+1 < len(options) {
				level := parseId(options[i+1])
				if level >= 0 && level <= 3 {
					config.OptimizationLevel = level
				}
				i++ // Skip the next argument
			}
		}
	}

	// Set output path for optimized model
	modelName := filepath.Base(modelPath)
	timestamp := time.Now().Format("20060102_150405")
	optimizedName := fmt.Sprintf("%s_optimized_%s%s",
		strings.TrimSuffix(modelName, filepath.Ext(modelName)),
		timestamp,
		filepath.Ext(modelName))
	config.TargetPath = filepath.Join(deploymentService.deploymentDir, optimizedName)

	// Display optimization plan
	fmt.Println("Model Optimization Plan")
	fmt.Println("======================")
	fmt.Printf("Model: %s\n", config.ModelPath)
	fmt.Printf("Optimization Level: %d\n", config.OptimizationLevel)
	fmt.Printf("Quantization: %v\n", config.Quantize)
	fmt.Printf("Validation: %v\n", config.ValidateModel)
	fmt.Printf("Output: %s\n", config.TargetPath)

	// Confirm optimization
	fmt.Print("\nProceed with optimization? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Optimization cancelled")
		return
	}

	// Run deployment (which includes optimization)
	fmt.Println("\nOptimizing model...")
	result, err := deploymentService.DeployModel(config)
	if err != nil {
		fmt.Printf("Optimization failed: %v\n", err)
		return
	}

	// Display result
	fmt.Println("\nOptimization Result")
	fmt.Println("==================")
	fmt.Printf("Success: %v\n", result.Success)
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
		return
	}

	fmt.Printf("Source: %s\n", result.SourcePath)
	fmt.Printf("Optimized Model: %s\n", result.DeployedPath)
	fmt.Printf("Original Size: %.2f MB\n", float64(result.ModelSize)/(1024*1024))
	fmt.Printf("Optimized Size: %.2f MB\n", float64(result.OptimizedSize)/(1024*1024))
	fmt.Printf("Size Reduction: %.2f%%\n",
		100.0*(1.0-float64(result.OptimizedSize)/float64(result.ModelSize)))

	if result.Quantized {
		fmt.Println("Quantization: Applied")
	}

	fmt.Printf("Validation Score: %.4f\n", result.ValidationScore)

	fmt.Println("\nModel has been successfully optimized")
	fmt.Println("Use ':deployment switch <id>' to switch to this model")
}

// showDeploymentHelp displays help for deployment commands
func showDeploymentHelp() {
	fmt.Println("Deployment Commands")
	fmt.Println("==================")
	fmt.Println("  :deployment               - Show deployment status")
	fmt.Println("  :deployment deploy [id|path] [options] - Deploy a model")
	fmt.Println("  :deployment list          - List available models")
	fmt.Println("  :deployment history       - Show deployment history")
	fmt.Println("  :deployment switch <id>   - Switch to a different model")
	fmt.Println("  :deployment info <id>     - Show model information")
	fmt.Println("  :deployment optimize <id> [options] - Optimize a model")
	fmt.Println("  :deployment help          - Show this help message")

	fmt.Println("\nDeploy Options:")
	fmt.Println("  --no-optimize        - Skip optimization")
	fmt.Println("  --quantize           - Apply quantization")
	fmt.Println("  --no-validate        - Skip validation")
	fmt.Println("  --no-backup          - Don't backup existing model")
	fmt.Println("  --no-symlink         - Don't create symlink to latest model")
	fmt.Println("  --optimization-level <n> - Set optimization level (0-3)")
	fmt.Println("  --format <format>    - Specify model format (onnx, pytorch, binary)")
	fmt.Println("  --target <path>      - Specify target path for deployed model")

	fmt.Println("\nModel IDs:")
	fmt.Println("  IDs are assigned by order in the list command, with 1 being the newest")
	fmt.Println("  You can also use 'latest' to refer to the most recent model")
}

// Helper functions

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// parseId parses a string to an integer ID
func parseId(s string) int {
	var id int
	fmt.Sscanf(s, "%d", &id)
	return id
}

// getBoolStatus returns a string representation of a boolean status
func getBoolStatus(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}
