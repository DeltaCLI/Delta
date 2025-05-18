package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// KnowledgeType represents different types of extracted knowledge
type KnowledgeType string

const (
	KnowledgeCommandPattern KnowledgeType = "command_pattern"
	KnowledgeDirectoryFlow KnowledgeType = "directory_flow"
	KnowledgeToolUsage     KnowledgeType = "tool_usage"
	KnowledgeFileOperation KnowledgeType = "file_operation"
	KnowledgeEnvironment   KnowledgeType = "environment"
	KnowledgeWorkflow      KnowledgeType = "workflow"
)

// KnowledgeEntity represents an extracted piece of knowledge
type KnowledgeEntity struct {
	ID          string       `json:"id"`
	Type        KnowledgeType `json:"type"`
	Pattern     string       `json:"pattern"`
	Examples    []string     `json:"examples"`
	Confidence  float64      `json:"confidence"`
	LastUpdated time.Time    `json:"last_updated"`
	UsageCount  int          `json:"usage_count"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// KnowledgeBatch represents a batch of commands for knowledge extraction
type KnowledgeBatch struct {
	Commands  []CommandEntry `json:"commands"`
	BatchID   string         `json:"batch_id"`
	Timestamp time.Time      `json:"timestamp"`
}

// KnowledgeExtractorConfig contains configuration for knowledge extraction
type KnowledgeExtractorConfig struct {
	Enabled             bool     `json:"enabled"`
	StoragePath         string   `json:"storage_path"`
	MinConfidence       float64  `json:"min_confidence"`
	BatchSize           int      `json:"batch_size"`
	MaxEntities         int      `json:"max_entities"`
	ScanInterval        int      `json:"scan_interval"`         // in minutes
	PatternThreshold    int      `json:"pattern_threshold"`    
	ExtractEnvironment  bool     `json:"extract_environment"`
	ExtractWorkflows    bool     `json:"extract_workflows"`
	SensitivePatterns   []string `json:"sensitive_patterns"`
	IncludeCommands     []string `json:"include_commands"`
	ExcludeCommands     []string `json:"exclude_commands"`
	ContextSize         int      `json:"context_size"`
}

// KnowledgeExtractor handles extracting knowledge from command history
type KnowledgeExtractor struct {
	config        KnowledgeExtractorConfig
	configPath    string
	entities      map[string]KnowledgeEntity
	patterns      map[string]int
	mutex         sync.RWMutex
	isInitialized bool
	lastScan      time.Time
}

// NewKnowledgeExtractor creates a new knowledge extractor
func NewKnowledgeExtractor() (*KnowledgeExtractor, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/knowledge directory
	configDir := filepath.Join(homeDir, ".config", "delta", "knowledge")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge directory: %v", err)
	}

	configPath := filepath.Join(configDir, "knowledge_config.json")
	storagePath := filepath.Join(configDir, "entities")

	// Create default configuration
	extractor := &KnowledgeExtractor{
		config: KnowledgeExtractorConfig{
			Enabled:            false,
			StoragePath:        storagePath,
			MinConfidence:      0.6,
			BatchSize:          100,
			MaxEntities:        1000,
			ScanInterval:       60, // 1 hour
			PatternThreshold:   3,
			ExtractEnvironment: true,
			ExtractWorkflows:   true,
			SensitivePatterns:  []string{"password", "token", "key", "secret", "credential"},
			IncludeCommands:    []string{"git", "docker", "kubectl", "npm", "pip", "yarn", "make", "cd", "cp", "mv", "rm"},
			ExcludeCommands:    []string{"ls", "clear", "exit", "history"},
			ContextSize:        3,
		},
		configPath:    configPath,
		entities:      make(map[string]KnowledgeEntity),
		patterns:      make(map[string]int),
		mutex:         sync.RWMutex{},
		isInitialized: false,
		lastScan:      time.Time{},
	}

	// Try to load existing configuration
	err = extractor.loadConfig()
	if err != nil {
		// Save the default configuration if loading fails
		extractor.saveConfig()
	}

	return extractor, nil
}

// Initialize initializes the knowledge extractor
func (ke *KnowledgeExtractor) Initialize() error {
	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	// Create storage directory if it doesn't exist
	err := os.MkdirAll(ke.config.StoragePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create knowledge storage directory: %v", err)
	}

	// Load existing entities
	err = ke.loadEntities()
	if err != nil {
		fmt.Printf("Warning: Failed to load knowledge entities: %v\n", err)
		// Continue anyway with empty entities
	}

	ke.isInitialized = true
	return nil
}

// loadConfig loads the knowledge extractor configuration from disk
func (ke *KnowledgeExtractor) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(ke.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(ke.configPath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data
	return json.Unmarshal(data, &ke.config)
}

// saveConfig saves the knowledge extractor configuration to disk
func (ke *KnowledgeExtractor) saveConfig() error {
	// Marshal the config to JSON with indentation for readability
	data, err := json.MarshalIndent(ke.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(ke.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(ke.configPath, data, 0644)
}

// loadEntities loads knowledge entities from storage
func (ke *KnowledgeExtractor) loadEntities() error {
	// Check if storage directory exists
	_, err := os.Stat(ke.config.StoragePath)
	if os.IsNotExist(err) {
		return nil // No entities to load
	}

	// Get list of entity files
	files, err := os.ReadDir(ke.config.StoragePath)
	if err != nil {
		return err
	}

	entities := make(map[string]KnowledgeEntity)
	patterns := make(map[string]int)

	// Load each entity file
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Read the file
		data, err := os.ReadFile(filepath.Join(ke.config.StoragePath, file.Name()))
		if err != nil {
			fmt.Printf("Warning: Failed to read entity file %s: %v\n", file.Name(), err)
			continue
		}

		// Unmarshal the JSON data
		var entity KnowledgeEntity
		err = json.Unmarshal(data, &entity)
		if err != nil {
			fmt.Printf("Warning: Failed to parse entity file %s: %v\n", file.Name(), err)
			continue
		}

		// Skip entities with low confidence
		if entity.Confidence < ke.config.MinConfidence {
			continue
		}

		// Add to entities map
		entities[entity.ID] = entity
		patterns[entity.Pattern] = entity.UsageCount
	}

	// Update entities and patterns maps
	ke.entities = entities
	ke.patterns = patterns

	return nil
}

// saveEntity saves a knowledge entity to storage
func (ke *KnowledgeExtractor) saveEntity(entity KnowledgeEntity) error {
	// Marshal the entity to JSON with indentation for readability
	data, err := json.MarshalIndent(entity, "", "  ")
	if err != nil {
		return err
	}

	// Create storage directory if it doesn't exist
	if err = os.MkdirAll(ke.config.StoragePath, 0755); err != nil {
		return err
	}

	// Write to file
	filename := fmt.Sprintf("%s_%s.json", entity.Type, entity.ID)
	return os.WriteFile(filepath.Join(ke.config.StoragePath, filename), data, 0644)
}

// IsEnabled returns whether the knowledge extractor is enabled
func (ke *KnowledgeExtractor) IsEnabled() bool {
	ke.mutex.RLock()
	defer ke.mutex.RUnlock()
	return ke.config.Enabled && ke.isInitialized
}

// Enable enables the knowledge extractor
func (ke *KnowledgeExtractor) Enable() error {
	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	if !ke.isInitialized {
		return fmt.Errorf("knowledge extractor not initialized")
	}

	ke.config.Enabled = true
	return ke.saveConfig()
}

// Disable disables the knowledge extractor
func (ke *KnowledgeExtractor) Disable() error {
	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	ke.config.Enabled = false
	return ke.saveConfig()
}

// ProcessBatch processes a batch of commands for knowledge extraction
func (ke *KnowledgeExtractor) ProcessBatch(batch KnowledgeBatch) error {
	if !ke.IsEnabled() {
		return nil
	}

	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	// Extract command patterns
	if err := ke.extractCommandPatterns(batch.Commands); err != nil {
		return err
	}

	// Extract directory flows
	if err := ke.extractDirectoryFlows(batch.Commands); err != nil {
		return err
	}

	// Extract tool usage patterns
	if err := ke.extractToolUsage(batch.Commands); err != nil {
		return err
	}

	// Extract file operations
	if err := ke.extractFileOperations(batch.Commands); err != nil {
		return err
	}

	// Extract environment information
	if ke.config.ExtractEnvironment {
		if err := ke.extractEnvironmentInfo(batch.Commands); err != nil {
			return err
		}
	}

	// Extract workflows
	if ke.config.ExtractWorkflows {
		if err := ke.extractWorkflows(batch.Commands); err != nil {
			return err
		}
	}

	// Update last scan time
	ke.lastScan = time.Now()

	return nil
}

// extractCommandPatterns extracts command patterns from command history
func (ke *KnowledgeExtractor) extractCommandPatterns(commands []CommandEntry) error {
	// Skip if no commands
	if len(commands) == 0 {
		return nil
	}

	// Extract command patterns
	for _, cmd := range commands {
		// Skip if command is in exclude list
		if ke.isExcludedCommand(cmd.Command) {
			continue
		}

		// Skip if command contains sensitive information
		if ke.containsSensitiveInfo(cmd.Command) {
			continue
		}

		// Extract command pattern
		pattern := ke.extractCommandPattern(cmd.Command)
		if pattern == "" {
			continue
		}

		// Update pattern count
		if _, ok := ke.patterns[pattern]; ok {
			ke.patterns[pattern]++
		} else {
			ke.patterns[pattern] = 1
		}

		// Check if pattern meets threshold
		if ke.patterns[pattern] >= ke.config.PatternThreshold {
			// Check if entity already exists
			entityID := fmt.Sprintf("pattern_%x", hash(pattern))
			entity, exists := ke.entities[entityID]

			if exists {
				// Update existing entity
				entity.UsageCount++
				entity.LastUpdated = time.Now()
				
				// Add command as example if not already present
				found := false
				for _, example := range entity.Examples {
					if example == cmd.Command {
						found = true
						break
					}
				}
				if !found && len(entity.Examples) < 5 {
					entity.Examples = append(entity.Examples, cmd.Command)
				}
				
				// Update confidence
				entity.Confidence = minFloat(1.0, entity.Confidence+0.05)

				// Save updated entity
				ke.entities[entityID] = entity
				ke.saveEntity(entity)
			} else {
				// Create new entity
				newEntity := KnowledgeEntity{
					ID:          entityID,
					Type:        KnowledgeCommandPattern,
					Pattern:     pattern,
					Examples:    []string{cmd.Command},
					Confidence:  0.6,
					LastUpdated: time.Now(),
					UsageCount:  ke.patterns[pattern],
					Metadata: map[string]string{
						"directory": cmd.Directory,
					},
				}

				// Save new entity
				ke.entities[entityID] = newEntity
				ke.saveEntity(newEntity)
			}
		}
	}

	return nil
}

// extractDirectoryFlows extracts directory navigation patterns
func (ke *KnowledgeExtractor) extractDirectoryFlows(commands []CommandEntry) error {
	// Skip if not enough commands
	if len(commands) < 3 {
		return nil
	}

	// Track directory changes
	dirChanges := make(map[string][]string)
	currentDir := ""

	for _, cmd := range commands {
		if cmd.Directory == currentDir {
			continue
		}

		// Record directory change
		if currentDir != "" {
			dirChanges[currentDir] = append(dirChanges[currentDir], cmd.Directory)
		}
		currentDir = cmd.Directory
	}

	// Process directory flows
	for source, destinations := range dirChanges {
		// Skip if not enough destinations
		if len(destinations) < 2 {
			continue
		}

		// Create flow pattern
		flowPattern := fmt.Sprintf("%s -> [%s]", source, strings.Join(destinations, ", "))
		
		// Create entity ID
		entityID := fmt.Sprintf("flow_%x", hash(flowPattern))
		
		// Check if entity already exists
		entity, exists := ke.entities[entityID]
		
		if exists {
			// Update existing entity
			entity.UsageCount++
			entity.LastUpdated = time.Now()
			entity.Confidence = minFloat(1.0, entity.Confidence+0.05)
			
			// Save updated entity
			ke.entities[entityID] = entity
			ke.saveEntity(entity)
		} else {
			// Create new entity
			examples := make([]string, 0)
			for i := 0; i < len(commands)-1; i++ {
				if commands[i].Directory == source && containsString(destinations, commands[i+1].Directory) {
					cmdPair := fmt.Sprintf("%s (in %s) -> %s (in %s)", 
						commands[i].Command, source, 
						commands[i+1].Command, commands[i+1].Directory)
					examples = append(examples, cmdPair)
					if len(examples) >= 3 {
						break
					}
				}
			}
			
			newEntity := KnowledgeEntity{
				ID:          entityID,
				Type:        KnowledgeDirectoryFlow,
				Pattern:     flowPattern,
				Examples:    examples,
				Confidence:  0.7,
				LastUpdated: time.Now(),
				UsageCount:  1,
				Metadata: map[string]string{
					"source": source,
					"destinations": strings.Join(destinations, ","),
				},
			}
			
			// Save new entity
			ke.entities[entityID] = newEntity
			ke.saveEntity(newEntity)
		}
	}

	return nil
}

// extractToolUsage extracts tool usage patterns
func (ke *KnowledgeExtractor) extractToolUsage(commands []CommandEntry) error {
	// Skip if no commands
	if len(commands) == 0 {
		return nil
	}

	// Extract tool usage
	toolPatterns := make(map[string]int)
	
	for _, cmd := range commands {
		// Skip if command is excluded
		if ke.isExcludedCommand(cmd.Command) {
			continue
		}
		
		// Extract tool name
		tool := ke.extractToolName(cmd.Command)
		if tool == "" {
			continue
		}
		
		// Create tool pattern
		toolPattern := fmt.Sprintf("%s %s", tool, ke.extractToolArgs(cmd.Command, tool))
		
		// Update pattern count
		if _, ok := toolPatterns[toolPattern]; ok {
			toolPatterns[toolPattern]++
		} else {
			toolPatterns[toolPattern] = 1
		}
		
		// Check if pattern meets threshold
		if toolPatterns[toolPattern] >= ke.config.PatternThreshold {
			// Create entity ID
			entityID := fmt.Sprintf("tool_%x", hash(toolPattern))
			
			// Check if entity already exists
			entity, exists := ke.entities[entityID]
			
			if exists {
				// Update existing entity
				entity.UsageCount++
				entity.LastUpdated = time.Now()
				
				// Add command as example if not already present
				found := false
				for _, example := range entity.Examples {
					if example == cmd.Command {
						found = true
						break
					}
				}
				if !found && len(entity.Examples) < 5 {
					entity.Examples = append(entity.Examples, cmd.Command)
				}
				
				// Update confidence
				entity.Confidence = minFloat(1.0, entity.Confidence+0.05)

				// Save updated entity
				ke.entities[entityID] = entity
				ke.saveEntity(entity)
			} else {
				// Create new entity
				newEntity := KnowledgeEntity{
					ID:          entityID,
					Type:        KnowledgeToolUsage,
					Pattern:     toolPattern,
					Examples:    []string{cmd.Command},
					Confidence:  0.6,
					LastUpdated: time.Now(),
					UsageCount:  toolPatterns[toolPattern],
					Metadata: map[string]string{
						"tool": tool,
						"directory": cmd.Directory,
					},
				}

				// Save new entity
				ke.entities[entityID] = newEntity
				ke.saveEntity(newEntity)
			}
		}
	}

	return nil
}

// extractFileOperations extracts file operation patterns
func (ke *KnowledgeExtractor) extractFileOperations(commands []CommandEntry) error {
	// Skip if no commands
	if len(commands) == 0 {
		return nil
	}

	// File operation commands
	fileOps := []string{"cp", "mv", "rm", "mkdir", "touch", "cat", "grep", "find", "sed", "awk"}
	
	// Extract file operations
	for _, cmd := range commands {
		// Check if command is a file operation
		isFileOp := false
		for _, op := range fileOps {
			if strings.HasPrefix(cmd.Command, op+" ") {
				isFileOp = true
				break
			}
		}
		
		if !isFileOp {
			continue
		}
		
		// Skip if command contains sensitive information
		if ke.containsSensitiveInfo(cmd.Command) {
			continue
		}
		
		// Create file operation pattern
		parts := strings.Fields(cmd.Command)
		if len(parts) < 2 {
			continue
		}
		
		opType := parts[0]
		
		// Create pattern based on operation type
		var pattern string
		switch opType {
		case "cp", "mv":
			// Pattern: cp/mv [file type] [destination type]
			if len(parts) >= 3 {
				srcType := ke.getFileType(parts[1])
				dstType := ke.getFileType(parts[2])
				pattern = fmt.Sprintf("%s %s %s", opType, srcType, dstType)
			}
		case "rm":
			// Pattern: rm [file type]
			if len(parts) >= 2 {
				fileType := ke.getFileType(parts[1])
				pattern = fmt.Sprintf("%s %s", opType, fileType)
			}
		case "mkdir", "touch":
			// Pattern: mkdir/touch [file type]
			if len(parts) >= 2 {
				fileType := ke.getFileType(parts[1])
				pattern = fmt.Sprintf("%s %s", opType, fileType)
			}
		case "cat", "grep":
			// Pattern: cat/grep [file type]
			if len(parts) >= 2 {
				fileType := ke.getFileType(parts[len(parts)-1])
				pattern = fmt.Sprintf("%s %s", opType, fileType)
			}
		}
		
		if pattern == "" {
			continue
		}
		
		// Create entity ID
		entityID := fmt.Sprintf("fileop_%x", hash(pattern))
		
		// Check if entity already exists
		entity, exists := ke.entities[entityID]
		
		if exists {
			// Update existing entity
			entity.UsageCount++
			entity.LastUpdated = time.Now()
			
			// Add command as example if not already present
			found := false
			for _, example := range entity.Examples {
				if example == cmd.Command {
					found = true
					break
				}
			}
			if !found && len(entity.Examples) < 5 {
				entity.Examples = append(entity.Examples, cmd.Command)
			}
			
			// Update confidence
			entity.Confidence = minFloat(1.0, entity.Confidence+0.05)

			// Save updated entity
			ke.entities[entityID] = entity
			ke.saveEntity(entity)
		} else {
			// Create new entity
			newEntity := KnowledgeEntity{
				ID:          entityID,
				Type:        KnowledgeFileOperation,
				Pattern:     pattern,
				Examples:    []string{cmd.Command},
				Confidence:  0.7,
				LastUpdated: time.Now(),
				UsageCount:  1,
				Metadata: map[string]string{
					"operation": opType,
					"directory": cmd.Directory,
				},
			}

			// Save new entity
			ke.entities[entityID] = newEntity
			ke.saveEntity(newEntity)
		}
	}

	return nil
}

// extractEnvironmentInfo extracts environment information
func (ke *KnowledgeExtractor) extractEnvironmentInfo(commands []CommandEntry) error {
	// Skip if no commands
	if len(commands) == 0 {
		return nil
	}

	// Collect environment variables from commands
	envVars := make(map[string][]string)
	
	for _, cmd := range commands {
		// Skip if no environment variables
		if cmd.Environment == nil || len(cmd.Environment) == 0 {
			continue
		}
		
		// Extract environment variables
		for key, value := range cmd.Environment {
			if ke.containsSensitiveInfo(value) {
				continue
			}
			
			envVars[key] = append(envVars[key], value)
		}
	}
	
	// Process environment variables
	for key, values := range envVars {
		// Skip if not enough values
		if len(values) < 2 {
			continue
		}
		
		// Create environment pattern
		envPattern := fmt.Sprintf("%s=[%s]", key, strings.Join(uniqueStrings(values), ", "))
		
		// Create entity ID
		entityID := fmt.Sprintf("env_%x", hash(envPattern))
		
		// Check if entity already exists
		entity, exists := ke.entities[entityID]
		
		if exists {
			// Update existing entity
			entity.UsageCount++
			entity.LastUpdated = time.Now()
			entity.Confidence = minFloat(1.0, entity.Confidence+0.05)
			
			// Save updated entity
			ke.entities[entityID] = entity
			ke.saveEntity(entity)
		} else {
			// Create examples
			examples := make([]string, 0)
			for i := 0; i < min(3, len(values)); i++ {
				examples = append(examples, fmt.Sprintf("%s=%s", key, values[i]))
			}
			
			// Create new entity
			newEntity := KnowledgeEntity{
				ID:          entityID,
				Type:        KnowledgeEnvironment,
				Pattern:     envPattern,
				Examples:    examples,
				Confidence:  0.8,
				LastUpdated: time.Now(),
				UsageCount:  1,
				Metadata: map[string]string{
					"variable": key,
					"values": strings.Join(uniqueStrings(values), ","),
				},
			}
			
			// Save new entity
			ke.entities[entityID] = newEntity
			ke.saveEntity(newEntity)
		}
	}

	return nil
}

// extractWorkflows extracts command workflows
func (ke *KnowledgeExtractor) extractWorkflows(commands []CommandEntry) error {
	// Skip if not enough commands
	if len(commands) < ke.config.ContextSize {
		return nil
	}

	// Extract workflows
	for i := 0; i <= len(commands)-ke.config.ContextSize; i++ {
		// Create workflow from context window
		workflow := make([]string, 0, ke.config.ContextSize)
		
		for j := 0; j < ke.config.ContextSize; j++ {
			cmd := commands[i+j]
			
			// Skip if command contains sensitive information
			if ke.containsSensitiveInfo(cmd.Command) {
				continue
			}
			
			// Add command pattern
			pattern := ke.extractCommandPattern(cmd.Command)
			if pattern != "" {
				workflow = append(workflow, pattern)
			}
		}
		
		// Skip if workflow is empty or incomplete
		if len(workflow) < ke.config.ContextSize {
			continue
		}
		
		// Create workflow pattern
		workflowPattern := strings.Join(workflow, " -> ")
		
		// Create entity ID
		entityID := fmt.Sprintf("workflow_%x", hash(workflowPattern))
		
		// Check if entity already exists
		entity, exists := ke.entities[entityID]
		
		if exists {
			// Update existing entity
			entity.UsageCount++
			entity.LastUpdated = time.Now()
			entity.Confidence = minFloat(1.0, entity.Confidence+0.05)
			
			// Save updated entity
			ke.entities[entityID] = entity
			ke.saveEntity(entity)
		} else {
			// Create examples
			examples := make([]string, 0, ke.config.ContextSize)
			for j := 0; j < ke.config.ContextSize; j++ {
				examples = append(examples, commands[i+j].Command)
			}
			
			// Create new entity
			newEntity := KnowledgeEntity{
				ID:          entityID,
				Type:        KnowledgeWorkflow,
				Pattern:     workflowPattern,
				Examples:    examples,
				Confidence:  0.6,
				LastUpdated: time.Now(),
				UsageCount:  1,
				Metadata: map[string]string{
					"directory": commands[i].Directory,
					"steps": fmt.Sprintf("%d", ke.config.ContextSize),
				},
			}
			
			// Save new entity
			ke.entities[entityID] = newEntity
			ke.saveEntity(newEntity)
		}
	}

	return nil
}

// Query searches for knowledge entities matching a query
func (ke *KnowledgeExtractor) Query(query string, entityType KnowledgeType, limit int) ([]KnowledgeEntity, error) {
	if !ke.IsEnabled() {
		return nil, fmt.Errorf("knowledge extractor not enabled")
	}

	ke.mutex.RLock()
	defer ke.mutex.RUnlock()

	// Filter entities by type and query
	var results []KnowledgeEntity
	
	for _, entity := range ke.entities {
		// Filter by type if specified
		if entityType != "" && entity.Type != entityType {
			continue
		}
		
		// Filter by query
		if !strings.Contains(strings.ToLower(entity.Pattern), strings.ToLower(query)) {
			// Check examples
			matchFound := false
			for _, example := range entity.Examples {
				if strings.Contains(strings.ToLower(example), strings.ToLower(query)) {
					matchFound = true
					break
				}
			}
			
			if !matchFound {
				continue
			}
		}
		
		// Add to results
		results = append(results, entity)
	}
	
	// Sort results by confidence and usage count
	sortEntities(results)
	
	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetStats returns statistics about the knowledge extractor
func (ke *KnowledgeExtractor) GetStats() map[string]interface{} {
	ke.mutex.RLock()
	defer ke.mutex.RUnlock()

	// Count entities by type
	counts := make(map[KnowledgeType]int)
	for _, entity := range ke.entities {
		counts[entity.Type]++
	}

	// Calculate average confidence
	var totalConfidence float64
	for _, entity := range ke.entities {
		totalConfidence += entity.Confidence
	}
	avgConfidence := 0.0
	if len(ke.entities) > 0 {
		avgConfidence = totalConfidence / float64(len(ke.entities))
	}

	// Count entities with embeddings (check vector database)
	entityIDsInVectorDB := make(map[string]bool)
	vectorDB := GetVectorDBManager()
	itemsWithEmbeddings := 0
	
	if vectorDB != nil && vectorDB.IsEnabled() {
		// Check if we have items in the vector database
		if db := vectorDB.db; db != nil {
			rows, err := db.Query(`
				SELECT command_id FROM command_embeddings 
				WHERE command_id LIKE 'knowledge_%'
			`)
			if err == nil {
				defer rows.Close()
				
				for rows.Next() {
					var commandID string
					if err := rows.Scan(&commandID); err == nil {
						// Extract entity ID
						entityID := strings.TrimPrefix(commandID, "knowledge_")
						entityIDsInVectorDB[entityID] = true
					}
				}
				
				// Count entities with embeddings
				for id := range ke.entities {
					if entityIDsInVectorDB[id] {
						itemsWithEmbeddings++
					}
				}
			}
		}
	}

	// Count sources
	sourceCounts := make(map[string]int)
	for _, entity := range ke.entities {
		if dir, ok := entity.Metadata["directory"]; ok {
			sourceCounts[dir]++
		}
	}

	stats := map[string]interface{}{
		"enabled":               ke.config.Enabled,
		"initialized":           ke.isInitialized,
		"entity_count":          len(ke.entities),
		"items_with_embeddings": itemsWithEmbeddings,
		"total_items":           len(ke.entities),
		"pattern_count":         len(ke.patterns),
		"command_patterns":      counts[KnowledgeCommandPattern],
		"directory_flows":       counts[KnowledgeDirectoryFlow],
		"tool_usage":            counts[KnowledgeToolUsage],
		"file_operations":       counts[KnowledgeFileOperation],
		"environment_entities":  counts[KnowledgeEnvironment],
		"workflow_entities":     counts[KnowledgeWorkflow],
		"average_confidence":    avgConfidence,
		"last_scan":             ke.lastScan,
		"last_refresh":          ke.lastScan,
		"source_counts":         sourceCounts,
		"type_counts": map[string]interface{}{
			"command_pattern": counts[KnowledgeCommandPattern],
			"directory_flow":  counts[KnowledgeDirectoryFlow],
			"tool_usage":      counts[KnowledgeToolUsage],
			"file_operation":  counts[KnowledgeFileOperation],
			"environment":     counts[KnowledgeEnvironment],
			"workflow":        counts[KnowledgeWorkflow],
		},
		"environment_awareness": ke.config.ExtractEnvironment,
		"command_awareness":     true,
		"code_awareness":        false,
		"project_awareness":     true,
		"privacy_enabled":       len(ke.config.SensitivePatterns) > 0,
		"max_file_size_kb":      100,
		"max_scan_depth":        3,
		"max_extracted_items":   ke.config.MaxEntities,
		"refresh_interval_minutes": ke.config.ScanInterval,
		"config": map[string]interface{}{
			"min_confidence":      ke.config.MinConfidence,
			"batch_size":          ke.config.BatchSize,
			"max_entities":        ke.config.MaxEntities,
			"scan_interval":       ke.config.ScanInterval,
			"pattern_threshold":   ke.config.PatternThreshold,
			"extract_environment": ke.config.ExtractEnvironment,
			"extract_workflows":   ke.config.ExtractWorkflows,
			"context_size":        ke.config.ContextSize,
		},
	}
	
	// Add vector database information if available
	if vectorDB != nil && vectorDB.IsEnabled() {
		vectorDBStats := vectorDB.GetStats()
		stats["vector_db_enabled"] = true
		stats["vector_db_status"] = "enabled"
		
		if vectorCount, ok := vectorDBStats["vector_count"].(int); ok {
			stats["vector_db_count"] = vectorCount
		}
		
		if metric, ok := vectorDBStats["metric"].(string); ok {
			stats["vector_db_metric"] = metric
		}
		
		if hasVectorExt, ok := vectorDBStats["has_vector_extension"].(bool); ok {
			stats["vector_db_extension"] = hasVectorExt
		}
	} else {
		stats["vector_db_enabled"] = false
		stats["vector_db_status"] = "disabled"
	}
	
	return stats
}

// ExportEntities exports knowledge entities to a file
func (ke *KnowledgeExtractor) ExportEntities(filepath string) error {
	ke.mutex.RLock()
	defer ke.mutex.RUnlock()

	// Convert entities map to slice
	entities := make([]KnowledgeEntity, 0, len(ke.entities))
	for _, entity := range ke.entities {
		entities = append(entities, entity)
	}

	// Marshal entities to JSON
	data, err := json.MarshalIndent(entities, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filepath, data, 0644)
}

// ImportEntities imports knowledge entities from a file
func (ke *KnowledgeExtractor) ImportEntities(filepath string) error {
	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	// Unmarshal entities
	var entities []KnowledgeEntity
	err = json.Unmarshal(data, &entities)
	if err != nil {
		return err
	}

	// Update entities map
	for _, entity := range entities {
		// Skip if entity has low confidence
		if entity.Confidence < ke.config.MinConfidence {
			continue
		}

		// Update or add entity
		ke.entities[entity.ID] = entity
		
		// Save entity to storage
		ke.saveEntity(entity)
		
		// Update patterns map
		if entity.Type == KnowledgeCommandPattern {
			ke.patterns[entity.Pattern] = entity.UsageCount
		}
	}

	return nil
}

// UpdateConfig updates the knowledge extractor configuration
func (ke *KnowledgeExtractor) UpdateConfig(config KnowledgeExtractorConfig) error {
	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	ke.config = config
	return ke.saveConfig()
}

// ClearEntities removes all knowledge entities
func (ke *KnowledgeExtractor) ClearEntities() error {
	ke.mutex.Lock()
	defer ke.mutex.Unlock()

	// Remove entity files
	err := os.RemoveAll(ke.config.StoragePath)
	if err != nil {
		return err
	}

	// Recreate storage directory
	err = os.MkdirAll(ke.config.StoragePath, 0755)
	if err != nil {
		return err
	}

	// Clear entities and patterns maps
	ke.entities = make(map[string]KnowledgeEntity)
	ke.patterns = make(map[string]int)

	return nil
}

// Helper functions

// extractCommandPattern extracts a pattern from a command
func (ke *KnowledgeExtractor) extractCommandPattern(command string) string {
	// Split command into words
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	// Get command name
	cmdName := parts[0]

	// Check if command is in include list
	isIncluded := false
	for _, includedCmd := range ke.config.IncludeCommands {
		if cmdName == includedCmd {
			isIncluded = true
			break
		}
	}

	if !isIncluded {
		return ""
	}

	// Create pattern based on command type
	switch cmdName {
	case "git":
		if len(parts) > 1 {
			if parts[1] == "commit" && len(parts) > 2 {
				return "git commit [message]"
			} else if parts[1] == "push" && len(parts) > 2 {
				return "git push [remote] [branch]"
			} else if parts[1] == "pull" && len(parts) > 2 {
				return "git pull [remote] [branch]"
			} else if parts[1] == "checkout" && len(parts) > 2 {
				return "git checkout [branch]"
			} else {
				return fmt.Sprintf("git %s", parts[1])
			}
		}
	case "docker":
		if len(parts) > 1 {
			if parts[1] == "run" && len(parts) > 2 {
				return "docker run [image]"
			} else if parts[1] == "build" && len(parts) > 2 {
				return "docker build [context]"
			} else if parts[1] == "exec" && len(parts) > 2 {
				return "docker exec [container]"
			} else {
				return fmt.Sprintf("docker %s", parts[1])
			}
		}
	case "kubectl":
		if len(parts) > 1 {
			if parts[1] == "get" && len(parts) > 2 {
				return fmt.Sprintf("kubectl get %s", parts[2])
			} else if parts[1] == "apply" && len(parts) > 2 {
				return "kubectl apply [file]"
			} else if parts[1] == "delete" && len(parts) > 2 {
				return fmt.Sprintf("kubectl delete %s", parts[2])
			} else {
				return fmt.Sprintf("kubectl %s", parts[1])
			}
		}
	case "npm", "yarn":
		if len(parts) > 1 {
			return fmt.Sprintf("%s %s", cmdName, parts[1])
		}
	case "make":
		if len(parts) > 1 {
			return fmt.Sprintf("make %s", parts[1])
		}
	case "cd":
		if len(parts) > 1 {
			// Simplify path
			path := parts[1]
			if path == ".." {
				return "cd .."
			} else if strings.HasPrefix(path, "./") {
				return "cd [relative]"
			} else if strings.HasPrefix(path, "/") {
				return "cd [absolute]"
			} else if strings.HasPrefix(path, "~") {
				return "cd [home]"
			} else {
				return "cd [dir]"
			}
		}
	case "cp", "mv", "rm":
		if len(parts) > 2 {
			return fmt.Sprintf("%s [source] [dest]", cmdName)
		} else if len(parts) > 1 {
			return fmt.Sprintf("%s [path]", cmdName)
		}
	default:
		// Generic pattern
		if len(parts) > 1 {
			return fmt.Sprintf("%s [args]", cmdName)
		} else {
			return cmdName
		}
	}

	return ""
}

// extractToolName extracts the name of a tool from a command
func (ke *KnowledgeExtractor) extractToolName(command string) string {
	// Split command into words
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	// Return first word as tool name
	return parts[0]
}

// extractToolArgs extracts tool arguments pattern
func (ke *KnowledgeExtractor) extractToolArgs(command string, tool string) string {
	// Split command into words
	parts := strings.Fields(command)
	if len(parts) <= 1 {
		return ""
	}

	// Skip tool name
	args := parts[1:]

	// Extract arguments pattern based on tool
	switch tool {
	case "git":
		if len(args) > 0 {
			return args[0]
		}
	case "docker":
		if len(args) > 0 {
			return args[0]
		}
	case "kubectl":
		if len(args) > 0 {
			if args[0] == "get" && len(args) > 1 {
				return fmt.Sprintf("%s %s", args[0], args[1])
			} else {
				return args[0]
			}
		}
	case "npm", "yarn":
		if len(args) > 0 {
			return args[0]
		}
	default:
		// Generic args pattern
		if len(args) > 0 {
			return args[0]
		}
	}

	return ""
}

// getFileType extracts the type of a file from a path
func (ke *KnowledgeExtractor) getFileType(path string) string {
	// Check for flags
	if strings.HasPrefix(path, "-") {
		return "[flag]"
	}

	// Get file extension
	ext := filepath.Ext(path)
	if ext != "" {
		return fmt.Sprintf("[%s file]", ext)
	}

	// Check path type
	if strings.HasPrefix(path, "./") {
		return "[relative path]"
	} else if strings.HasPrefix(path, "/") {
		return "[absolute path]"
	} else if strings.HasPrefix(path, "~") {
		return "[home path]"
	} else if strings.HasPrefix(path, "$") {
		return "[variable path]"
	} else if strings.Contains(path, "*") {
		return "[glob pattern]"
	} else {
		return "[path]"
	}
}

// isExcludedCommand checks if a command is in the exclude list
func (ke *KnowledgeExtractor) isExcludedCommand(command string) bool {
	// Get command name
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return true
	}
	cmdName := parts[0]

	// Check if in exclude list
	for _, excludedCmd := range ke.config.ExcludeCommands {
		if cmdName == excludedCmd {
			return true
		}
	}

	return false
}

// containsSensitiveInfo checks if a string contains sensitive information
func (ke *KnowledgeExtractor) containsSensitiveInfo(str string) bool {
	// Check against sensitive patterns
	for _, pattern := range ke.config.SensitivePatterns {
		if strings.Contains(strings.ToLower(str), strings.ToLower(pattern)) {
			return true
		}
	}

	// Check for common sensitive patterns
	sensitiveRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)password\s*=\s*.+`),
		regexp.MustCompile(`(?i)secret\s*=\s*.+`),
		regexp.MustCompile(`(?i)token\s*=\s*.+`),
		regexp.MustCompile(`(?i)key\s*=\s*.+`),
		regexp.MustCompile(`(?i)api[_-]?key\s*=\s*.+`),
		regexp.MustCompile(`(?i)auth\s*=\s*.+`),
		regexp.MustCompile(`(?i)credential\s*=\s*.+`),
	}

	for _, regex := range sensitiveRegexes {
		if regex.MatchString(str) {
			return true
		}
	}

	return false
}

// sortEntities sorts knowledge entities by confidence and usage count
func sortEntities(entities []KnowledgeEntity) {
	sort.Slice(entities, func(i, j int) bool {
		// Sort by confidence first
		if entities[i].Confidence > entities[j].Confidence {
			return true
		}
		if entities[i].Confidence < entities[j].Confidence {
			return false
		}
		
		// Then by usage count
		if entities[i].UsageCount > entities[j].UsageCount {
			return true
		}
		if entities[i].UsageCount < entities[j].UsageCount {
			return false
		}
		
		// Then by last updated
		return entities[i].LastUpdated.After(entities[j].LastUpdated)
	})
}

// uniqueStrings returns a slice of unique strings
func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// containsString checks if a slice contains a string
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minFloat returns the minimum of two float64 values
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// hash returns a hash of a string
func hash(s string) uint32 {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return h
}

// GenerateEmbeddings generates embeddings for knowledge entities
func (ke *KnowledgeExtractor) GenerateEmbeddings() error {
	if !ke.IsEnabled() {
		return fmt.Errorf("knowledge extractor not enabled")
	}

	ke.mutex.RLock()
	defer ke.mutex.RUnlock()

	// Get AI manager for embedding generation
	ai := GetAIManager()
	if ai == nil {
		return fmt.Errorf("AI manager not available for embedding generation")
	}

	// Get vector database manager
	vectorDB := GetVectorDBManager()
	if vectorDB == nil {
		return fmt.Errorf("vector database manager not available")
	}

	// Initialize and enable vector database if needed
	if !vectorDB.IsEnabled() {
		if err := vectorDB.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize vector database: %v", err)
		}
		if err := vectorDB.Enable(); err != nil {
			return fmt.Errorf("failed to enable vector database: %v", err)
		}
	}

	// Count entities to process
	entitiesCount := len(ke.entities)
	if entitiesCount == 0 {
		return nil // No entities to process
	}

	fmt.Printf("Generating embeddings for %d knowledge entities...\n", entitiesCount)

	// Generate embeddings for entities
	processedCount := 0
	for id, entity := range ke.entities {
		// Create text representation of entity for embedding
		textToEmbed := fmt.Sprintf("%s %s %s", 
			entity.Type, 
			entity.Pattern, 
			strings.Join(entity.Examples, " "))

		// Generate embedding
		embedding, err := ai.GenerateEmbedding(textToEmbed)
		if err != nil {
			fmt.Printf("Warning: Failed to generate embedding for entity %s: %v\n", id, err)
			continue
		}

		// Create command embedding for vector database
		commandEmbedding := CommandEmbedding{
			CommandID:   "knowledge_" + id,
			Command:     entity.Pattern,
			Directory:   entity.Metadata["directory"],
			Timestamp:   entity.LastUpdated,
			ExitCode:    0,
			Embedding:   embedding,
			Metadata:    fmt.Sprintf(`{"type":"%s","examples":%d,"confidence":%.2f}`, 
							entity.Type, len(entity.Examples), entity.Confidence),
			Frequency:   entity.UsageCount,
			LastUsed:    entity.LastUpdated,
			SuccessRate: float32(entity.Confidence),
		}

		// Add to vector database
		err = vectorDB.AddCommandEmbedding(commandEmbedding)
		if err != nil {
			fmt.Printf("Warning: Failed to add embedding to vector database: %v\n", err)
			continue
		}

		processedCount++
	}

	fmt.Printf("Successfully generated embeddings for %d/%d knowledge entities\n", 
		processedCount, entitiesCount)

	return nil
}

// SearchKnowledge searches for knowledge items matching a query
func (ke *KnowledgeExtractor) SearchKnowledge(query string, limit int) ([]KnowledgeEntity, error) {
	if !ke.IsEnabled() {
		return nil, fmt.Errorf("knowledge extractor not enabled")
	}

	ke.mutex.RLock()
	defer ke.mutex.RUnlock()

	// First, try semantic search using vector database if available
	vectorResults, err := ke.searchKnowledgeWithVectors(query, limit)
	if err == nil && len(vectorResults) > 0 {
		// Vector search successful
		return vectorResults, nil
	}

	// Fall back to text-based search
	textResults := ke.searchKnowledgeWithText(query, limit)
	if len(textResults) > 0 {
		return textResults, nil
	}

	// If no results from either method, return an empty slice
	return []KnowledgeEntity{}, nil
}

// searchKnowledgeWithVectors performs semantic search using vector database
func (ke *KnowledgeExtractor) searchKnowledgeWithVectors(query string, limit int) ([]KnowledgeEntity, error) {
	// Get AI manager for embedding generation
	ai := GetAIManager()
	if ai == nil {
		return nil, fmt.Errorf("AI manager not available for embedding generation")
	}

	// Get vector database manager
	vectorDB := GetVectorDBManager()
	if vectorDB == nil || !vectorDB.IsEnabled() {
		return nil, fmt.Errorf("vector database not available or not enabled")
	}

	// Generate embedding for query
	embedding, err := ai.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %v", err)
	}

	// Search for similar commands in vector database
	similarCommands, err := vectorDB.SearchSimilarCommands(embedding, "", limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %v", err)
	}

	// Convert command embeddings to knowledge entities
	var results []KnowledgeEntity
	for _, cmd := range similarCommands {
		// Skip non-knowledge entries
		if !strings.HasPrefix(cmd.CommandID, "knowledge_") {
			continue
		}

		// Extract entity ID from command ID
		entityID := strings.TrimPrefix(cmd.CommandID, "knowledge_")
		
		// Look up entity in our map
		entity, exists := ke.entities[entityID]
		if exists {
			// Add entity to results
			results = append(results, entity)
		} else {
			// If entity doesn't exist in our map, create a synthetic one from the command
			var entityType KnowledgeType
			var confidence float64
			examples := []string{cmd.Command}
			
			// Try to parse metadata
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(cmd.Metadata), &metadata); err == nil {
				if typeStr, ok := metadata["type"].(string); ok {
					entityType = KnowledgeType(typeStr)
				}
				if conf, ok := metadata["confidence"].(float64); ok {
					confidence = conf
				}
			}
			
			// Create synthetic entity
			syntheticEntity := KnowledgeEntity{
				ID:          entityID,
				Type:        entityType,
				Pattern:     cmd.Command,
				Examples:    examples,
				Confidence:  confidence,
				LastUpdated: cmd.LastUsed,
				UsageCount:  cmd.Frequency,
				Metadata: map[string]string{
					"directory": cmd.Directory,
					"synthetic": "true",
				},
			}
			
			results = append(results, syntheticEntity)
		}
	}

	return results, nil
}

// searchKnowledgeWithText performs text-based search using string matching
func (ke *KnowledgeExtractor) searchKnowledgeWithText(query string, limit int) []KnowledgeEntity {
	query = strings.ToLower(query)
	var results []KnowledgeEntity
	
	// Search all entities for text matches
	for _, entity := range ke.entities {
		// Check if query matches pattern
		if strings.Contains(strings.ToLower(entity.Pattern), query) {
			results = append(results, entity)
			continue
		}
		
		// Check if query matches examples
		for _, example := range entity.Examples {
			if strings.Contains(strings.ToLower(example), query) {
				results = append(results, entity)
				break
			}
		}
		
		// Stop if we have enough results
		if len(results) >= limit {
			break
		}
	}
	
	// Sort results by confidence and usage count
	sortEntities(results)
	
	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}
	
	return results
}

// UpdateContext updates the current context with information from a directory
func (ke *KnowledgeExtractor) UpdateContext(directory string) error {
	if !ke.IsEnabled() {
		return fmt.Errorf("knowledge extractor not enabled")
	}

	// In a real implementation, this would analyze the directory
	// For now, just update the last scan time
	ke.lastScan = time.Now()

	return nil
}

// Context represents environment context information
type Context struct {
	OS                string              `json:"os"`
	Arch              string              `json:"arch"`
	Shell             string              `json:"shell"`
	User              string              `json:"user"`
	Hostname          string              `json:"hostname"`
	CurrentDir        string              `json:"current_dir"`
	HomeDir           string              `json:"home_dir"`
	ShellEnvironment  map[string]string   `json:"shell_environment"`
	ProjectType       string              `json:"project_type"`
	GitBranch         string              `json:"git_branch"`
	GitRepo           string              `json:"git_repo"`
	LastCommands      []string            `json:"last_commands"`
	PackageManagers   map[string]bool     `json:"package_managers"`
	DetectedTools     map[string]string   `json:"detected_tools"`
	FileExtensions    map[string]int      `json:"file_extensions"`
	DirectoryStats    map[string]int      `json:"directory_stats"`
	NetworkInterfaces []string            `json:"network_interfaces"`
	DockerInfo        map[string]string   `json:"docker_info"`
	KubernetesInfo    map[string]string   `json:"kubernetes_info"`
	RuntimeVersions   map[string]string   `json:"runtime_versions"`
	SystemLoad        map[string]float64  `json:"system_load"`
}

// GetCurrentContext returns detailed information about the current environment
func (ke *KnowledgeExtractor) GetCurrentContext() Context {
	context := Context{
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		Shell:            os.Getenv("SHELL"),
		User:             os.Getenv("USER"),
		PackageManagers:  make(map[string]bool),
		DetectedTools:    make(map[string]string),
		FileExtensions:   make(map[string]int),
		DirectoryStats:   make(map[string]int),
		RuntimeVersions:  make(map[string]string),
		SystemLoad:       make(map[string]float64),
	}
	
	// Get basic system info
	var err error
	context.HomeDir, err = os.UserHomeDir()
	if err != nil {
		context.HomeDir = os.Getenv("HOME")
	}
	
	context.Hostname, _ = os.Hostname()
	context.CurrentDir, _ = os.Getwd()
	
	// Collect shell environment variables
	context.ShellEnvironment = make(map[string]string)
	importantEnvVars := []string{
		"TERM", "PATH", "LANG", "LC_ALL", "EDITOR", "PAGER", 
		"PWD", "OLDPWD", "GOPATH", "GOROOT", "JAVA_HOME", 
		"PYTHONPATH", "NODE_PATH", "NVM_DIR", "VIRTUAL_ENV",
		"HISTSIZE", "HISTFILESIZE", "PROMPT_COMMAND",
	}
	
	for _, envVar := range importantEnvVars {
		value := os.Getenv(envVar)
		if value != "" {
			context.ShellEnvironment[envVar] = value
		}
	}
	
	// Determine project type
	context.ProjectType = detectProjectType(context.CurrentDir)
	
	// Get Git information
	context.GitBranch = getGitBranch(context.CurrentDir)
	context.GitRepo = getGitRemoteURL(context.CurrentDir)
	
	// Detect installed package managers
	context.PackageManagers = detectPackageManagers()
	
	// Detect installed tools
	context.DetectedTools = detectTools()
	
	// Collect file extension statistics
	context.FileExtensions = collectFileExtensions(context.CurrentDir)
	
	// Collect directory statistics
	context.DirectoryStats = collectDirectoryStats(context.CurrentDir)
	
	// Collect network interface information
	context.NetworkInterfaces = collectNetworkInterfaces()
	
	// Get Docker information if available
	context.DockerInfo = getDockerInfo()
	
	// Get Kubernetes information if available
	context.KubernetesInfo = getKubernetesInfo()
	
	// Get runtime versions
	context.RuntimeVersions = detectRuntimeVersions()
	
	// Get system load information
	context.SystemLoad = getSystemLoad()
	
	// Get last commands (this would come from the history in a real implementation)
	context.LastCommands = getLastCommands(5)
	
	return context
}

// detectProjectType tries to determine the type of project in the given directory
func detectProjectType(dir string) string {
	// Check for common project files
	projectFiles := map[string]string{
		"go.mod":        "go",
		"package.json":  "javascript",
		"Cargo.toml":    "rust",
		"pom.xml":       "java",
		"build.gradle":  "java",
		"Gemfile":       "ruby",
		"requirements.txt": "python",
		"setup.py":      "python",
		"composer.json": "php",
		"Dockerfile":    "docker",
		"CMakeLists.txt": "cmake",
		"Makefile":      "make",
	}
	
	for file, projectType := range projectFiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return projectType
		}
	}
	
	// Check for language-specific directories
	projectDirs := map[string]string{
		"src/main/java": "java",
		"node_modules": "javascript",
		".venv":        "python",
		"vendor/bundle": "ruby",
	}
	
	for directory, projectType := range projectDirs {
		if _, err := os.Stat(filepath.Join(dir, directory)); err == nil {
			return projectType
		}
	}
	
	return "unknown"
}

// getGitBranch gets the current Git branch name
func getGitBranch(dir string) string {
	// Check if .git directory exists
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return ""
	}
	
	// Try to read .git/HEAD file
	headFile := filepath.Join(dir, ".git", "HEAD")
	content, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}
	
	// Parse the content to extract branch name
	headContent := string(content)
	if strings.HasPrefix(headContent, "ref: refs/heads/") {
		return strings.TrimSpace(strings.TrimPrefix(headContent, "ref: refs/heads/"))
	}
	
	return strings.TrimSpace(headContent)
}

// getGitRemoteURL gets the Git remote URL
func getGitRemoteURL(dir string) string {
	// Check if .git directory exists
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return ""
	}
	
	// Try to read .git/config file
	configFile := filepath.Join(dir, ".git", "config")
	content, err := os.ReadFile(configFile)
	if err != nil {
		return ""
	}
	
	// Look for remote "origin" URL
	configContent := string(content)
	lines := strings.Split(configContent, "\n")
	inOriginSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "[remote \"origin\"]" {
			inOriginSection = true
			continue
		}
		
		if inOriginSection && strings.HasPrefix(line, "url = ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "url = "))
		}
		
		if inOriginSection && strings.HasPrefix(line, "[") {
			inOriginSection = false
		}
	}
	
	return ""
}

// detectPackageManagers detects installed package managers
func detectPackageManagers() map[string]bool {
	packageManagers := map[string]bool{
		"npm":     checkCommandExists("npm"),
		"yarn":    checkCommandExists("yarn"),
		"pip":     checkCommandExists("pip") || checkCommandExists("pip3"),
		"gem":     checkCommandExists("gem"),
		"cargo":   checkCommandExists("cargo"),
		"go":      checkCommandExists("go"),
		"maven":   checkCommandExists("mvn"),
		"gradle":  checkCommandExists("gradle"),
		"docker":  checkCommandExists("docker"),
		"kubectl": checkCommandExists("kubectl"),
		"helm":    checkCommandExists("helm"),
	}
	
	return packageManagers
}

// checkCommandExists checks if a command exists in PATH
func checkCommandExists(command string) bool {
	// This is a simple implementation - in a real environment you would use exec.LookPath
	path := os.Getenv("PATH")
	dirs := strings.Split(path, string(os.PathListSeparator))
	
	for _, dir := range dirs {
		fullPath := filepath.Join(dir, command)
		if fileExists(fullPath) || fileExists(fullPath+".exe") {
			return true
		}
	}
	
	return false
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// detectTools detects installed development tools
func detectTools() map[string]string {
	tools := make(map[string]string)
	
	// Check for common development tools
	toolCommands := []string{
		"git", "python", "python3", "node", "java", "javac", "ruby", "gcc", "g++",
		"make", "docker", "kubectl", "terraform", "ansible", "vagrant", "virtualbox",
	}
	
	for _, tool := range toolCommands {
		if checkCommandExists(tool) {
			// In a real implementation, you would get version information
			// by running the tool with --version or similar
			tools[tool] = "installed"
		}
	}
	
	return tools
}

// collectFileExtensions collects statistics about file extensions in a directory
func collectFileExtensions(dir string) map[string]int {
	extensions := make(map[string]int)
	
	// In a real implementation, you would walk the directory tree
	// This is a simplified version that just checks the current directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return extensions
	}
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		ext := filepath.Ext(file.Name())
		if ext != "" {
			extensions[ext]++
		}
	}
	
	return extensions
}

// collectDirectoryStats collects statistics about directories
func collectDirectoryStats(dir string) map[string]int {
	stats := map[string]int{
		"total_files": 0,
		"total_dirs":  0,
	}
	
	// In a real implementation, you would walk the directory tree
	// This is a simplified version that just checks the current directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return stats
	}
	
	for _, file := range files {
		if file.IsDir() {
			stats["total_dirs"]++
		} else {
			stats["total_files"]++
		}
	}
	
	return stats
}

// collectNetworkInterfaces gets information about network interfaces
func collectNetworkInterfaces() []string {
	// In a real implementation, you would use net package to get network interfaces
	// This is a simplified version
	return []string{"lo", "eth0", "wlan0"}
}

// getDockerInfo gets information about Docker if installed
func getDockerInfo() map[string]string {
	// In a real implementation, you would run docker commands to get info
	// This is a simplified version
	dockerInfo := make(map[string]string)
	
	if checkCommandExists("docker") {
		dockerInfo["installed"] = "true"
		dockerInfo["version"] = "detected"
	}
	
	return dockerInfo
}

// getKubernetesInfo gets information about Kubernetes if installed
func getKubernetesInfo() map[string]string {
	// In a real implementation, you would run kubectl commands to get info
	// This is a simplified version
	kubeInfo := make(map[string]string)
	
	if checkCommandExists("kubectl") {
		kubeInfo["installed"] = "true"
		kubeInfo["version"] = "detected"
	}
	
	return kubeInfo
}

// detectRuntimeVersions detects versions of installed runtimes
func detectRuntimeVersions() map[string]string {
	// In a real implementation, you would run commands to get version info
	// This is a simplified version
	versions := make(map[string]string)
	
	versions["go"] = runtime.Version()
	
	return versions
}

// getSystemLoad gets system load information
func getSystemLoad() map[string]float64 {
	// In a real implementation, you would get actual system load
	// This is a simplified version
	load := make(map[string]float64)
	
	load["cpu"] = 0.0
	load["memory"] = 0.0
	
	return load
}

// getLastCommands gets the last N commands from history
func getLastCommands(n int) []string {
	// In a real implementation, you would read from history file
	// This is a simplified version with sample data
	commands := []string{
		"git status",
		"git diff",
		"make build",
		"./deltacli",
		"go test ./...",
	}
	
	if len(commands) > n {
		return commands[:n]
	}
	
	return commands
}

// ProjectInfo represents information about a detected project
type ProjectInfo struct {
	Type           string            `json:"type"`
	Path           string            `json:"path"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Dependencies   []string          `json:"dependencies"`
	Languages      []string          `json:"languages"`
	BuildSystem    string            `json:"build_system"`
	TestFramework  string            `json:"test_framework"`
	Config         map[string]string `json:"config"`
	RepoURL        string            `json:"repo_url"`
	Branch         string            `json:"branch"`
	LastModified   time.Time         `json:"last_modified"`
	CodeStats      map[string]int    `json:"code_stats"`
	Contributors   []string          `json:"contributors"`
	Readme         string            `json:"readme"`
}

// GetProjectInfo returns detailed information about the current project
func (ke *KnowledgeExtractor) GetProjectInfo() ProjectInfo {
	context := ke.GetCurrentContext()
	
	// Initialize the project info with data from the context
	projectInfo := ProjectInfo{
		Type:         context.ProjectType,
		Path:         context.CurrentDir,
		Name:         filepath.Base(context.CurrentDir),
		RepoURL:      context.GitRepo,
		Branch:       context.GitBranch,
		LastModified: time.Now(),
		Config:       make(map[string]string),
		CodeStats:    make(map[string]int),
	}
	
	// Copy file extension stats to code stats
	for ext, count := range context.FileExtensions {
		projectInfo.CodeStats[ext] = count
	}
	
	// Set languages based on file extensions and project type
	projectInfo.Languages = detectProjectLanguages(context)
	
	// Determine build system
	projectInfo.BuildSystem = detectBuildSystem(context)
	
	// Determine test framework
	projectInfo.TestFramework = detectTestFramework(context.ProjectType, projectInfo.BuildSystem)
	
	// Check for version information
	projectInfo.Version = detectProjectVersion(context.CurrentDir, context.ProjectType)
	
	// Get dependencies from appropriate files
	projectInfo.Dependencies = detectDependencies(context.CurrentDir, context.ProjectType)
	
	// Try to find README file
	projectInfo.Readme = findReadmeContent(context.CurrentDir)
	
	// Get contributors if git repo
	if context.GitRepo != "" {
		projectInfo.Contributors = getGitContributors(context.CurrentDir)
	}
	
	return projectInfo
}

// detectProjectLanguages determines the programming languages used in a project
func detectProjectLanguages(context Context) []string {
	languages := make(map[string]bool)
	
	// Add language based on project type
	switch context.ProjectType {
	case "go":
		languages["Go"] = true
	case "javascript", "typescript":
		languages["JavaScript"] = true
		// Check for TypeScript
		if _, ok := context.FileExtensions[".ts"]; ok {
			languages["TypeScript"] = true
		}
	case "java":
		languages["Java"] = true
	case "python":
		languages["Python"] = true
	case "rust":
		languages["Rust"] = true
	case "ruby":
		languages["Ruby"] = true
	case "php":
		languages["PHP"] = true
	}
	
	// Add languages based on file extensions
	for ext := range context.FileExtensions {
		switch ext {
		case ".go":
			languages["Go"] = true
		case ".js":
			languages["JavaScript"] = true
		case ".ts":
			languages["TypeScript"] = true
		case ".py":
			languages["Python"] = true
		case ".java":
			languages["Java"] = true
		case ".kt", ".kts":
			languages["Kotlin"] = true
		case ".rb":
			languages["Ruby"] = true
		case ".php":
			languages["PHP"] = true
		case ".c", ".h":
			languages["C"] = true
		case ".cpp", ".hpp", ".cc", ".hh":
			languages["C++"] = true
		case ".cs":
			languages["C#"] = true
		case ".rs":
			languages["Rust"] = true
		case ".swift":
			languages["Swift"] = true
		case ".sh", ".bash":
			languages["Shell"] = true
		case ".html", ".htm":
			languages["HTML"] = true
		case ".css":
			languages["CSS"] = true
		}
	}
	
	// Convert map to slice
	result := make([]string, 0, len(languages))
	for lang := range languages {
		result = append(result, lang)
	}
	
	// Sort languages alphabetically
	sort.Strings(result)
	
	return result
}

// detectBuildSystem determines the build system used in a project
func detectBuildSystem(context Context) string {
	// Check if Makefile exists
	if _, ok := context.FileExtensions[".mk"]; ok || fileExists(filepath.Join(context.CurrentDir, "Makefile")) {
		return "make"
	}
	
	// Determine build system based on project type
	switch context.ProjectType {
	case "go":
		return "go build"
	case "javascript", "typescript":
		if fileExists(filepath.Join(context.CurrentDir, "package.json")) {
			// Check for yarn.lock
			if fileExists(filepath.Join(context.CurrentDir, "yarn.lock")) {
				return "yarn"
			}
			return "npm"
		}
	case "java":
		if fileExists(filepath.Join(context.CurrentDir, "pom.xml")) {
			return "maven"
		}
		if fileExists(filepath.Join(context.CurrentDir, "build.gradle")) {
			return "gradle"
		}
	case "python":
		if fileExists(filepath.Join(context.CurrentDir, "setup.py")) {
			return "setup.py"
		}
		if fileExists(filepath.Join(context.CurrentDir, "requirements.txt")) {
			return "pip"
		}
	case "rust":
		return "cargo"
	case "ruby":
		if fileExists(filepath.Join(context.CurrentDir, "Gemfile")) {
			return "bundle"
		}
	case "php":
		if fileExists(filepath.Join(context.CurrentDir, "composer.json")) {
			return "composer"
		}
	}
	
	return "unknown"
}

// detectTestFramework determines the test framework used in a project
func detectTestFramework(projectType string, buildSystem string) string {
	switch projectType {
	case "go":
		return "go test"
	case "javascript", "typescript":
		if buildSystem == "npm" || buildSystem == "yarn" {
			// Could check package.json for test frameworks like jest, mocha, etc.
			return "npm test"
		}
	case "java":
		if buildSystem == "maven" {
			return "junit"
		}
		if buildSystem == "gradle" {
			return "junit"
		}
	case "python":
		// Could check for pytest, unittest, etc.
		return "pytest"
	case "rust":
		return "cargo test"
	case "ruby":
		return "rspec"
	}
	
	return "unknown"
}

// detectProjectVersion tries to determine the project version
func detectProjectVersion(dir string, projectType string) string {
	switch projectType {
	case "go":
		// Try to read version from go.mod
		if fileExists(filepath.Join(dir, "go.mod")) {
			content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err == nil {
				// Simple version extraction, could be improved
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "module ") {
						// Module name might have version in path
						parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "module ")), "/")
						if len(parts) > 0 && strings.HasPrefix(parts[len(parts)-1], "v") {
							return parts[len(parts)-1]
						}
					}
				}
			}
		}
	case "javascript", "typescript":
		// Try to read version from package.json
		if fileExists(filepath.Join(dir, "package.json")) {
			content, err := os.ReadFile(filepath.Join(dir, "package.json"))
			if err == nil {
				// Simple version extraction, could be improved
				var packageJSON map[string]interface{}
				if err := json.Unmarshal(content, &packageJSON); err == nil {
					if version, ok := packageJSON["version"].(string); ok {
						return version
					}
				}
			}
		}
	}
	
	return "0.1.0" // Default version if not found
}

// detectDependencies tries to extract project dependencies
func detectDependencies(dir string, projectType string) []string {
	dependencies := []string{}
	
	switch projectType {
	case "go":
		// Read from go.mod
		if fileExists(filepath.Join(dir, "go.mod")) {
			content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err == nil {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "require ") && !strings.HasSuffix(line, "(") {
						dep := strings.TrimPrefix(line, "require ")
						dependencies = append(dependencies, dep)
					} else if strings.HasPrefix(line, "github.com/") || 
							  strings.HasPrefix(line, "golang.org/") || 
							  strings.HasPrefix(line, "gopkg.in/") {
						dependencies = append(dependencies, line)
					}
				}
			}
		}
	case "javascript", "typescript":
		// Read from package.json
		if fileExists(filepath.Join(dir, "package.json")) {
			content, err := os.ReadFile(filepath.Join(dir, "package.json"))
			if err == nil {
				var packageJSON map[string]interface{}
				if err := json.Unmarshal(content, &packageJSON); err == nil {
					// Get dependencies
					if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
						for dep := range deps {
							dependencies = append(dependencies, dep)
						}
					}
					// Get devDependencies
					if devDeps, ok := packageJSON["devDependencies"].(map[string]interface{}); ok {
						for dep := range devDeps {
							dependencies = append(dependencies, dep+" (dev)")
						}
					}
				}
			}
		}
	}
	
	// Sort dependencies
	sort.Strings(dependencies)
	
	// Limit number of dependencies to return
	maxDeps := 20
	if len(dependencies) > maxDeps {
		dependencies = dependencies[:maxDeps]
	}
	
	return dependencies
}

// findReadmeContent tries to find and read the README file
func findReadmeContent(dir string) string {
	readmeFiles := []string{
		"README.md",
		"README",
		"README.txt",
		"README.markdown",
		"readme.md",
	}
	
	for _, name := range readmeFiles {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			content, err := os.ReadFile(path)
			if err == nil {
				// Limit readme content length
				readmeContent := string(content)
				if len(readmeContent) > 500 {
					readmeContent = readmeContent[:500] + "...(truncated)"
				}
				return readmeContent
			}
		}
	}
	
	return ""
}

// getGitContributors gets the list of contributors from git
func getGitContributors(dir string) []string {
	// In a real implementation, you would run:
	// git log --format='%aN' | sort -u
	
	// Return a default list
	return []string{"User", "Contributor"}
}

// AddCommand adds a command for knowledge extraction
func (ke *KnowledgeExtractor) AddCommand(command, directory string, exitCode int) error {
	if !ke.IsEnabled() {
		return nil
	}

	// Sample implementation - would process command for knowledge extraction
	ke.lastScan = time.Now()
	return nil
}

// GetKnowledgeItems returns knowledge items
func (ke *KnowledgeExtractor) GetKnowledgeItems() []struct {
	Source     string
	Type       string
	Content    string
	Context    string
	Path       string
	Tags       []string
	Confidence float64
} {
	// Return sample knowledge items
	return []struct {
		Source     string
		Type       string
		Content    string
		Context    string
		Path       string
		Tags       []string
		Confidence float64
	}{
		{
			Source:     "command",
			Type:       "git",
			Content:    "git commit",
			Context:    "/home/bleepbloop/deltacli",
			Path:       "/home/bleepbloop/deltacli",
			Tags:       []string{"git", "version-control"},
			Confidence: 0.9,
		},
		{
			Source:     "environment",
			Type:       "shell",
			Content:    "bash",
			Context:    "/home/bleepbloop/deltacli",
			Path:       "/home/bleepbloop/deltacli",
			Tags:       []string{"shell", "environment"},
			Confidence: 1.0,
		},
	}
}


// Global KnowledgeExtractor instance
var globalKnowledgeExtractor *KnowledgeExtractor

// GetKnowledgeExtractor returns the global KnowledgeExtractor instance
func GetKnowledgeExtractor() *KnowledgeExtractor {
	if globalKnowledgeExtractor == nil {
		var err error
		globalKnowledgeExtractor, err = NewKnowledgeExtractor()
		if err != nil {
			fmt.Printf("Error initializing knowledge extractor: %v\n", err)
			return nil
		}
	}
	return globalKnowledgeExtractor
}