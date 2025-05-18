package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"math"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// VectorDBConfig contains configuration for the vector database
type VectorDBConfig struct {
	Enabled            bool     `json:"enabled"`
	DBPath             string   `json:"db_path"`
	EmbeddingDimension int      `json:"embedding_dimension"`
	DistanceMetric     string   `json:"distance_metric"` // "cosine", "dot", "euclidean", "manhattan", "jaccard"
	MaxEntries         int      `json:"max_entries"`
	IndexBuildInterval int      `json:"index_build_interval"` // in minutes
	CommandTypes       []string `json:"command_types"`       // Specific types of commands to embed
	InMemoryMode       bool     `json:"in_memory_mode"`     // Whether to keep vectors in memory
	JaccardThreshold   float32  `json:"jaccard_threshold"`  // Threshold for Jaccard similarity (default 0.1)
}

// CommandEmbedding represents a single command with its embedding
type CommandEmbedding struct {
	CommandID   string    `json:"command_id"`
	Command     string    `json:"command"`
	Directory   string    `json:"directory"`
	Timestamp   time.Time `json:"timestamp"`
	ExitCode    int       `json:"exit_code"`
	Embedding   []float32 `json:"embedding"`
	Metadata    string    `json:"metadata"`
	Frequency   int       `json:"frequency"`
	LastUsed    time.Time `json:"last_used"`
	SuccessRate float32   `json:"success_rate"`
}

// VectorDBManager manages the vector database for semantic search
type VectorDBManager struct {
	config         VectorDBConfig
	configPath     string
	db             *sql.DB
	mutex          sync.RWMutex
	isInitialized  bool
	lastIndexBuild time.Time
}

// NewVectorDBManager creates a new vector database manager
func NewVectorDBManager() (*VectorDBManager, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/vector directory
	configDir := filepath.Join(homeDir, ".config", "delta", "vector")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector DB directory: %v", err)
	}

	configPath := filepath.Join(configDir, "vector_config.json")
	dbPath := filepath.Join(configDir, "command_vectors.db")

	// Create default configuration
	manager := &VectorDBManager{
		config: VectorDBConfig{
			Enabled:            false,
			DBPath:             dbPath,
			EmbeddingDimension: 384,  // Default dimension for small embeddings
			DistanceMetric:     "cosine",
			MaxEntries:         10000,
			IndexBuildInterval: 60,    // 1 hour
			CommandTypes:       []string{"shell", "git", "docker", "npm", "python"},
			JaccardThreshold:   0.1,   // Default threshold for Jaccard similarity
		},
		configPath:     configPath,
		mutex:          sync.RWMutex{},
		isInitialized:  false,
		lastIndexBuild: time.Time{},
	}

	// Try to load existing configuration
	err = manager.loadConfig()
	if err != nil {
		// Save the default configuration if loading fails
		manager.saveConfig()
	}

	return manager, nil
}

// Initialize initializes the vector database
func (vm *VectorDBManager) Initialize() error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	// Ensure the database directory exists
	dir := filepath.Dir(vm.config.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open the SQLite database
	db, err := sql.Open("sqlite3", vm.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	vm.db = db

	// Initialize the database schema if needed
	err = vm.initializeSchema()
	if err != nil {
		vm.db.Close()
		return fmt.Errorf("failed to initialize database schema: %v", err)
	}

	vm.isInitialized = true
	return nil
}

// initializeSchema creates the necessary tables and indexes in the database
func (vm *VectorDBManager) initializeSchema() error {
	// Create the command_embeddings table
	_, err := vm.db.Exec(`
		CREATE TABLE IF NOT EXISTS command_embeddings (
			command_id TEXT PRIMARY KEY,
			command TEXT NOT NULL,
			directory TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			exit_code INTEGER NOT NULL,
			embedding BLOB NOT NULL,
			metadata TEXT,
			frequency INTEGER DEFAULT 1,
			last_used INTEGER NOT NULL,
			success_rate REAL DEFAULT 1.0
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = vm.db.Exec(`CREATE INDEX IF NOT EXISTS idx_command ON command_embeddings(command)`)
	if err != nil {
		return err
	}

	_, err = vm.db.Exec(`CREATE INDEX IF NOT EXISTS idx_directory ON command_embeddings(directory)`)
	if err != nil {
		return err
	}

	_, err = vm.db.Exec(`CREATE INDEX IF NOT EXISTS idx_timestamp ON command_embeddings(timestamp)`)
	if err != nil {
		return err
	}

	// Try to load the vectorx extension first
	_, err = vm.db.Exec(`SELECT load_extension('vectorx')`)
	
	// If the extension load fails, try a few common paths where it might be installed
	if err != nil {
		commonPaths := []string{
			"libvectorx", // System might find it in library path
			"/usr/lib/sqlite3/vectorx",
			"/usr/local/lib/sqlite3/vectorx",
			"./vectorx", // Current directory
			"/usr/lib/libvectorx",
			"/usr/local/lib/libvectorx",
		}
		
		extensionLoaded := false
		for _, path := range commonPaths {
			_, loadErr := vm.db.Exec(fmt.Sprintf(`SELECT load_extension('%s')`, path))
			if loadErr == nil {
				extensionLoaded = true
				fmt.Printf("Successfully loaded vectorx extension from %s\n", path)
				break
			}
		}
		
		if !extensionLoaded {
			fmt.Printf("Warning: SQLite vectorx extension not available - falling back to in-memory search: %v\n", err)
			fmt.Println("To enable vectorx, install the SQLite vectorx extension")
			
			// Log information about how to install the vectorx extension
			fmt.Println("Installation tips:")
			fmt.Println("  1. Download vectorx extension from https://github.com/asg017/sqlite-vectorx/releases")
			fmt.Println("  2. Place the extension file in your library path")
			fmt.Println("  3. Load the extension with SQLite's .load directive")
			
			// We will fall back to a standard implementation that does vector distance calculation in Go
		}
	} else {
		fmt.Println("SQLite vectorx extension loaded successfully")
	}
	
	// Try to create the vector index table
	createVectorTableSQL := fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS vector_index USING vectorx(
			embedding(%d)
		)
	`, vm.config.EmbeddingDimension)
	
	_, err = vm.db.Exec(createVectorTableSQL)
	
	// Check if the vector table creation was successful
	if err != nil {
		fmt.Printf("Warning: Failed to create vector index table: %v\n", err)
		fmt.Println("Will use in-memory vector search as fallback")
	} else {
		fmt.Println("SQLite vectorx virtual table created successfully")
		
		// Insert any existing embeddings into the index
		rowsAffected, err := vm.rebuildVectorIndex()
		if err != nil {
			fmt.Printf("Warning: Failed to build initial vector index: %v\n", err)
		} else if rowsAffected > 0 {
			fmt.Printf("Built vector index with %d embeddings\n", rowsAffected)
		}
	}

	return nil
}

// loadConfig loads the vector database configuration from disk
func (vm *VectorDBManager) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(vm.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(vm.configPath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data
	return json.Unmarshal(data, &vm.config)
}

// saveConfig saves the vector database configuration to disk
func (vm *VectorDBManager) saveConfig() error {
	// Marshal the config to JSON with indentation for readability
	data, err := json.MarshalIndent(vm.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(vm.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(vm.configPath, data, 0644)
}

// IsEnabled returns whether the vector database is enabled
func (vm *VectorDBManager) IsEnabled() bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()
	return vm.config.Enabled && vm.isInitialized
}

// Enable enables the vector database
func (vm *VectorDBManager) Enable() error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	if !vm.isInitialized {
		return fmt.Errorf("vector database not initialized")
	}

	vm.config.Enabled = true
	return vm.saveConfig()
}

// Disable disables the vector database
func (vm *VectorDBManager) Disable() error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	vm.config.Enabled = false
	return vm.saveConfig()
}

// AddCommandEmbedding adds a command embedding to the database
func (vm *VectorDBManager) AddCommandEmbedding(embedding CommandEmbedding) error {
	if !vm.IsEnabled() {
		return nil
	}

	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	// Check if the command already exists
	var exists bool
	err := vm.db.QueryRow("SELECT 1 FROM command_embeddings WHERE command_id = ?", embedding.CommandID).Scan(&exists)
	
	if err == nil {
		// Command exists, update it
		_, err = vm.db.Exec(
			`UPDATE command_embeddings SET 
				frequency = frequency + 1, 
				last_used = ?, 
				success_rate = ? 
			WHERE command_id = ?`,
			embedding.LastUsed.Unix(),
			embedding.SuccessRate,
			embedding.CommandID,
		)
		return err
	} else if err != sql.ErrNoRows {
		return err
	}

	// Command doesn't exist, insert it
	// Convert embedding to bytes
	embeddingJSON, err := json.Marshal(embedding.Embedding)
	if err != nil {
		return err
	}

	// Insert into database
	_, err = vm.db.Exec(
		`INSERT INTO command_embeddings 
			(command_id, command, directory, timestamp, exit_code, embedding, metadata, frequency, last_used, success_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		embedding.CommandID,
		embedding.Command,
		embedding.Directory,
		embedding.Timestamp.Unix(),
		embedding.ExitCode,
		embeddingJSON,
		embedding.Metadata,
		embedding.Frequency,
		embedding.LastUsed.Unix(),
		embedding.SuccessRate,
	)

	if err != nil {
		return err
	}

	// Check if we need to rebuild the index
	if time.Since(vm.lastIndexBuild).Minutes() > float64(vm.config.IndexBuildInterval) {
		go vm.rebuildIndex()
	}

	return nil
}

// rebuildIndex rebuilds the vector index
func (vm *VectorDBManager) rebuildIndex() {
	// This function is called in a goroutine, so we need to handle panics
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in rebuildIndex: %v\n", r)
		}
	}()

	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	rowsAffected, err := vm.rebuildVectorIndex()
	if err != nil {
		fmt.Printf("Error rebuilding vector index: %v\n", err)
	} else if rowsAffected > 0 {
		fmt.Printf("Rebuilt vector index with %d embeddings\n", rowsAffected)
	}

	vm.lastIndexBuild = time.Now()
}

// rebuildVectorIndex rebuilds the vector index and returns the number of rows affected
func (vm *VectorDBManager) rebuildVectorIndex() (int64, error) {
	// Check if the vector extension is available
	_, err := vm.db.Exec("SELECT * FROM vector_index LIMIT 1")
	if err != nil {
		return 0, fmt.Errorf("vector extension not available: %v", err)
	}
	
	// Start a transaction for improved performance
	tx, err := vm.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %v", err)
	}
	
	// Ensure transaction is either committed or rolled back
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()
	
	// Vector extension is available, rebuild the index
	_, err = tx.Exec("DELETE FROM vector_index")
	if err != nil {
		return 0, fmt.Errorf("failed to clear vector index: %v", err)
	}

	// Get the count of embeddings
	var count int64
	err = tx.QueryRow("SELECT COUNT(*) FROM command_embeddings").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count embeddings: %v", err)
	}
	
	// If no embeddings, just commit and return
	if count == 0 {
		err = tx.Commit()
		tx = nil
		if err != nil {
			return 0, fmt.Errorf("failed to commit transaction: %v", err)
		}
		return 0, nil
	}

	// Insert embeddings into vector index in batches for performance
	// We need to convert BLOB to proper vector format for vectorx
	batchSize := 1000
	totalInserted := int64(0)
	
	// Prepare a statement for better performance
	stmt, err := tx.Prepare(`
		INSERT INTO vector_index (rowid, embedding) 
		VALUES (?, json(?))
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()
	
	// Fetch embeddings in chunks
	offset := 0
	for {
		rows, err := tx.Query(`
			SELECT rowid, embedding
			FROM command_embeddings
			ORDER BY rowid
			LIMIT ? OFFSET ?
		`, batchSize, offset)
		
		if err != nil {
			return totalInserted, fmt.Errorf("failed to fetch embeddings: %v", err)
		}
		
		// Process the batch
		rowCount := 0
		for rows.Next() {
			var (
				rowid int64
				embeddingBlob []byte
			)
			
			err := rows.Scan(&rowid, &embeddingBlob)
			if err != nil {
				rows.Close()
				return totalInserted, fmt.Errorf("failed to scan row: %v", err)
			}
			
			// Insert into vector index
			_, err = stmt.Exec(rowid, string(embeddingBlob))
			if err != nil {
				rows.Close()
				return totalInserted, fmt.Errorf("failed to insert into vector index: %v", err)
			}
			
			rowCount++
			totalInserted++
		}
		rows.Close()
		
		// If we got fewer rows than batch size, we're done
		if rowCount < batchSize {
			break
		}
		
		// Increment offset for next batch
		offset += batchSize
	}
	
	// Commit the transaction
	err = tx.Commit()
	tx = nil
	if err != nil {
		return totalInserted, fmt.Errorf("failed to commit transaction: %v", err)
	}
	
	// Update last index build time
	vm.lastIndexBuild = time.Now()
	
	return totalInserted, nil
}

// SearchSimilarCommands searches for similar commands
func (vm *VectorDBManager) SearchSimilarCommands(query []float32, context string, limit int) ([]CommandEmbedding, error) {
	if !vm.IsEnabled() {
		return nil, fmt.Errorf("vector database not enabled")
	}

	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	var commands []CommandEmbedding

	// Check if the vector extension is available
	hasVectorX := vm.hasVectorExtension()
	
	if hasVectorX {
		// Vector extension is available, use it for search
		// Convert query to JSON
		queryJSON, err := json.Marshal(query)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query: %v", err)
		}

		// Prepare context filter if provided
		contextFilter := ""
		var args []interface{}
		
		if context != "" {
			contextFilter = "AND ce.directory LIKE ?"
			args = append(args, "%"+context+"%")
		}
		
		// Determine which similarity function and metrics to use based on config
		var vectorxFuncName string
		var isDistanceMetric bool // If true, smaller is better (ASC), if false, larger is better (DESC)

		switch vm.config.DistanceMetric {
		case "cosine":
			// similarityType = "cosine_similarity"
			vectorxFuncName = "vectorx_cosine_similarity"
			isDistanceMetric = false // Higher is better
		case "dot":
			// similarityType = "dot_product"
			vectorxFuncName = "vectorx_dot_product"
			isDistanceMetric = false // Higher is better
		case "euclidean":
			// similarityType = "euclidean_distance"
			vectorxFuncName = "vectorx_euclidean_distance"
			isDistanceMetric = true // Lower is better
		case "manhattan":
			// similarityType = "manhattan_distance"
			vectorxFuncName = "vectorx_manhattan_distance"
			isDistanceMetric = true // Lower is better
		case "jaccard":
			// similarityType = "jaccard_similarity"
			vectorxFuncName = "vectorx_jaccard_similarity"
			isDistanceMetric = false // Higher is better
		default:
			// Default to cosine similarity if the metric isn't recognized
			// similarityType = "cosine_similarity"
			vectorxFuncName = "vectorx_cosine_similarity"
			isDistanceMetric = false
		}
		
		// Check if the vectorx function we want is available
		hasSimilarityFunc := vm.hasVectorFunction(vectorxFuncName)
		
		if hasSimilarityFunc {
			// Build the query string
			var queryStr strings.Builder
			queryStr.WriteString(`
				SELECT ce.command_id, ce.command, ce.directory, ce.timestamp, ce.exit_code, 
					   ce.embedding, ce.metadata, ce.frequency, ce.last_used, ce.success_rate,
					   `)
			queryStr.WriteString(vectorxFuncName)
			
			// For distance metrics, smaller is better; for similarity metrics, larger is better
			var orderDir string
			if isDistanceMetric {
				orderDir = "ASC"
			} else {
				orderDir = "DESC"
			}
			
			queryStr.WriteString(`(vi.embedding, json(?)) AS similarity
				FROM command_embeddings ce
				JOIN vector_index vi ON ce.rowid = vi.rowid
				WHERE 1=1 `)
			queryStr.WriteString(contextFilter)
			queryStr.WriteString(`
				ORDER BY similarity `)
			queryStr.WriteString(orderDir)
			queryStr.WriteString(`
				LIMIT ?`)
			
			// Add args
			args = append([]interface{}{string(queryJSON)}, args...)
			args = append(args, limit)
			
			// Execute query
			rows, err := vm.db.Query(queryStr.String(), args...)
			if err == nil {
				defer rows.Close()
				// We have 11 columns in the result (10 from command + similarity)
				commands, err := vm.scanCommandRowsWithSimilarity(rows)
				if err == nil {
					return commands, nil
				}
				fmt.Printf("Error scanning vector search results: %v\n", err)
			} else {
				fmt.Printf("Vector search with %s failed, trying fallback: %v\n", vectorxFuncName, err)
			}
		} else {
			fmt.Printf("Vector function %s not available, using fallback\n", vectorxFuncName)
		}
	}

	// Fall back to in-memory search
	// Get all commands
	rows, err := vm.db.Query(`
		SELECT command_id, command, directory, timestamp, exit_code, 
			   embedding, metadata, frequency, last_used, success_rate
		FROM command_embeddings
		WHERE directory LIKE ?
		ORDER BY frequency DESC, last_used DESC
		LIMIT 1000
	`, "%"+context+"%")

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan all rows
	allCommands, err := vm.scanCommandRows(rows)
	if err != nil {
		return nil, err
	}

	// Calculate distances in memory
	type distanceEntry struct {
		cmd      CommandEmbedding
		score    float32  // Higher means more similar for all metrics (we'll invert distances)
		rawScore float32  // The actual raw score (for debugging/metadata)
	}
	
	distances := make([]distanceEntry, len(allCommands))

	// Determine which similarity function to use based on config
	var similarityFuncType string
	
	switch vm.config.DistanceMetric {
	case "cosine":
		similarityFuncType = "cosine_similarity"
	case "dot":
		similarityFuncType = "dot_product"
	case "euclidean":
		similarityFuncType = "euclidean_distance"
	case "manhattan":
		similarityFuncType = "manhattan_distance"
	case "jaccard":
		similarityFuncType = "jaccard_similarity"
	default:
		// Default to cosine similarity if the metric isn't recognized
		similarityFuncType = "cosine_similarity"
	}
	
	// Calculate similarity based on the selected metric
	for i, cmd := range allCommands {
		distances[i].cmd = cmd
		
		var score float32
		var rawScore float32
		
		// Use the appropriate similarity function based on the config
		switch similarityFuncType {
		case "cosine_similarity":
			// For cosine similarity, higher is better
			score = cosineSimilarity(query, cmd.Embedding)
			rawScore = score
		case "dot_product":
			// For dot product, higher is better
			score = dotProduct(query, cmd.Embedding)
			rawScore = score
		case "euclidean_distance":
			// For euclidean distance, lower is better, so invert for scoring
			distance := euclideanDistance(query, cmd.Embedding)
			if distance > 0 {
				score = 1.0 / distance  // Invert so higher means more similar
			}
			rawScore = distance  // Store the actual distance for reference
		case "manhattan_distance":
			// For manhattan distance, lower is better, so invert for scoring
			distance := manhattanDistance(query, cmd.Embedding)
			if distance > 0 {
				score = 1.0 / distance  // Invert so higher means more similar
			}
			rawScore = distance  // Store the actual distance for reference
		case "jaccard_similarity":
			// For jaccard similarity, higher is better
			score = jaccardSimilarity(query, cmd.Embedding, vm.config.JaccardThreshold)
			rawScore = score
		default:
			// Default to cosine similarity
			score = cosineSimilarity(query, cmd.Embedding)
			rawScore = score
		}
		
		distances[i].score = score
		distances[i].rawScore = rawScore
		
		// Store the raw score in the command's metadata for reference
		metadataMap := make(map[string]interface{})
		if cmd.Metadata != "" {
			// Try to parse existing metadata
			if err := json.Unmarshal([]byte(cmd.Metadata), &metadataMap); err != nil {
				// If it fails, just create a new map
				metadataMap = make(map[string]interface{})
			}
		}
		
		metadataMap["similarity"] = rawScore
		metadataMap["metric"] = vm.config.DistanceMetric
		
		metadataBytes, err := json.Marshal(metadataMap)
		if err == nil {
			distances[i].cmd.Metadata = string(metadataBytes)
		}
	}

	// Sort by score (higher is better for all metrics after our conversion)
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].score > distances[j].score
	})

	// Return top results
	resultLimit := limit
	if resultLimit > len(distances) {
		resultLimit = len(distances)
	}

	for i := 0; i < resultLimit; i++ {
		commands = append(commands, distances[i].cmd)
	}

	return commands, nil
}

// scanCommandRows scans rows from a database query into CommandEmbedding objects
func (vm *VectorDBManager) scanCommandRows(rows *sql.Rows) ([]CommandEmbedding, error) {
	var commands []CommandEmbedding

	for rows.Next() {
		var (
			commandID   string
			command     string
			directory   string
			timestamp   int64
			exitCode    int
			embeddingJSON []byte
			metadata    string
			frequency   int
			lastUsed    int64
			successRate float32
		)

		err := rows.Scan(
			&commandID, &command, &directory, &timestamp, &exitCode,
			&embeddingJSON, &metadata, &frequency, &lastUsed, &successRate,
		)
		if err != nil {
			return nil, err
		}

		// Parse embedding
		var embedding []float32
		err = json.Unmarshal(embeddingJSON, &embedding)
		if err != nil {
			return nil, err
		}

		commands = append(commands, CommandEmbedding{
			CommandID:   commandID,
			Command:     command,
			Directory:   directory,
			Timestamp:   time.Unix(timestamp, 0),
			ExitCode:    exitCode,
			Embedding:   embedding,
			Metadata:    metadata,
			Frequency:   frequency,
			LastUsed:    time.Unix(lastUsed, 0),
			SuccessRate: successRate,
		})
	}

	return commands, nil
}

// scanCommandRowsWithSimilarity scans rows from a vector search query including similarity scores
func (vm *VectorDBManager) scanCommandRowsWithSimilarity(rows *sql.Rows) ([]CommandEmbedding, error) {
	var commands []CommandEmbedding

	for rows.Next() {
		var (
			commandID   string
			command     string
			directory   string
			timestamp   int64
			exitCode    int
			embeddingJSON []byte
			metadata    string
			frequency   int
			lastUsed    int64
			successRate float32
			similarity  float32 // Additional column from vector search
		)

		err := rows.Scan(
			&commandID, &command, &directory, &timestamp, &exitCode,
			&embeddingJSON, &metadata, &frequency, &lastUsed, &successRate,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row with similarity: %v", err)
		}

		// Parse embedding
		var embedding []float32
		err = json.Unmarshal(embeddingJSON, &embedding)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling embedding: %v", err)
		}

		// Store similarity in metadata if not empty
		metadataMap := make(map[string]interface{})
		if metadata != "" {
			// Try to parse existing metadata
			if err := json.Unmarshal([]byte(metadata), &metadataMap); err != nil {
				// If it fails, just create a new map
				metadataMap = make(map[string]interface{})
			}
		}
		
		// Add similarity to metadata
		metadataMap["similarity"] = similarity
		
		// Marshal back to JSON
		metadataBytes, err := json.Marshal(metadataMap)
		if err != nil {
			return nil, fmt.Errorf("error marshaling metadata: %v", err)
		}

		commands = append(commands, CommandEmbedding{
			CommandID:   commandID,
			Command:     command,
			Directory:   directory,
			Timestamp:   time.Unix(timestamp, 0),
			ExitCode:    exitCode,
			Embedding:   embedding,
			Metadata:    string(metadataBytes),
			Frequency:   frequency,
			LastUsed:    time.Unix(lastUsed, 0),
			SuccessRate: successRate,
		})
	}

	return commands, nil
}

// GetStats returns statistics about the vector database
func (vm *VectorDBManager) GetStats() map[string]interface{} {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	stats := map[string]interface{}{
		"enabled":        vm.config.Enabled,
		"initialized":    vm.isInitialized,
		"db_path":        vm.config.DBPath,
		"dimension":      vm.config.EmbeddingDimension,
		"metric":         vm.config.DistanceMetric,
		"max_entries":    vm.config.MaxEntries,
		"index_interval": vm.config.IndexBuildInterval,
		"vector_count":   0,
	}

	if vm.isInitialized {
		// Get count of vectors
		var count int
		err := vm.db.QueryRow("SELECT COUNT(*) FROM command_embeddings").Scan(&count)
		if err == nil {
			stats["vector_count"] = count
		}

		// Get database size
		fileInfo, err := os.Stat(vm.config.DBPath)
		if err == nil {
			stats["db_size_bytes"] = fileInfo.Size()
			stats["db_size_mb"] = float64(fileInfo.Size()) / (1024 * 1024)
		}

		// Check if vectorx extension is available
		hasVectorX := vm.hasVectorExtension()
		stats["has_vector_extension"] = hasVectorX
		stats["last_index_build"] = vm.lastIndexBuild
		
		// Get vectorx extension version and details if available
		if hasVectorX {
			// Check which vectorx functions are available
			vectorxFunctions := []string{
				"vectorx_cosine_similarity",
				"vectorx_dot_product", 
				"vectorx_euclidean_distance",
				"vectorx_manhattan_distance",
				"vectorx_jaccard_similarity",
				"vectorx_version",
			}
			
			availableFunctions := make(map[string]bool)
			for _, fn := range vectorxFunctions {
				var result string
				err := vm.db.QueryRow("SELECT " + fn + "(json('[1]'), json('[1]'))").Scan(&result)
				availableFunctions[fn] = (err == nil)
			}
			
			stats["vectorx_functions"] = availableFunctions
			
			// Get vector index statistics
			var indexCount int
			err := vm.db.QueryRow("SELECT COUNT(*) FROM vector_index").Scan(&indexCount)
			if err == nil {
				stats["vector_index_count"] = indexCount
				stats["vector_index_sync"] = (indexCount == count)
			}
		}
	}

	return stats
}

// hasVectorExtension checks if the SQLite vector extension is available
func (vm *VectorDBManager) hasVectorExtension() bool {
	if !vm.isInitialized {
		return false
	}

	_, err := vm.db.Exec("SELECT * FROM vector_index LIMIT 1")
	return err == nil
}

// hasVectorFunction checks if a specific vectorx function is available
func (vm *VectorDBManager) hasVectorFunction(functionName string) bool {
	if !vm.isInitialized {
		return false
	}

	// Try to call the function with minimal arguments to see if it exists
	// This will fail if the function doesn't exist, but that's what we want to check
	queryString := fmt.Sprintf("SELECT %s(json('[1]'), json('[1]')) LIMIT 1", functionName)
	_, err := vm.db.Exec(queryString)
	return err == nil
}

// UpdateConfig updates the vector database configuration
func (vm *VectorDBManager) UpdateConfig(config VectorDBConfig) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	// Check if dimension changed
	dimensionChanged := vm.config.EmbeddingDimension != config.EmbeddingDimension

	vm.config = config
	err := vm.saveConfig()
	if err != nil {
		return err
	}

	// If dimension changed, we need to recreate the schema
	if dimensionChanged && vm.isInitialized {
		// Close the database
		vm.db.Close()
		vm.isInitialized = false

		// Reinitialize with new dimension
		return vm.Initialize()
	}

	return nil
}

// Close closes the vector database connection
func (vm *VectorDBManager) Close() error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	if vm.isInitialized && vm.db != nil {
		err := vm.db.Close()
		vm.isInitialized = false
		vm.db = nil
		return err
	}

	return nil
}

// Helper functions

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, magnitudeA, magnitudeB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		magnitudeA += a[i] * a[i]
		magnitudeB += b[i] * b[i]
	}

	if magnitudeA == 0 || magnitudeB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(magnitudeA))) * float32(math.Sqrt(float64(magnitudeB))))
}

// dotProduct calculates the dot product similarity between two vectors
func dotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProd float32
	for i := 0; i < len(a); i++ {
		dotProd += a[i] * b[i]
	}

	return dotProd
}

// euclideanDistance calculates the Euclidean distance between two vectors
// Lower values indicate higher similarity
func euclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}

	var sumSquared float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sumSquared += diff * diff
	}

	return float32(math.Sqrt(float64(sumSquared)))
}

// manhattanDistance calculates the Manhattan (L1) distance between two vectors
// Lower values indicate higher similarity
func manhattanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		sum += float32(math.Abs(float64(a[i] - b[i])))
	}

	return sum
}

// jaccardSimilarity calculates a variant of Jaccard similarity for float vectors
// by considering values above a threshold as "present"
// Higher values indicate higher similarity
func jaccardSimilarity(a, b []float32, threshold float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var intersection, union float32
	for i := 0; i < len(a); i++ {
		// Consider a component "present" if it's above the threshold
		aPresent := a[i] > threshold
		bPresent := b[i] > threshold
		
		if aPresent && bPresent {
			intersection++
		}
		
		if aPresent || bPresent {
			union++
		}
	}

	if union == 0 {
		return 0
	}

	return intersection / union
}

// ExportData exports all vector data to a JSON file
func (vm *VectorDBManager) ExportData(filePath string) error {
	if !vm.isInitialized {
		return fmt.Errorf("vector database not initialized")
	}

	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	// Get all command embeddings
	rows, err := vm.db.Query(`
		SELECT command_id, command, directory, timestamp, exit_code, 
			   embedding, metadata, frequency, last_used, success_rate
		FROM command_embeddings
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query command embeddings: %v", err)
	}
	defer rows.Close()

	// Scan all rows
	commands, err := vm.scanCommandRows(rows)
	if err != nil {
		return fmt.Errorf("failed to scan command rows: %v", err)
	}

	// Create export structure
	exportData := struct {
		Version           string            `json:"version"`
		ExportDate        time.Time         `json:"export_date"`
		Config            VectorDBConfig    `json:"config"`
		CommandEmbeddings []CommandEmbedding `json:"command_embeddings"`
	}{
		Version:           "1.0",
		ExportDate:        time.Now(),
		Config:            vm.config,
		CommandEmbeddings: commands,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %v", err)
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write export file: %v", err)
	}

	return nil
}

// ImportData imports vector data from a JSON file
func (vm *VectorDBManager) ImportData(filePath string, mergeStrategy string) error {
	if !vm.isInitialized {
		return fmt.Errorf("vector database not initialized")
	}

	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	// Read the import file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %v", err)
	}

	// Unmarshal JSON
	var importData struct {
		Version           string            `json:"version"`
		ExportDate        time.Time         `json:"export_date"`
		Config            VectorDBConfig    `json:"config"`
		CommandEmbeddings []CommandEmbedding `json:"command_embeddings"`
	}

	err = json.Unmarshal(data, &importData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal import data: %v", err)
	}

	// Check version compatibility
	if importData.Version != "1.0" {
		return fmt.Errorf("unsupported import version: %s", importData.Version)
	}

	// Begin transaction
	tx, err := vm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Handle different merge strategies
	switch mergeStrategy {
	case "replace":
		// Delete all existing data
		_, err = tx.Exec("DELETE FROM command_embeddings")
		if err != nil {
			return fmt.Errorf("failed to clear existing data: %v", err)
		}

		// Insert all imported commands
		stmt, err := tx.Prepare(`
			INSERT INTO command_embeddings 
				(command_id, command, directory, timestamp, exit_code, embedding, metadata, frequency, last_used, success_rate)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		for _, cmd := range importData.CommandEmbeddings {
			embeddingJSON, err := json.Marshal(cmd.Embedding)
			if err != nil {
				return fmt.Errorf("failed to marshal embedding: %v", err)
			}

			_, err = stmt.Exec(
				cmd.CommandID,
				cmd.Command,
				cmd.Directory,
				cmd.Timestamp.Unix(),
				cmd.ExitCode,
				embeddingJSON,
				cmd.Metadata,
				cmd.Frequency,
				cmd.LastUsed.Unix(),
				cmd.SuccessRate,
			)
			if err != nil {
				return fmt.Errorf("failed to insert command: %v", err)
			}
		}
	
	case "merge":
		// Merge strategy - add new commands, update existing ones
		stmt, err := tx.Prepare(`
			INSERT INTO command_embeddings 
				(command_id, command, directory, timestamp, exit_code, embedding, metadata, frequency, last_used, success_rate)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(command_id) DO UPDATE SET
				frequency = frequency + excluded.frequency,
				last_used = MAX(last_used, excluded.last_used),
				success_rate = (success_rate + excluded.success_rate) / 2
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare merge statement: %v", err)
		}
		defer stmt.Close()

		for _, cmd := range importData.CommandEmbeddings {
			embeddingJSON, err := json.Marshal(cmd.Embedding)
			if err != nil {
				return fmt.Errorf("failed to marshal embedding: %v", err)
			}

			_, err = stmt.Exec(
				cmd.CommandID,
				cmd.Command,
				cmd.Directory,
				cmd.Timestamp.Unix(),
				cmd.ExitCode,
				embeddingJSON,
				cmd.Metadata,
				cmd.Frequency,
				cmd.LastUsed.Unix(),
				cmd.SuccessRate,
			)
			if err != nil {
				return fmt.Errorf("failed to merge command: %v", err)
			}
		}

	case "keep_newer":
		// Keep newer strategy - only import commands that are newer than existing ones
		stmt, err := tx.Prepare(`
			INSERT INTO command_embeddings 
				(command_id, command, directory, timestamp, exit_code, embedding, metadata, frequency, last_used, success_rate)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(command_id) DO UPDATE SET
				command = excluded.command,
				directory = excluded.directory,
				timestamp = excluded.timestamp,
				exit_code = excluded.exit_code,
				embedding = excluded.embedding,
				metadata = excluded.metadata,
				frequency = excluded.frequency,
				last_used = excluded.last_used,
				success_rate = excluded.success_rate
			WHERE excluded.timestamp > timestamp
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare keep_newer statement: %v", err)
		}
		defer stmt.Close()

		for _, cmd := range importData.CommandEmbeddings {
			embeddingJSON, err := json.Marshal(cmd.Embedding)
			if err != nil {
				return fmt.Errorf("failed to marshal embedding: %v", err)
			}

			_, err = stmt.Exec(
				cmd.CommandID,
				cmd.Command,
				cmd.Directory,
				cmd.Timestamp.Unix(),
				cmd.ExitCode,
				embeddingJSON,
				cmd.Metadata,
				cmd.Frequency,
				cmd.LastUsed.Unix(),
				cmd.SuccessRate,
			)
			if err != nil {
				return fmt.Errorf("failed to import command with keep_newer strategy: %v", err)
			}
		}

	default:
		return fmt.Errorf("unsupported merge strategy: %s", mergeStrategy)
	}

	// Commit the transaction
	err = tx.Commit()
	tx = nil
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Rebuild the vector index
	_, err = vm.rebuildVectorIndex()
	if err != nil {
		return fmt.Errorf("failed to rebuild vector index: %v", err)
	}

	return nil
}

// Global VectorDBManager instance
var globalVectorDBManager *VectorDBManager

// GetVectorDBManager returns the global VectorDBManager instance
func GetVectorDBManager() *VectorDBManager {
	if globalVectorDBManager == nil {
		var err error
		globalVectorDBManager, err = NewVectorDBManager()
		if err != nil {
			fmt.Printf("Error initializing vector database manager: %v\n", err)
			return nil
		}
	}
	return globalVectorDBManager
}