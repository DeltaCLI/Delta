package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ModelEvaluationMetric defines a specific metric for model evaluation
type ModelEvaluationMetric string

const (
	// MetricAccuracy measures overall prediction accuracy
	MetricAccuracy ModelEvaluationMetric = "accuracy"
	// MetricPrecision measures precision (true positives / predicted positives)
	MetricPrecision ModelEvaluationMetric = "precision"
	// MetricRecall measures recall (true positives / actual positives)
	MetricRecall ModelEvaluationMetric = "recall"
	// MetricF1Score measures F1 score (harmonic mean of precision and recall)
	MetricF1Score ModelEvaluationMetric = "f1_score"
	// MetricPerplexity measures perplexity (how well model predicts data)
	MetricPerplexity ModelEvaluationMetric = "perplexity"
	// MetricConfusionMatrix outputs a confusion matrix
	MetricConfusionMatrix ModelEvaluationMetric = "confusion_matrix"
)

// ModelEvaluationConfig defines configuration for model evaluation
type ModelEvaluationConfig struct {
	ModelPath      string                  // Path to the model to evaluate
	TestDataPath   string                  // Path to test data
	ModelType      string                  // Type of model (e.g., "onnx", "pytorch")
	Metrics        []ModelEvaluationMetric // Metrics to compute
	OutputDir      string                  // Directory for evaluation results
	BatchSize      int                     // Batch size for evaluation
	ClassThreshold float64                 // Threshold for positive class prediction
}

// ModelEvaluationResult contains the results of model evaluation
type ModelEvaluationResult struct {
	ModelPath      string                 // Path to the evaluated model
	ModelType      string                 // Type of model
	TestDataPath   string                 // Path to test data
	Timestamp      time.Time              // When evaluation was performed
	Metrics        map[string]float64     // Computed metrics
	ConfusionMatrix [][]int               // Confusion matrix (if computed)
	ExampleResults  []ExampleEvaluation   // Per-example evaluation results
	ErrorAnalysis   map[string]int        // Error analysis
}

// ExampleEvaluation contains evaluation for a single example
type ExampleEvaluation struct {
	Command         string  // Input command
	ActualPrediction string  // Actual model prediction
	ExpectedPrediction string // Expected prediction
	Correct         bool    // Whether prediction was correct
	Confidence      float64 // Model confidence in prediction
}

// ModelEvaluator manages evaluation of AI models
type ModelEvaluator struct {
	inferenceManager *InferenceManager
	trainingService  *TrainingDataService
	outputDir        string
}

// NewModelEvaluator creates a new model evaluator
func NewModelEvaluator() (*ModelEvaluator, error) {
	inferenceManager := GetInferenceManager()
	if inferenceManager == nil {
		return nil, fmt.Errorf("inference manager not available")
	}

	trainingService := GetTrainingDataService()
	if trainingService == nil {
		return nil, fmt.Errorf("training data service not available")
	}

	// Set up output directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	outputDir := filepath.Join(homeDir, ".config", "delta", "memory", "evaluations")
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	return &ModelEvaluator{
		inferenceManager: inferenceManager,
		trainingService:  trainingService,
		outputDir:        outputDir,
	}, nil
}

// EvaluateModel evaluates a model with the given configuration
func (e *ModelEvaluator) EvaluateModel(config ModelEvaluationConfig) (*ModelEvaluationResult, error) {
	// Validate config
	if config.ModelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	// Check if model exists
	if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model not found: %s", config.ModelPath)
	}

	// Determine test data path if not specified
	if config.TestDataPath == "" {
		// Use latest training data
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		
		testDataDir := filepath.Join(homeDir, ".config", "delta", "memory", "training_data")
		files, err := os.ReadDir(testDataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read training data directory: %v", err)
		}

		// Find most recent validation data file
		var newestValidation os.FileInfo
		var newestValidationPath string
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "val_data_") {
				info, err := file.Info()
				if err != nil {
					continue
				}
				
				if newestValidation == nil || info.ModTime().After(newestValidation.ModTime()) {
					newestValidation = info
					newestValidationPath = filepath.Join(testDataDir, file.Name())
				}
			}
		}
		
		if newestValidationPath != "" {
			config.TestDataPath = newestValidationPath
		} else {
			// If no validation data, look for training data
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "train_data_") {
					info, err := file.Info()
					if err != nil {
						continue
					}
					
					if newestValidation == nil || info.ModTime().After(newestValidation.ModTime()) {
						newestValidation = info
						newestValidationPath = filepath.Join(testDataDir, file.Name())
					}
				}
			}
			
			if newestValidationPath != "" {
				config.TestDataPath = newestValidationPath
			} else {
				return nil, fmt.Errorf("no test data found")
			}
		}
	}

	// Check if test data exists
	if _, err := os.Stat(config.TestDataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("test data not found: %s", config.TestDataPath)
	}

	// Set default output directory if not specified
	if config.OutputDir == "" {
		config.OutputDir = e.outputDir
	}

	// Set default metrics if not specified
	if len(config.Metrics) == 0 {
		config.Metrics = []ModelEvaluationMetric{
			MetricAccuracy,
			MetricPrecision,
			MetricRecall,
			MetricF1Score,
			MetricConfusionMatrix,
		}
	}

	// Set default batch size if not specified
	if config.BatchSize <= 0 {
		config.BatchSize = 16
	}

	// Set default class threshold if not specified
	if config.ClassThreshold <= 0 {
		config.ClassThreshold = 0.5
	}

	// Load test data
	testData, err := e.loadTestData(config.TestDataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load test data: %v", err)
	}

	// Run predictions on test data
	evaluations, err := e.evaluateExamples(config, testData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate examples: %v", err)
	}

	// Compute metrics
	result := &ModelEvaluationResult{
		ModelPath:    config.ModelPath,
		ModelType:    config.ModelType,
		TestDataPath: config.TestDataPath,
		Timestamp:    time.Now(),
		Metrics:      make(map[string]float64),
		ExampleResults: evaluations,
	}

	// For each requested metric, compute and add to results
	for _, metric := range config.Metrics {
		switch metric {
		case MetricAccuracy:
			result.Metrics["accuracy"] = e.computeAccuracy(evaluations)
		case MetricPrecision:
			result.Metrics["precision"] = e.computePrecision(evaluations)
		case MetricRecall:
			result.Metrics["recall"] = e.computeRecall(evaluations)
		case MetricF1Score:
			precision := e.computePrecision(evaluations)
			recall := e.computeRecall(evaluations)
			result.Metrics["f1_score"] = e.computeF1Score(precision, recall)
		case MetricPerplexity:
			result.Metrics["perplexity"] = e.computePerplexity(evaluations)
		case MetricConfusionMatrix:
			result.ConfusionMatrix = e.computeConfusionMatrix(evaluations)
		}
	}

	// Perform error analysis
	result.ErrorAnalysis = e.analyzeErrors(evaluations)

	// Save evaluation results
	err = e.saveEvaluationResults(result, config.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to save evaluation results: %v", err)
	}

	return result, nil
}

// loadTestData loads test data from a file
func (e *ModelEvaluator) loadTestData(path string) ([]TrainingExtendedExample, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data: %v", err)
	}

	// Parse the JSON data
	var examples []TrainingExtendedExample
	err = json.Unmarshal(data, &examples)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test data: %v", err)
	}

	return examples, nil
}

// evaluateExamples runs model predictions on test examples
func (e *ModelEvaluator) evaluateExamples(config ModelEvaluationConfig, 
	examples []TrainingExtendedExample) ([]ExampleEvaluation, error) {
	
	// In a real implementation, we'd load the model and run inference
	// Since we don't have the actual model inference code yet, we'll simulate it
	
	// For now, simulate predictions with a simple heuristic
	// In the full implementation, this would use the actual model
	results := make([]ExampleEvaluation, 0, len(examples))
	
	// Group examples for batch processing
	batches := make([][]TrainingExtendedExample, 0)
	for i := 0; i < len(examples); i += config.BatchSize {
		end := i + config.BatchSize
		if end > len(examples) {
			end = len(examples)
		}
		batches = append(batches, examples[i:end])
	}
	
	// Process each batch
	for _, batch := range batches {
		// Simulate batch prediction
		for _, example := range batch {
			// In the actual implementation, we'd run model inference here
			// For now, simulate with a simplistic approach
			
			// Simulate model prediction
			var actualPrediction string
			var confidence float64
			
			// Simple simulation - in real implementation we'd use the model
			if example.Label > 0 {
				// For positive examples, 80% chance of correct prediction
				if random() < 0.8 {
					actualPrediction = example.Prediction
					confidence = 0.7 + random()*0.25
				} else {
					actualPrediction = simulateIncorrectPrediction(example.Prediction)
					confidence = 0.5 + random()*0.2
				}
			} else if example.Label < 0 {
				// For negative examples, 70% chance of correct prediction
				if random() < 0.7 {
					actualPrediction = simulateIncorrectPrediction(example.Prediction)
					confidence = 0.6 + random()*0.3
				} else {
					actualPrediction = example.Prediction
					confidence = 0.5 + random()*0.15
				}
			} else {
				// For neutral examples, 60% chance either way
				if random() < 0.6 {
					actualPrediction = example.Prediction
					confidence = 0.55 + random()*0.2
				} else {
					actualPrediction = simulateIncorrectPrediction(example.Prediction)
					confidence = 0.5 + random()*0.15
				}
			}
			
			// Check if prediction matches expected
			correct := actualPrediction == example.Prediction
			
			// Add to results
			results = append(results, ExampleEvaluation{
				Command:          example.Command,
				ActualPrediction: actualPrediction,
				ExpectedPrediction: example.Prediction,
				Correct:          correct,
				Confidence:       confidence,
			})
		}
	}
	
	return results, nil
}

// computeAccuracy calculates prediction accuracy
func (e *ModelEvaluator) computeAccuracy(evaluations []ExampleEvaluation) float64 {
	if len(evaluations) == 0 {
		return 0.0
	}
	
	correct := 0
	for _, eval := range evaluations {
		if eval.Correct {
			correct++
		}
	}
	
	return float64(correct) / float64(len(evaluations))
}

// computePrecision calculates precision
func (e *ModelEvaluator) computePrecision(evaluations []ExampleEvaluation) float64 {
	truePositives := 0
	falsePositives := 0
	
	for _, eval := range evaluations {
		// Simplistic approach - in real implementation would use labels
		if eval.Correct && strings.Contains(eval.ActualPrediction, "helpful") {
			truePositives++
		} else if !eval.Correct && strings.Contains(eval.ActualPrediction, "helpful") {
			falsePositives++
		}
	}
	
	if truePositives+falsePositives == 0 {
		return 0.0
	}
	
	return float64(truePositives) / float64(truePositives+falsePositives)
}

// computeRecall calculates recall
func (e *ModelEvaluator) computeRecall(evaluations []ExampleEvaluation) float64 {
	truePositives := 0
	falseNegatives := 0
	
	for _, eval := range evaluations {
		// Simplistic approach - in real implementation would use labels
		if eval.Correct && strings.Contains(eval.ActualPrediction, "helpful") {
			truePositives++
		} else if !eval.Correct && strings.Contains(eval.ExpectedPrediction, "helpful") {
			falseNegatives++
		}
	}
	
	if truePositives+falseNegatives == 0 {
		return 0.0
	}
	
	return float64(truePositives) / float64(truePositives+falseNegatives)
}

// computeF1Score calculates F1 score
func (e *ModelEvaluator) computeF1Score(precision, recall float64) float64 {
	if precision+recall == 0 {
		return 0.0
	}
	
	return 2 * (precision * recall) / (precision + recall)
}

// computePerplexity calculates perplexity
func (e *ModelEvaluator) computePerplexity(evaluations []ExampleEvaluation) float64 {
	// Simplified perplexity calculation
	// In a real implementation, we'd use the model's actual probabilities
	
	// For each prediction, use confidence as a proxy for probability
	logProb := 0.0
	count := 0
	
	for _, eval := range evaluations {
		prob := eval.Confidence
		if prob < 0.01 {
			prob = 0.01 // Avoid log(0)
		}
		logProb += math.Log(prob)
		count++
	}
	
	if count == 0 {
		return 0.0
	}
	
	// Perplexity = exp(-1/N * sum(log(p)))
	return math.Exp(-logProb / float64(count))
}

// computeConfusionMatrix calculates confusion matrix
func (e *ModelEvaluator) computeConfusionMatrix(evaluations []ExampleEvaluation) [][]int {
	// 2x2 confusion matrix: predicted vs actual
	// [0][0]: true negatives, [0][1]: false negatives
	// [1][0]: false positives, [1][1]: true positives
	matrix := [][]int{
		{0, 0},
		{0, 0},
	}
	
	for _, eval := range evaluations {
		if eval.Correct {
			if strings.Contains(eval.ActualPrediction, "helpful") {
				matrix[1][1]++ // True positive
			} else {
				matrix[0][0]++ // True negative
			}
		} else {
			if strings.Contains(eval.ActualPrediction, "helpful") {
				matrix[1][0]++ // False positive
			} else {
				matrix[0][1]++ // False negative
			}
		}
	}
	
	return matrix
}

// analyzeErrors analyzes common error types
func (e *ModelEvaluator) analyzeErrors(evaluations []ExampleEvaluation) map[string]int {
	errors := make(map[string]int)
	
	for _, eval := range evaluations {
		if !eval.Correct {
			// Analyze the nature of the error
			
			// Check for certain error patterns
			if len(eval.ActualPrediction) < len(eval.ExpectedPrediction)/2 {
				errors["too_short"]++
			} else if len(eval.ActualPrediction) > len(eval.ExpectedPrediction)*2 {
				errors["too_long"]++
			} else if strings.Contains(eval.ActualPrediction, "git") && 
				!strings.Contains(eval.ExpectedPrediction, "git") {
				errors["wrong_tool"]++
			} else if !strings.Contains(eval.ActualPrediction, "git") && 
				strings.Contains(eval.ExpectedPrediction, "git") {
				errors["missed_tool"]++
			} else if eval.Confidence < 0.6 {
				errors["low_confidence"]++
			} else {
				errors["other"]++
			}
		}
	}
	
	return errors
}

// saveEvaluationResults saves evaluation results to a file
func (e *ModelEvaluator) saveEvaluationResults(result *ModelEvaluationResult, outputDir string) error {
	// Create output directory if it doesn't exist
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	
	// Create a filename with timestamp
	timestamp := result.Timestamp.Format("20060102_150405")
	modelName := filepath.Base(result.ModelPath)
	filename := fmt.Sprintf("%s_eval_%s.json", modelName, timestamp)
	outputPath := filepath.Join(outputDir, filename)
	
	// Marshal to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal evaluation results: %v", err)
	}
	
	// Write to file
	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluation results: %v", err)
	}
	
	// Also save a summary file for quick reference
	summaryPath := filepath.Join(outputDir, fmt.Sprintf("%s_summary_%s.txt", modelName, timestamp))
	summary := e.generateSummary(result)
	
	err = os.WriteFile(summaryPath, []byte(summary), 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluation summary: %v", err)
	}
	
	return nil
}

// generateSummary generates a human-readable summary of evaluation results
func (e *ModelEvaluator) generateSummary(result *ModelEvaluationResult) string {
	var sb strings.Builder
	
	sb.WriteString("Model Evaluation Summary\n")
	sb.WriteString("=======================\n\n")
	
	sb.WriteString(fmt.Sprintf("Model: %s\n", result.ModelPath))
	sb.WriteString(fmt.Sprintf("Type: %s\n", result.ModelType))
	sb.WriteString(fmt.Sprintf("Test Data: %s\n", result.TestDataPath))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n\n", result.Timestamp.Format(time.RFC1123)))
	
	sb.WriteString("Metrics:\n")
	
	// Sort metrics for consistent output
	metricNames := make([]string, 0, len(result.Metrics))
	for name := range result.Metrics {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)
	
	for _, name := range metricNames {
		value := result.Metrics[name]
		sb.WriteString(fmt.Sprintf("  %s: %.4f\n", name, value))
	}
	
	// Add confusion matrix if available
	if result.ConfusionMatrix != nil && len(result.ConfusionMatrix) >= 2 {
		sb.WriteString("\nConfusion Matrix:\n")
		sb.WriteString("  TN: " + fmt.Sprint(result.ConfusionMatrix[0][0]))
		sb.WriteString(" | FN: " + fmt.Sprint(result.ConfusionMatrix[0][1]) + "\n")
		sb.WriteString("  FP: " + fmt.Sprint(result.ConfusionMatrix[1][0]))
		sb.WriteString(" | TP: " + fmt.Sprint(result.ConfusionMatrix[1][1]) + "\n")
	}
	
	// Add error analysis if available
	if result.ErrorAnalysis != nil && len(result.ErrorAnalysis) > 0 {
		sb.WriteString("\nError Analysis:\n")
		
		errorTypes := make([]string, 0, len(result.ErrorAnalysis))
		for errorType := range result.ErrorAnalysis {
			errorTypes = append(errorTypes, errorType)
		}
		sort.Strings(errorTypes)
		
		for _, errorType := range errorTypes {
			count := result.ErrorAnalysis[errorType]
			sb.WriteString(fmt.Sprintf("  %s: %d\n", errorType, count))
		}
	}
	
	// Add a sample of incorrect predictions
	incorrectCount := 0
	for _, eval := range result.ExampleResults {
		if !eval.Correct {
			incorrectCount++
		}
	}
	
	sb.WriteString(fmt.Sprintf("\nExamples Evaluated: %d\n", len(result.ExampleResults)))
	sb.WriteString(fmt.Sprintf("Correct: %d (%.1f%%)\n", len(result.ExampleResults)-incorrectCount, 
		100.0*float64(len(result.ExampleResults)-incorrectCount)/float64(len(result.ExampleResults))))
	sb.WriteString(fmt.Sprintf("Incorrect: %d (%.1f%%)\n", incorrectCount, 
		100.0*float64(incorrectCount)/float64(len(result.ExampleResults))))
	
	// Include a few example errors
	if incorrectCount > 0 {
		sb.WriteString("\nSample Errors:\n")
		
		// Find a few interesting examples
		errCount := 0
		for _, eval := range result.ExampleResults {
			if !eval.Correct && errCount < 5 {
				sb.WriteString(fmt.Sprintf("\nCommand: %s\n", eval.Command))
				sb.WriteString(fmt.Sprintf("Expected: %s\n", eval.ExpectedPrediction))
				sb.WriteString(fmt.Sprintf("Actual:   %s\n", eval.ActualPrediction))
				sb.WriteString(fmt.Sprintf("Confidence: %.2f\n", eval.Confidence))
				errCount++
			}
		}
	}
	
	return sb.String()
}

// ListEvaluations lists all evaluation results
func (e *ModelEvaluator) ListEvaluations() ([]string, error) {
	// Read the output directory
	files, err := os.ReadDir(e.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluations directory: %v", err)
	}
	
	// Filter for evaluation files
	evaluations := make([]string, 0)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") && strings.Contains(file.Name(), "_eval_") {
			evaluations = append(evaluations, filepath.Join(e.outputDir, file.Name()))
		}
	}
	
	return evaluations, nil
}

// CompareModels compares multiple model evaluation results
func (e *ModelEvaluator) CompareModels(evalPaths []string) (string, error) {
	if len(evalPaths) == 0 {
		return "", fmt.Errorf("no evaluations to compare")
	}
	
	// Load evaluation results
	results := make([]*ModelEvaluationResult, 0, len(evalPaths))
	for _, path := range evalPaths {
		// Read the file
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read evaluation results: %v", err)
		}
		
		// Parse the JSON data
		var result ModelEvaluationResult
		err = json.Unmarshal(data, &result)
		if err != nil {
			return "", fmt.Errorf("failed to parse evaluation results: %v", err)
		}
		
		results = append(results, &result)
	}
	
	// Generate comparison report
	var sb strings.Builder
	
	sb.WriteString("Model Comparison Report\n")
	sb.WriteString("======================\n\n")
	
	// Table header
	sb.WriteString("| Metric      |")
	for i, result := range results {
		modelName := filepath.Base(result.ModelPath)
		sb.WriteString(fmt.Sprintf(" Model %d (%s) |", i+1, modelName))
	}
	sb.WriteString("\n")
	
	sb.WriteString("|-------------|")
	for range results {
		sb.WriteString("----------------|")
	}
	sb.WriteString("\n")
	
	// Common metrics
	commonMetrics := []string{"accuracy", "precision", "recall", "f1_score"}
	
	for _, metric := range commonMetrics {
		sb.WriteString(fmt.Sprintf("| %-11s |", metric))
		
		for _, result := range results {
			if value, ok := result.Metrics[metric]; ok {
				sb.WriteString(fmt.Sprintf(" %-14.4f |", value))
			} else {
				sb.WriteString(" N/A            |")
			}
		}
		sb.WriteString("\n")
	}
	
	// Test data info
	sb.WriteString("\nTest Data Information:\n")
	for i, result := range results {
		modelName := filepath.Base(result.ModelPath)
		testData := filepath.Base(result.TestDataPath)
		timestamp := result.Timestamp.Format("2006-01-02 15:04:05")
		
		sb.WriteString(fmt.Sprintf("Model %d (%s):\n", i+1, modelName))
		sb.WriteString(fmt.Sprintf("  Test Data: %s\n", testData))
		sb.WriteString(fmt.Sprintf("  Evaluated: %s\n", timestamp))
		
		// Count examples
		exampleCount := len(result.ExampleResults)
		correctCount := 0
		for _, eval := range result.ExampleResults {
			if eval.Correct {
				correctCount++
			}
		}
		
		sb.WriteString(fmt.Sprintf("  Examples: %d (%.1f%% correct)\n", 
			exampleCount, 100.0*float64(correctCount)/float64(exampleCount)))
	}
	
	return sb.String(), nil
}

// Helper functions

// random generates a random number between 0 and 1
func random() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// simulateIncorrectPrediction generates an incorrect prediction
func simulateIncorrectPrediction(correct string) string {
	// Simplistic approach - in real implementation would use model
	
	alternatives := []string{
		"This command is used for managing git repositories",
		"This looks like a Docker command for container management",
		"This command is related to system file operations",
		"This appears to be a build or compilation command",
		"This command is used for network operations",
		"You're working with database operations here",
		"This is related to text processing and manipulation",
	}
	
	// Pick a random alternative that's different from the correct one
	for {
		index := int(random() * float64(len(alternatives)))
		if index < len(alternatives) && alternatives[index] != correct {
			return alternatives[index]
		}
	}
}

// Global ModelEvaluator instance
var globalModelEvaluator *ModelEvaluator

// GetModelEvaluator returns the global ModelEvaluator instance
func GetModelEvaluator() *ModelEvaluator {
	if globalModelEvaluator == nil {
		var err error
		globalModelEvaluator, err = NewModelEvaluator()
		if err != nil {
			fmt.Printf("Error initializing model evaluator: %v\n", err)
			return nil
		}
	}
	return globalModelEvaluator
}