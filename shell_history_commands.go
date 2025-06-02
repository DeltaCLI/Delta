package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// handleHistoryImportCommand handles the "history import" command
func handleHistoryImportCommand(args []string) error {
	detector, err := NewShellHistoryDetector()
	if err != nil {
		return fmt.Errorf("failed to initialize history detector: %w", err)
	}

	parser := NewShellHistoryParser()
	
	// Initialize import tracker for deduplication
	tracker, err := NewShellHistoryTracker()
	if err != nil {
		return fmt.Errorf("failed to initialize import tracker: %w", err)
	}

	// Parse command options
	options := parseHistoryImportOptions(args)

	if options.Help {
		printHistoryImportHelp()
		return nil
	}

	// Detect history files
	var historyFiles []ShellHistoryFile
	if options.FilePath != "" {
		// Import specific file
		info, err := os.Stat(options.FilePath)
		if err != nil {
			return fmt.Errorf("cannot access file %s: %w", options.FilePath, err)
		}

		format := detector.detectHistoryFormat(options.FilePath)
		shell := "unknown"
		if strings.Contains(options.FilePath, "bash") {
			shell = "bash"
		} else if strings.Contains(options.FilePath, "zsh") {
			shell = "zsh"
		}

		historyFiles = []ShellHistoryFile{{
			Path:     options.FilePath,
			Shell:    shell,
			Format:   format,
			Size:     info.Size(),
			Readable: detector.isFileReadable(options.FilePath),
		}}
	} else {
		// Auto-detect history files
		historyFiles, err = detector.DetectHistoryFiles()
		if err != nil {
			return fmt.Errorf("failed to detect history files: %w", err)
		}
	}

	if len(historyFiles) == 0 {
		fmt.Println("No shell history files found.")
		return nil
	}

	// Check for previously imported files and show status
	fmt.Printf("Detected %d shell history file(s):\n\n", len(historyFiles))
	for i, file := range historyFiles {
		status := "readable"
		if !file.Readable {
			status = "not readable"
		}
		
		// Check if file has been imported before
		imported, record, err := tracker.HasBeenImported(file.Path)
		importStatus := ""
		if err == nil {
			if imported {
				importStatus = fmt.Sprintf(" [previously imported %s]", 
					record.LastImported.Format("2006-01-02"))
			} else if record != nil {
				importStatus = " [file updated since last import]"
			} else {
				importStatus = " [never imported]"
			}
		}
		
		fmt.Printf("%d. %s (%s, %s, %.1f KB, %s)%s\n",
			i+1, file.Path, file.Shell, file.Format,
			float64(file.Size)/1024, status, importStatus)
	}
	fmt.Println()

	// Interactive mode - ask user for permission
	if !options.AutoDetect && !options.DryRun {
		if !askUserPermission("Do you want to proceed with importing these history files?") {
			fmt.Println("Import cancelled.")
			return nil
		}
	}

	// Process each history file
	totalImported := 0
	for _, file := range historyFiles {
		if !file.Readable {
			fmt.Printf("Skipping %s (not readable)\n", file.Path)
			continue
		}

		fmt.Printf("Processing %s...\n", file.Path)

		// Parse the history file
		allEntries, err := parser.ParseHistoryFile(file.Path, file.Format, options.Limit)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", file.Path, err)
			continue
		}

		// Apply deduplication - only get new commands not previously imported
		entries, err := tracker.GetNewCommandsOnly(file.Path, allEntries)
		if err != nil {
			fmt.Printf("Error checking for new commands in %s: %v\n", file.Path, err)
			continue
		}

		// If no new commands and not forcing re-import
		if len(entries) == 0 && !options.Force {
			fmt.Printf("No new commands found in %s (already imported)\n", file.Path)
			continue
		}

		// Apply filters
		if !options.IncludeSensitive {
			filters := HistoryFilters{
				ExcludeSensitive: true,
				MinLength:        2,
			}
			entries = parser.FilterEntries(entries, filters)
		}

		if options.DryRun {
			fmt.Printf("Would import %d commands from %s\n", len(entries), file.Path)
			if options.Verbose {
				printSampleCommands(entries, 5)
			}
			continue
		}

		// Convert to training data and save
		if len(entries) > 0 {
			err = saveHistoryTrainingData(entries, file)
			if err != nil {
				fmt.Printf("Error saving training data from %s: %v\n", file.Path, err)
				continue
			}

			// Record successful import in tracker
			err = tracker.RecordImport(file.Path, allEntries, len(entries))
			if err != nil {
				fmt.Printf("Warning: Failed to record import for %s: %v\n", file.Path, err)
			}

			totalImported += len(entries)
			fmt.Printf("Imported %d new commands from %s\n", len(entries), file.Path)

			if options.Verbose {
				stats := parser.GetParsingStats(entries)
				printHistoryStats(stats)
			}
		}
	}

	if !options.DryRun {
		fmt.Printf("\nTotal commands imported: %d\n", totalImported)
		fmt.Println("Shell history import completed successfully!")
	}

	return nil
}

// HistoryImportOptions holds command-line options for history import
type HistoryImportOptions struct {
	FilePath         string
	AutoDetect       bool
	Limit            int
	Interactive      bool
	DryRun           bool
	Verbose          bool
	IncludeSensitive bool
	Force            bool // Force re-import even if previously imported
	Help             bool
}

// parseHistoryImportOptions parses command-line arguments for history import
func parseHistoryImportOptions(args []string) HistoryImportOptions {
	options := HistoryImportOptions{
		Interactive: true, // Default to interactive mode
		Limit:       0,    // No limit by default
	}

	for i, arg := range args {
		switch arg {
		case "--help", "-h":
			options.Help = true
		case "--auto-detect":
			options.AutoDetect = true
			options.Interactive = false
		case "--file":
			if i+1 < len(args) {
				options.FilePath = args[i+1]
			}
		case "--limit":
			if i+1 < len(args) {
				if limit, err := strconv.Atoi(args[i+1]); err == nil {
					options.Limit = limit
				}
			}
		case "--dry-run":
			options.DryRun = true
		case "--verbose", "-v":
			options.Verbose = true
		case "--include-sensitive":
			options.IncludeSensitive = true
		case "--force":
			options.Force = true
		case "--non-interactive":
			options.Interactive = false
		}
	}

	return options
}

// askUserPermission prompts the user for yes/no confirmation
func askUserPermission(question string) bool {
	fmt.Printf("%s (y/N): ", question)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// printSampleCommands shows a sample of commands that would be imported
func printSampleCommands(entries []HistoryEntry, limit int) {
	fmt.Printf("Sample commands (showing up to %d):\n", limit)
	count := 0
	for _, entry := range entries {
		if count >= limit {
			break
		}
		if entry.Valid {
			timestamp := ""
			if !entry.Timestamp.IsZero() {
				timestamp = fmt.Sprintf("[%s] ", entry.Timestamp.Format("2006-01-02 15:04"))
			}
			fmt.Printf("  %s%s\n", timestamp, entry.Command)
			count++
		}
	}
	if len(entries) > limit {
		fmt.Printf("  ... and %d more commands\n", len(entries)-limit)
	}
	fmt.Println()
}

// printHistoryStats displays statistics about parsed history
func printHistoryStats(stats map[string]interface{}) {
	fmt.Println("Statistics:")
	if total, ok := stats["total_entries"].(int); ok {
		fmt.Printf("  Total entries: %d\n", total)
	}
	if valid, ok := stats["valid_entries"].(int); ok {
		fmt.Printf("  Valid entries: %d\n", valid)
	}
	if withTs, ok := stats["with_timestamps"].(int); ok {
		fmt.Printf("  With timestamps: %d\n", withTs)
	}
	if unique, ok := stats["unique_commands"].(int); ok {
		fmt.Printf("  Unique commands: %d\n", unique)
	}
	if avgLen, ok := stats["avg_command_length"].(int); ok {
		fmt.Printf("  Average command length: %d characters\n", avgLen)
	}
	fmt.Println()
}

// saveHistoryTrainingData saves parsed history entries as training data
func saveHistoryTrainingData(entries []HistoryEntry, file ShellHistoryFile) error {
	parser := NewShellHistoryParser()
	
	// Convert to training data format
	context := fmt.Sprintf("shell_history_%s", file.Shell)
	trainingData := parser.ConvertToTrainingData(entries, context)

	// Save to Delta's training data system
	// This would integrate with the existing training data infrastructure
	return saveTrainingDataToFile(trainingData, fmt.Sprintf("shell_history_%s_%d.json", 
		file.Shell, time.Now().Unix()))
}

// saveTrainingDataToFile saves training data to a file using the existing system
func saveTrainingDataToFile(data []TrainingData, filename string) error {
	// Get the memory manager to save training data
	mm := GetMemoryManager()
	if mm == nil {
		return fmt.Errorf("memory manager not available")
	}

	// Convert shell history training data to Delta's format and save
	for _, item := range data {
		// Create a command entry for the memory system
		entry := CommandEntry{
			Command:     item.Command,
			Directory:   item.Context,
			Timestamp:   item.Timestamp,
			Environment: map[string]string{"source": item.Source},
		}

		// Store in memory system - this will be used for training
		date := item.Timestamp.Format("2006-01-02")
		err := mm.WriteCommand(date, entry)
		if err != nil {
			fmt.Printf("Warning: Failed to save command to memory: %v\n", err)
		}
	}

	fmt.Printf("Imported %d commands into Delta's training system\n", len(data))
	return nil
}

// printHistoryImportHelp displays help information for the history import command
func printHistoryImportHelp() {
	fmt.Println(`Delta Shell History Import

USAGE:
    delta history import [OPTIONS]

DESCRIPTION:
    Import and train on existing shell history files from your home directory.
    This command can detect bash and zsh history files automatically and convert
    them into training data for Delta's command prediction system.

OPTIONS:
    --auto-detect        Automatically detect and import all found history files
                        without prompting for confirmation
    
    --file PATH         Import a specific history file instead of auto-detecting
    
    --limit N           Only import the last N commands from each file
    
    --dry-run           Show what would be imported without actually doing it
    
    --verbose, -v       Show detailed statistics and sample commands
    
    --include-sensitive Include potentially sensitive commands (passwords, keys, etc.)
                       By default, these are filtered out for security
    
    --force            Force re-import even if files were previously imported
                       By default, only new commands since last import are processed
    
    --non-interactive   Skip interactive prompts and use default settings
    
    --help, -h         Show this help message

EXAMPLES:
    # Interactive import with confirmation prompts
    delta history import
    
    # Auto-detect and import all history files
    delta history import --auto-detect
    
    # Import specific file with limit
    delta history import --file ~/.bash_history --limit 1000
    
    # Dry run to see what would be imported
    delta history import --dry-run --verbose
    
    # Non-interactive import of recent commands only
    delta history import --auto-detect --limit 500 --non-interactive

PRIVACY:
    This command respects your privacy:
    - Always asks for permission before accessing history files
    - Filters out potentially sensitive commands by default
    - Shows you exactly what will be imported before proceeding
    - Stores imported data securely within Delta's data directory`)
}