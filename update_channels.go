package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UpdateChannel represents a release channel for updates
type UpdateChannel string

const (
	// ChannelStable represents the stable release channel
	ChannelStable UpdateChannel = "stable"
	// ChannelBeta represents the beta release channel
	ChannelBeta UpdateChannel = "beta"
	// ChannelAlpha represents the alpha release channel
	ChannelAlpha UpdateChannel = "alpha"
	// ChannelNightly represents the nightly build channel
	ChannelNightly UpdateChannel = "nightly"
	// ChannelCustom represents a custom/enterprise channel
	ChannelCustom UpdateChannel = "custom"
)

// ChannelPolicy defines the policy for a specific channel
type ChannelPolicy struct {
	Name               UpdateChannel `json:"name"`
	Description        string        `json:"description"`
	AllowDowngrade     bool          `json:"allow_downgrade"`
	AllowPrerelease    bool          `json:"allow_prerelease"`
	AutoInstall        bool          `json:"auto_install"`
	RequireApproval    bool          `json:"require_approval"`
	MinVersionRequired string        `json:"min_version_required"`
	MaxVersionAllowed  string        `json:"max_version_allowed"`
	UpdateFrequency    string        `json:"update_frequency"` // "immediate", "daily", "weekly", "monthly"
	AllowedUsers       []string      `json:"allowed_users"`     // Empty means all users
	AllowedGroups      []string      `json:"allowed_groups"`    // Empty means all groups
	RestrictedRegions  []string      `json:"restricted_regions"` // Empty means no restrictions
	CustomURL          string        `json:"custom_url"`         // For custom/enterprise channels
	VerificationKey    string        `json:"verification_key"`   // Public key for signature verification
}

// ChannelAccess defines access control for channels
type ChannelAccess struct {
	UserID        string    `json:"user_id"`
	AllowedChannels []UpdateChannel `json:"allowed_channels"`
	ForcedChannel  *UpdateChannel  `json:"forced_channel,omitempty"`
	Restrictions   []string        `json:"restrictions"`
	LastModified   time.Time      `json:"last_modified"`
	ModifiedBy     string         `json:"modified_by"`
}

// ChannelMigration represents a channel migration operation
type ChannelMigration struct {
	ID            string        `json:"id"`
	FromChannel   UpdateChannel `json:"from_channel"`
	ToChannel     UpdateChannel `json:"to_channel"`
	ScheduledTime time.Time     `json:"scheduled_time"`
	Status        string        `json:"status"` // "pending", "in_progress", "completed", "failed"
	Reason        string        `json:"reason"`
	AffectedUsers []string      `json:"affected_users,omitempty"`
	StartedAt     *time.Time    `json:"started_at,omitempty"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	Error         string        `json:"error,omitempty"`
}

// ChannelManager manages update channels and policies
type ChannelManager struct {
	configPath      string
	policiesPath    string
	accessPath      string
	migrationsPath  string
	currentChannel  UpdateChannel
	policies        map[UpdateChannel]*ChannelPolicy
	userAccess      map[string]*ChannelAccess
	migrations      []ChannelMigration
	isEnterprise    bool
	mutex           sync.RWMutex
}

// ChannelConfig represents the channel configuration
type ChannelConfig struct {
	CurrentChannel   UpdateChannel             `json:"current_channel"`
	ChannelHistory   []ChannelChangeEntry      `json:"channel_history"`
	EnterpriseMode   bool                      `json:"enterprise_mode"`
	PolicyOverrides  map[string]interface{}    `json:"policy_overrides,omitempty"`
	LastCheck        time.Time                 `json:"last_check"`
	NextScheduledCheck time.Time               `json:"next_scheduled_check"`
}

// ChannelChangeEntry represents a channel change history entry
type ChannelChangeEntry struct {
	FromChannel UpdateChannel `json:"from_channel"`
	ToChannel   UpdateChannel `json:"to_channel"`
	Timestamp   time.Time     `json:"timestamp"`
	Reason      string        `json:"reason"`
	ChangedBy   string        `json:"changed_by"`
}

// NewChannelManager creates a new channel manager
func NewChannelManager() (*ChannelManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".config", "delta", "update")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create update config directory: %v", err)
	}

	cm := &ChannelManager{
		configPath:     filepath.Join(configDir, "channel_config.json"),
		policiesPath:   filepath.Join(configDir, "channel_policies.json"),
		accessPath:     filepath.Join(configDir, "channel_access.json"),
		migrationsPath: filepath.Join(configDir, "channel_migrations.json"),
		currentChannel: ChannelStable, // Default to stable
		policies:       make(map[UpdateChannel]*ChannelPolicy),
		userAccess:     make(map[string]*ChannelAccess),
		migrations:     make([]ChannelMigration, 0),
		isEnterprise:   false,
	}

	// Initialize default policies
	cm.initializeDefaultPolicies()

	// Load configurations
	if err := cm.loadConfig(); err != nil {
		// If loading fails, save default config
		cm.saveConfig()
	}

	if err := cm.loadPolicies(); err != nil {
		// Save default policies if loading fails
		cm.savePolicies()
	}

	if err := cm.loadAccess(); err != nil {
		// Initialize empty access if loading fails
		cm.saveAccess()
	}

	if err := cm.loadMigrations(); err != nil {
		// Initialize empty migrations if loading fails
		cm.saveMigrations()
	}

	return cm, nil
}

// initializeDefaultPolicies sets up the default channel policies
func (cm *ChannelManager) initializeDefaultPolicies() {
	cm.policies[ChannelStable] = &ChannelPolicy{
		Name:            ChannelStable,
		Description:     "Stable releases recommended for production use",
		AllowDowngrade:  false,
		AllowPrerelease: false,
		AutoInstall:     false,
		RequireApproval: false,
		UpdateFrequency: "weekly",
	}

	cm.policies[ChannelBeta] = &ChannelPolicy{
		Name:            ChannelBeta,
		Description:     "Beta releases for testing new features",
		AllowDowngrade:  true,
		AllowPrerelease: true,
		AutoInstall:     false,
		RequireApproval: false,
		UpdateFrequency: "daily",
	}

	cm.policies[ChannelAlpha] = &ChannelPolicy{
		Name:            ChannelAlpha,
		Description:     "Alpha releases for early adopters",
		AllowDowngrade:  true,
		AllowPrerelease: true,
		AutoInstall:     false,
		RequireApproval: false,
		UpdateFrequency: "immediate",
	}

	cm.policies[ChannelNightly] = &ChannelPolicy{
		Name:            ChannelNightly,
		Description:     "Nightly builds with latest changes",
		AllowDowngrade:  true,
		AllowPrerelease: true,
		AutoInstall:     true,
		RequireApproval: false,
		UpdateFrequency: "immediate",
	}
}

// GetCurrentChannel returns the current update channel
func (cm *ChannelManager) GetCurrentChannel() UpdateChannel {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.currentChannel
}

// SetChannel changes the current update channel
func (cm *ChannelManager) SetChannel(channel UpdateChannel, reason string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if channel exists
	if _, exists := cm.policies[channel]; !exists && channel != ChannelCustom {
		return fmt.Errorf("unknown channel: %s", channel)
	}

	// Check access permissions
	if err := cm.checkChannelAccess(channel); err != nil {
		return fmt.Errorf("access denied: %v", err)
	}

	// Record channel change
	oldChannel := cm.currentChannel
	cm.currentChannel = channel

	// Add to history
	if cm.configPath != "" {
		config, _ := cm.loadConfigFile()
		if config.ChannelHistory == nil {
			config.ChannelHistory = make([]ChannelChangeEntry, 0)
		}

		config.ChannelHistory = append(config.ChannelHistory, ChannelChangeEntry{
			FromChannel: oldChannel,
			ToChannel:   channel,
			Timestamp:   time.Now(),
			Reason:      reason,
			ChangedBy:   cm.getCurrentUser(),
		})

		config.CurrentChannel = channel
		cm.saveConfigFile(config)
	}

	// Record channel switch metrics
	if metrics := GetUpdateMetrics(); metrics != nil {
		metrics.RecordChannelSwitch(string(oldChannel), string(channel), reason)
	}

	return nil
}

// GetChannelPolicy returns the policy for a specific channel
func (cm *ChannelManager) GetChannelPolicy(channel UpdateChannel) (*ChannelPolicy, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	policy, exists := cm.policies[channel]
	if !exists {
		return nil, fmt.Errorf("no policy found for channel: %s", channel)
	}

	return policy, nil
}

// SetChannelPolicy updates the policy for a channel (enterprise only)
func (cm *ChannelManager) SetChannelPolicy(channel UpdateChannel, policy *ChannelPolicy) error {
	if !cm.isEnterprise {
		return fmt.Errorf("channel policy modification requires enterprise mode")
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.policies[channel] = policy
	return cm.savePolicies()
}

// checkChannelAccess verifies if the current user can access the channel
func (cm *ChannelManager) checkChannelAccess(channel UpdateChannel) error {
	if !cm.isEnterprise {
		// In non-enterprise mode, all channels are accessible
		return nil
	}

	userID := cm.getCurrentUser()
	access, exists := cm.userAccess[userID]
	if !exists {
		// No specific access rules means default access
		return nil
	}

	// Check forced channel
	if access.ForcedChannel != nil && *access.ForcedChannel != channel {
		return fmt.Errorf("user is restricted to channel: %s", *access.ForcedChannel)
	}

	// Check allowed channels
	if len(access.AllowedChannels) > 0 {
		allowed := false
		for _, ch := range access.AllowedChannels {
			if ch == channel {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("user not allowed to use channel: %s", channel)
		}
	}

	return nil
}

// GetAvailableChannels returns channels available to the current user
func (cm *ChannelManager) GetAvailableChannels() []UpdateChannel {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.isEnterprise {
		// Return all standard channels
		return []UpdateChannel{ChannelStable, ChannelBeta, ChannelAlpha, ChannelNightly}
	}

	userID := cm.getCurrentUser()
	access, exists := cm.userAccess[userID]
	if !exists || len(access.AllowedChannels) == 0 {
		// No restrictions, return all channels
		channels := make([]UpdateChannel, 0, len(cm.policies))
		for ch := range cm.policies {
			channels = append(channels, ch)
		}
		return channels
	}

	// Return only allowed channels
	return access.AllowedChannels
}

// ScheduleMigration schedules a channel migration
func (cm *ChannelManager) ScheduleMigration(from, to UpdateChannel, scheduledTime time.Time, reason string, affectedUsers []string) (*ChannelMigration, error) {
	if !cm.isEnterprise {
		return nil, fmt.Errorf("channel migration requires enterprise mode")
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	migration := ChannelMigration{
		ID:            fmt.Sprintf("mig_%d", time.Now().Unix()),
		FromChannel:   from,
		ToChannel:     to,
		ScheduledTime: scheduledTime,
		Status:        "pending",
		Reason:        reason,
		AffectedUsers: affectedUsers,
	}

	cm.migrations = append(cm.migrations, migration)
	
	if err := cm.saveMigrations(); err != nil {
		return nil, err
	}

	return &migration, nil
}

// ProcessPendingMigrations processes any pending channel migrations
func (cm *ChannelManager) ProcessPendingMigrations() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	updated := false

	for i := range cm.migrations {
		migration := &cm.migrations[i]
		
		if migration.Status == "pending" && now.After(migration.ScheduledTime) {
			// Check if current user is affected
			if cm.isUserAffected(migration) {
				// Execute migration
				migration.Status = "in_progress"
				migration.StartedAt = &now
				updated = true

				// Perform the channel switch
				if err := cm.SetChannel(migration.ToChannel, fmt.Sprintf("Migration: %s", migration.Reason)); err != nil {
					migration.Status = "failed"
					migration.Error = err.Error()
				} else {
					migration.Status = "completed"
					completedTime := time.Now()
					migration.CompletedAt = &completedTime
				}
			}
		}
	}

	if updated {
		return cm.saveMigrations()
	}

	return nil
}

// Helper methods

func (cm *ChannelManager) getCurrentUser() string {
	// Get current user ID
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "unknown"
}

func (cm *ChannelManager) isUserAffected(migration *ChannelMigration) bool {
	if len(migration.AffectedUsers) == 0 {
		// No specific users means all users are affected
		return true
	}

	currentUser := cm.getCurrentUser()
	for _, user := range migration.AffectedUsers {
		if user == currentUser || user == "*" {
			return true
		}
	}

	return false
}

// Configuration persistence methods

func (cm *ChannelManager) loadConfig() error {
	config, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	cm.currentChannel = config.CurrentChannel
	cm.isEnterprise = config.EnterpriseMode

	return nil
}

func (cm *ChannelManager) loadConfigFile() (*ChannelConfig, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return &ChannelConfig{CurrentChannel: ChannelStable}, err
	}

	var config ChannelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return &ChannelConfig{CurrentChannel: ChannelStable}, err
	}

	return &config, nil
}

func (cm *ChannelManager) saveConfig() error {
	config := &ChannelConfig{
		CurrentChannel: cm.currentChannel,
		EnterpriseMode: cm.isEnterprise,
		LastCheck:      time.Now(),
	}

	return cm.saveConfigFile(config)
}

func (cm *ChannelManager) saveConfigFile(config *ChannelConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configPath, data, 0644)
}

func (cm *ChannelManager) loadPolicies() error {
	data, err := os.ReadFile(cm.policiesPath)
	if err != nil {
		return err
	}

	var policies map[UpdateChannel]*ChannelPolicy
	if err := json.Unmarshal(data, &policies); err != nil {
		return err
	}

	cm.policies = policies
	return nil
}

func (cm *ChannelManager) savePolicies() error {
	data, err := json.MarshalIndent(cm.policies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.policiesPath, data, 0644)
}

func (cm *ChannelManager) loadAccess() error {
	data, err := os.ReadFile(cm.accessPath)
	if err != nil {
		return err
	}

	var access map[string]*ChannelAccess
	if err := json.Unmarshal(data, &access); err != nil {
		return err
	}

	cm.userAccess = access
	return nil
}

func (cm *ChannelManager) saveAccess() error {
	data, err := json.MarshalIndent(cm.userAccess, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.accessPath, data, 0644)
}

func (cm *ChannelManager) loadMigrations() error {
	data, err := os.ReadFile(cm.migrationsPath)
	if err != nil {
		return err
	}

	var migrations []ChannelMigration
	if err := json.Unmarshal(data, &migrations); err != nil {
		return err
	}

	cm.migrations = migrations
	return nil
}

func (cm *ChannelManager) saveMigrations() error {
	data, err := json.MarshalIndent(cm.migrations, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.migrationsPath, data, 0644)
}

// SetEnterpriseMode enables or disables enterprise mode
func (cm *ChannelManager) SetEnterpriseMode(enabled bool) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.isEnterprise = enabled
	return cm.saveConfig()
}

// SetUserAccess sets access control for a specific user (enterprise only)
func (cm *ChannelManager) SetUserAccess(userID string, access *ChannelAccess) error {
	if !cm.isEnterprise {
		return fmt.Errorf("user access control requires enterprise mode")
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	access.LastModified = time.Now()
	access.ModifiedBy = cm.getCurrentUser()
	cm.userAccess[userID] = access

	return cm.saveAccess()
}

// GetChannelStats returns statistics about channel usage
func (cm *ChannelManager) GetChannelStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["current_channel"] = cm.currentChannel
	stats["enterprise_mode"] = cm.isEnterprise
	stats["total_policies"] = len(cm.policies)
	stats["total_access_rules"] = len(cm.userAccess)
	
	// Count migrations by status
	migrationStats := make(map[string]int)
	for _, mig := range cm.migrations {
		migrationStats[mig.Status]++
	}
	stats["migrations"] = migrationStats

	// Get channel history count
	if config, err := cm.loadConfigFile(); err == nil {
		stats["channel_changes"] = len(config.ChannelHistory)
	}

	return stats
}

// ValidateChannelName validates if a channel name is valid
func ValidateChannelName(channel string) bool {
	validChannels := []string{
		string(ChannelStable),
		string(ChannelBeta),
		string(ChannelAlpha),
		string(ChannelNightly),
		string(ChannelCustom),
	}

	for _, valid := range validChannels {
		if channel == valid {
			return true
		}
	}

	// Allow custom channel names with specific format
	if strings.HasPrefix(channel, "custom-") {
		return true
	}

	return false
}

// Global channel manager instance
var globalChannelManager *ChannelManager
var channelManagerOnce sync.Once

// GetChannelManager returns the global channel manager instance
func GetChannelManager() *ChannelManager {
	channelManagerOnce.Do(func() {
		var err error
		globalChannelManager, err = NewChannelManager()
		if err != nil {
			fmt.Printf("Warning: failed to initialize channel manager: %v\n", err)
		}
	})
	return globalChannelManager
}