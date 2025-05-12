package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	DistanceMetric     string   `json:"distance_metric"` // "cosine" or "dot" or "euclidean"
	MaxEntries         int      `json:"max_entries"`
	IndexBuildInterval int      `json:"index_build_interval"` // in minutes
	CommandTypes       []string `json:"command_types"`       // Specific types of commands to embed
	InMemoryMode       bool     `json:"in_memory_mode"`     // Whether to keep vectors in memory
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

	// Create a virtual table for vector search if SQLite has the vector extension
	// Note: This requires SQLite with vector extension to be installed
	_, err = vm.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS vector_index USING vectorx(
			embedding(${dimension})
		)
	`)
	
	// If virtual table creation fails, it might be because the vector extension is not available
	// We will fall back to a standard implementation that does vector distance calculation in Go

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

	// Check if the vector extension is available
	_, err := vm.db.Exec("SELECT * FROM vector_index LIMIT 1")
	
	if err == nil {
		// Vector extension is available, rebuild the index
		_, err = vm.db.Exec("DELETE FROM vector_index")
		if err != nil {
			fmt.Printf("Error clearing vector index: %v\n", err)
			return
		}

		// Insert embeddings into vector index
		_, err = vm.db.Exec(`
			INSERT INTO vector_index (rowid, embedding)
			SELECT rowid, embedding FROM command_embeddings
		`)
		if err != nil {
			fmt.Printf("Error rebuilding vector index: %v\n", err)
			return
		}
	}

	vm.lastIndexBuild = time.Now()
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
	_, err := vm.db.Exec("SELECT * FROM vector_index LIMIT 1")
	
	if err == nil {
		// Vector extension is available, use it for search
		// Convert query to JSON
		queryJSON, err := json.Marshal(query)
		if err != nil {
			return nil, err
		}

		// Prepare context filter if provided
		contextFilter := ""
		args := []interface{}{queryJSON, limit}
		
		if context != "" {
			contextFilter = "AND directory LIKE ?"
			args = append(args, "%"+context+"%")
		}

		// Query using vector index
		rows, err := vm.db.Query(`
			SELECT ce.command_id, ce.command, ce.directory, ce.timestamp, ce.exit_code, 
				   ce.embedding, ce.metadata, ce.frequency, ce.last_used, ce.success_rate
			FROM command_embeddings ce
			JOIN vector_index vi ON ce.rowid = vi.rowid
			WHERE `+contextFilter+`
			ORDER BY vi.embedding <=> ?
			LIMIT ?
		`, args...)

		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return vm.scanCommandRows(rows)
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
	distances := make([]struct {
		cmd      CommandEmbedding
		distance float32
	}, len(allCommands))

	for i, cmd := range allCommands {
		distances[i].cmd = cmd
		distances[i].distance = cosineSimilarity(query, cmd.Embedding)
	}

	// Sort by distance
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].distance > distances[j].distance
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

		// Get most frequent commands
		stats["has_vector_extension"] = vm.hasVectorExtension()
		stats["last_index_build"] = vm.lastIndexBuild
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