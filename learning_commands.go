package main

import (
	"fmt"
	"strconv"
)

// HandleLearningCommand processes learning-related commands
func HandleLearningCommand(args []string) bool {
	if len(args) == 0 {
		showLearningStatus()
		return true
	}

	switch args[0] {
	case "status":
		showLearningStatus()
		return true

	case "enable":
		enableLearning(true)
		return true

	case "disable":
		enableLearning(false)
		return true

	case "feedback":
		if len(args) > 1 {
			handleFeedbackCommand(args[1:])
		} else {
			collectInteractiveFeedback()
		}
		return true

	case "train":
		if len(args) > 1 {
			handleTrainCommand(args[1:])
		} else {
			showTrainHelp()
		}
		return true

	case "patterns":
		showLearnedPatterns(args[1:])
		return true

	case "process":
		processLearningData()
		return true

	case "stats":
		showLearningStats()
		return true

	case "config":
		if len(args) > 1 {
			configureLearning(args[1:])
		} else {
			showLearningConfig()
		}
		return true

	case "help":
		showLearningHelp()
		return true

	default:
		fmt.Printf("Unknown learning command: %s\n", args[0])
		fmt.Println("Type :learn help for available commands")
		return true
	}
}

// showLearningStatus displays the current learning system status
func showLearningStatus() {
	fmt.Println("Learning System Status")
	fmt.Println("=====================")

	// Get learning engine status
	le := GetLearningEngine()
	if le == nil {
		fmt.Println("❌ Learning Engine: Not initialized")
	} else {
		fmt.Println("✅ Learning Engine: Active")
		patterns := len(le.patterns)
		sequences := len(le.sequences)
		fmt.Printf("   Learned Patterns: %d\n", patterns)
		fmt.Printf("   Command Sequences: %d\n", sequences)
		fmt.Printf("   Last Processed: %s\n", le.lastProcessed.Format("2006-01-02 15:04:05"))
	}

	// Get feedback collector status
	fc := GetFeedbackCollector()
	if fc == nil {
		fmt.Println("❌ Feedback Collector: Not initialized")
	} else {
		if fc.IsEnabled() {
			fmt.Println("✅ Feedback Collector: Enabled")
		} else {
			fmt.Println("⚠️  Feedback Collector: Disabled")
		}
		stats := fc.GetFeedbackStats()
		fmt.Printf("   Total Feedback: %d\n", stats["total_feedback"])
		fmt.Printf("   Recent Predictions: %d\n", stats["recent_predictions"])
	}

	// Get training pipeline status
	tp := GetTrainingPipeline()
	if tp == nil {
		fmt.Println("❌ Training Pipeline: Not initialized")
	} else {
		if tp.config.Enabled {
			fmt.Println("✅ Training Pipeline: Enabled")
		} else {
			fmt.Println("⚠️  Training Pipeline: Disabled")
		}
		fmt.Printf("   Auto-train Interval: %d hours\n", tp.config.AutoTrainInterval)
		fmt.Printf("   Min Examples Required: %d\n", tp.config.MinTrainingExamples)
		
		// Show recent training runs
		history := tp.GetTrainingHistory(1)
		if len(history) > 0 {
			run := history[0]
			fmt.Printf("   Last Training: %s (Status: %s)\n", 
				run.StartTime.Format("2006-01-02 15:04"), run.Status)
		}
	}

	// Get training data stats
	tds := GetTrainingDataService()
	if tds != nil {
		stats := tds.GetTrainingDataStats()
		fmt.Println("\nTraining Data:")
		fmt.Printf("   Total Examples: %d\n", stats["total_examples"])
		fmt.Printf("   Accumulated: %d\n", stats["accumulated_examples"])
		
		if isReady, ok := stats["is_training_ready"].(bool); ok && isReady {
			fmt.Println("   ✅ Ready for training")
		} else {
			threshold := stats["training_threshold"].(int)
			accumulated := stats["accumulated_examples"].(int)
			fmt.Printf("   ⏳ Need %d more examples\n", threshold-accumulated)
		}
	}
}

// enableLearning enables or disables the learning system
func enableLearning(enable bool) {
	// Enable/disable learning engine
	le := GetLearningEngine()
	if le != nil {
		le.isEnabled = enable
	}

	// Enable/disable feedback collector
	fc := GetFeedbackCollector()
	if fc != nil {
		fc.EnableFeedbackCollection(enable)
	}

	// Enable/disable training pipeline
	tp := GetTrainingPipeline()
	if tp != nil {
		tp.config.Enabled = enable
		tp.saveConfig()
	}

	if enable {
		fmt.Println("✅ Learning system enabled")
		fmt.Println("Delta will now learn from your command patterns and feedback.")
	} else {
		fmt.Println("⚠️  Learning system disabled")
		fmt.Println("Delta will not collect feedback or learn from commands.")
	}
}

// collectInteractiveFeedback starts interactive feedback collection
func collectInteractiveFeedback() {
	fc := GetFeedbackCollector()
	if fc == nil {
		fmt.Println("Feedback collector not initialized")
		return
	}

	fc.CollectInteractiveFeedback()
}

// handleFeedbackCommand handles feedback subcommands
func handleFeedbackCommand(args []string) {
	if len(args) == 0 {
		collectInteractiveFeedback()
		return
	}

	switch args[0] {
	case "stats":
		showFeedbackStats()
	case "enable":
		fc := GetFeedbackCollector()
		if fc != nil {
			fc.EnableFeedbackCollection(true)
			fmt.Println("✅ Feedback collection enabled")
		}
	case "disable":
		fc := GetFeedbackCollector()
		if fc != nil {
			fc.EnableFeedbackCollection(false)
			fmt.Println("⚠️  Feedback collection disabled")
		}
	default:
		fmt.Printf("Unknown feedback command: %s\n", args[0])
	}
}

// showFeedbackStats displays feedback statistics
func showFeedbackStats() {
	fc := GetFeedbackCollector()
	if fc == nil {
		fmt.Println("Feedback collector not initialized")
		return
	}

	stats := fc.GetFeedbackStats()
	fmt.Println("Feedback Statistics")
	fmt.Println("==================")
	fmt.Printf("Total Feedback: %d\n", stats["total_feedback"])
	
	if byType, ok := stats["by_type"].(map[string]int); ok {
		fmt.Println("\nFeedback by Type:")
		for feedbackType, count := range byType {
			fmt.Printf("  %s: %d\n", feedbackType, count)
		}
	}
	
	fmt.Printf("\nRecent Predictions Tracked: %d\n", stats["recent_predictions"])
}

// handleTrainCommand handles training subcommands
func handleTrainCommand(args []string) {
	if len(args) == 0 {
		showTrainHelp()
		return
	}

	switch args[0] {
	case "start":
		startTraining()
	case "status":
		showLearningTrainingStatus()
	case "history":
		showTrainingHistory()
	case "config":
		if len(args) > 1 {
			configureLearning(args[1:])
		} else {
			showLearningConfig()
		}
	default:
		fmt.Printf("Unknown train command: %s\n", args[0])
		showTrainHelp()
	}
}

// startTraining starts the training pipeline
func startTraining() {
	tp := GetTrainingPipeline()
	if tp == nil {
		fmt.Println("Training pipeline not initialized")
		return
	}

	fmt.Println("Starting training pipeline...")
	
	// Run training in background
	go func() {
		if err := tp.RunDailyTraining(); err != nil {
			fmt.Printf("Training failed: %v\n", err)
		}
	}()
	
	fmt.Println("Training started in background. Use ':learn train status' to check progress.")
}

// showLearningTrainingStatus shows current training status
func showLearningTrainingStatus() {
	tp := GetTrainingPipeline()
	if tp == nil {
		fmt.Println("Training pipeline not initialized")
		return
	}

	tp.mutex.RLock()
	isRunning := tp.isRunning
	tp.mutex.RUnlock()

	if isRunning {
		fmt.Println("🔄 Training is currently running...")
	} else {
		fmt.Println("✅ No training in progress")
	}

	// Show recent runs
	history := tp.GetTrainingHistory(3)
	if len(history) > 0 {
		fmt.Println("\nRecent Training Runs:")
		fmt.Println("====================")
		for _, run := range history {
			duration := "N/A"
			if !run.EndTime.IsZero() {
				duration = run.EndTime.Sub(run.StartTime).String()
			}
			
			statusIcon := "❓"
			switch run.Status {
			case "completed":
				statusIcon = "✅"
			case "failed":
				statusIcon = "❌"
			case "training", "evaluating", "deploying":
				statusIcon = "🔄"
			}
			
			fmt.Printf("%s %s - Status: %s, Examples: %d, Score: %.3f, Duration: %s\n",
				statusIcon, run.ID, run.Status, run.ExamplesUsed, run.ValidationScore, duration)
		}
	}
}

// showTrainingHistory shows training history
func showTrainingHistory() {
	tp := GetTrainingPipeline()
	if tp == nil {
		fmt.Println("Training pipeline not initialized")
		return
	}

	history := tp.GetTrainingHistory(10)
	if len(history) == 0 {
		fmt.Println("No training history found")
		return
	}

	fmt.Println("Training History")
	fmt.Println("================")
	
	for _, run := range history {
		fmt.Printf("\nRun ID: %s\n", run.ID)
		fmt.Printf("  Started: %s\n", run.StartTime.Format("2006-01-02 15:04:05"))
		if !run.EndTime.IsZero() {
			fmt.Printf("  Ended: %s\n", run.EndTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Duration: %v\n", run.EndTime.Sub(run.StartTime))
		}
		fmt.Printf("  Status: %s\n", run.Status)
		fmt.Printf("  Examples Used: %d\n", run.ExamplesUsed)
		fmt.Printf("  Validation Score: %.3f\n", run.ValidationScore)
		if run.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", run.ErrorMessage)
		}
	}
}

// showLearnedPatterns displays learned patterns
func showLearnedPatterns(args []string) {
	le := GetLearningEngine()
	if le == nil {
		fmt.Println("Learning engine not initialized")
		return
	}

	le.mutex.RLock()
	defer le.mutex.RUnlock()

	// Filter patterns by type if specified
	filterType := ""
	if len(args) > 0 {
		filterType = args[0]
	}

	fmt.Println("Learned Patterns")
	fmt.Println("================")

	count := 0
	for key, pattern := range le.patterns {
		// Apply filter
		if filterType != "" && string(pattern.Type) != filterType {
			continue
		}

		if count >= 20 { // Limit output
			fmt.Println("\n... and more. Use filters to see specific patterns.")
			break
		}

		fmt.Printf("\n%s:\n", key)
		fmt.Printf("  Type: %s\n", pattern.Type)
		fmt.Printf("  Pattern: %s\n", pattern.Pattern)
		fmt.Printf("  Frequency: %d\n", pattern.Frequency)
		fmt.Printf("  Success Rate: %.2f\n", pattern.SuccessRate)
		fmt.Printf("  Confidence: %.2f\n", pattern.Confidence)
		
		if len(pattern.Predictions) > 0 {
			fmt.Printf("  Predictions: %v\n", pattern.Predictions)
		}
		
		count++
	}

	if count == 0 {
		if filterType != "" {
			fmt.Printf("No patterns found for type: %s\n", filterType)
		} else {
			fmt.Println("No patterns learned yet")
		}
	}
}

// processLearningData manually triggers learning data processing
func processLearningData() {
	le := GetLearningEngine()
	if le == nil {
		fmt.Println("Learning engine not initialized")
		return
	}

	fmt.Println("Processing learning data...")
	
	if err := le.ProcessDailyData(); err != nil {
		fmt.Printf("Error processing data: %v\n", err)
	} else {
		fmt.Println("✅ Learning data processed successfully")
		fmt.Printf("Patterns: %d, Sequences: %d\n", len(le.patterns), len(le.sequences))
	}
}

// showLearningStats displays learning statistics
func showLearningStats() {
	fmt.Println("Learning System Statistics")
	fmt.Println("=========================")

	// Learning engine stats
	le := GetLearningEngine()
	if le != nil {
		le.mutex.RLock()
		patternCount := len(le.patterns)
		sequenceCount := len(le.sequences)
		
		// Count by pattern type
		typeCounts := make(map[PatternType]int)
		totalFreq := 0
		avgConfidence := 0.0
		
		for _, pattern := range le.patterns {
			typeCounts[pattern.Type]++
			totalFreq += pattern.Frequency
			avgConfidence += pattern.Confidence
		}
		
		le.mutex.RUnlock()

		fmt.Printf("Total Patterns: %d\n", patternCount)
		fmt.Printf("Total Sequences: %d\n", sequenceCount)
		
		if patternCount > 0 {
			fmt.Println("\nPatterns by Type:")
			for pType, count := range typeCounts {
				fmt.Printf("  %s: %d\n", pType, count)
			}
			
			fmt.Printf("\nAverage Pattern Frequency: %.1f\n", float64(totalFreq)/float64(patternCount))
			fmt.Printf("Average Confidence: %.2f\n", avgConfidence/float64(patternCount))
		}
	}

	// Feedback stats
	fc := GetFeedbackCollector()
	if fc != nil {
		stats := fc.GetFeedbackStats()
		fmt.Printf("\nTotal Feedback Collected: %d\n", stats["total_feedback"])
		
		if byType, ok := stats["by_type"].(map[string]int); ok {
			total := 0
			for _, count := range byType {
				total += count
			}
			
			if total > 0 {
				fmt.Println("Feedback Distribution:")
				for feedbackType, count := range byType {
					percentage := float64(count) * 100.0 / float64(total)
					fmt.Printf("  %s: %d (%.1f%%)\n", feedbackType, count, percentage)
				}
			}
		}
	}

	// Training stats
	tds := GetTrainingDataService()
	if tds != nil {
		stats := tds.GetTrainingDataStats()
		fmt.Printf("\nTraining Examples: %d\n", stats["total_examples"])
		fmt.Printf("  Positive: %d\n", stats["positive_examples"])
		fmt.Printf("  Negative: %d\n", stats["negative_examples"])
		fmt.Printf("  Neutral: %d\n", stats["neutral_examples"])
	}
}

// configureLearning handles learning configuration
func configureLearning(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: :learn config <setting> <value>")
		fmt.Println("Available settings: auto_train_interval, min_examples, use_gpu")
		return
	}

	setting := args[0]
	value := args[1]

	tp := GetTrainingPipeline()
	if tp == nil {
		fmt.Println("Training pipeline not initialized")
		return
	}

	switch setting {
	case "auto_train_interval":
		if hours, err := strconv.Atoi(value); err == nil && hours > 0 {
			tp.config.AutoTrainInterval = hours
			tp.saveConfig()
			fmt.Printf("✅ Auto-train interval set to %d hours\n", hours)
		} else {
			fmt.Println("Invalid value. Must be a positive number of hours.")
		}

	case "min_examples":
		if min, err := strconv.Atoi(value); err == nil && min > 0 {
			tp.config.MinTrainingExamples = min
			tp.saveConfig()
			fmt.Printf("✅ Minimum training examples set to %d\n", min)
		} else {
			fmt.Println("Invalid value. Must be a positive number.")
		}

	case "use_gpu":
		if value == "true" || value == "on" || value == "1" {
			tp.config.UseGPU = true
			tp.saveConfig()
			fmt.Println("✅ GPU training enabled")
		} else if value == "false" || value == "off" || value == "0" {
			tp.config.UseGPU = false
			tp.saveConfig()
			fmt.Println("✅ GPU training disabled")
		} else {
			fmt.Println("Invalid value. Use true/false.")
		}

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
	}
}

// showLearningConfig displays current learning configuration
func showLearningConfig() {
	fmt.Println("Learning Configuration")
	fmt.Println("=====================")

	tp := GetTrainingPipeline()
	if tp != nil {
		fmt.Println("\nTraining Pipeline:")
		fmt.Printf("  Enabled: %v\n", tp.config.Enabled)
		fmt.Printf("  Auto-train Interval: %d hours\n", tp.config.AutoTrainInterval)
		fmt.Printf("  Min Training Examples: %d\n", tp.config.MinTrainingExamples)
		fmt.Printf("  Max Training Time: %d minutes\n", tp.config.MaxTrainingTime)
		fmt.Printf("  Use GPU: %v\n", tp.config.UseGPU)
		fmt.Printf("  Batch Size: %d\n", tp.config.BatchSize)
		fmt.Printf("  Learning Rate: %f\n", tp.config.LearningRate)
		fmt.Printf("  Validation Split: %.2f\n", tp.config.ValidationSplit)
		fmt.Printf("  Early Stopping: %v\n", tp.config.EarlyStopping)
		fmt.Printf("  Model Type: %s\n", tp.config.ModelType)
	}

	im := GetInferenceManager()
	if im != nil {
		fmt.Println("\nInference Settings:")
		fmt.Printf("  Collect Feedback: %v\n", im.learningConfig.CollectFeedback)
		fmt.Printf("  Automatic Feedback: %v\n", im.learningConfig.AutomaticFeedback)
		fmt.Printf("  Feedback Threshold: %.2f\n", im.learningConfig.FeedbackThreshold)
		fmt.Printf("  Adaptation Rate: %.2f\n", im.learningConfig.AdaptationRate)
	}
}

// showTrainHelp displays help for training commands
func showTrainHelp() {
	fmt.Println("Training Commands")
	fmt.Println("================")
	fmt.Println("  :learn train start     - Start training pipeline")
	fmt.Println("  :learn train status    - Check training status")
	fmt.Println("  :learn train history   - View training history")
	fmt.Println("  :learn train config    - Configure training settings")
}

// showLearningHelp displays help for learning commands
func showLearningHelp() {
	fmt.Println("Learning System Commands")
	fmt.Println("=======================")
	fmt.Println("  :learn                     - Show learning system status")
	fmt.Println("  :learn status              - Show detailed status")
	fmt.Println("  :learn enable              - Enable learning system")
	fmt.Println("  :learn disable             - Disable learning system")
	fmt.Println("  :learn feedback [cmd]      - Provide feedback or start interactive mode")
	fmt.Println("  :learn train [cmd]         - Training pipeline commands")
	fmt.Println("  :learn patterns [type]     - Show learned patterns")
	fmt.Println("  :learn process             - Process learning data manually")
	fmt.Println("  :learn stats               - Show learning statistics")
	fmt.Println("  :learn config [key] [val]  - Configure learning settings")
	fmt.Println("  :learn help                - Show this help message")
	
	fmt.Println("\nPattern Types:")
	fmt.Println("  command    - Command-specific patterns")
	fmt.Println("  sequence   - Command sequence patterns")
	fmt.Println("  directory  - Directory-specific patterns")
	fmt.Println("  time       - Time-based patterns")
	fmt.Println("  error      - Error resolution patterns")
	
	fmt.Println("\nExamples:")
	fmt.Println("  :learn feedback            - Start interactive feedback mode")
	fmt.Println("  :learn patterns command    - Show command patterns")
	fmt.Println("  :learn train start         - Start training pipeline")
	fmt.Println("  :learn config use_gpu true - Enable GPU training")
}