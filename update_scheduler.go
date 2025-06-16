package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// UpdateScheduler manages scheduled updates with cron-like functionality
type UpdateScheduler struct {
	mutex          sync.RWMutex
	scheduledTasks map[string]*ScheduledUpdate
	ticker         *time.Ticker
	running        bool
	updateManager  *UpdateManager
	i18nManager    *I18nManager
}

// ScheduledUpdate represents a scheduled update task
type ScheduledUpdate struct {
	ID              string            `json:"id"`
	Version         string            `json:"version"`
	ScheduledTime   time.Time         `json:"scheduled_time"`
	Status          ScheduleStatus    `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Metadata        map[string]string `json:"metadata"`
	RetryCount      int               `json:"retry_count"`
	MaxRetries      int               `json:"max_retries"`
	LastError       string            `json:"last_error,omitempty"`
	CronExpression  string            `json:"cron_expression,omitempty"`
	IsRecurring     bool              `json:"is_recurring"`
	AutoConfirm     bool              `json:"auto_confirm"`
}

// ScheduleStatus represents the status of a scheduled update
type ScheduleStatus string

const (
	ScheduleStatusPending   ScheduleStatus = "pending"
	ScheduleStatusRunning   ScheduleStatus = "running"
	ScheduleStatusCompleted ScheduleStatus = "completed"
	ScheduleStatusFailed    ScheduleStatus = "failed"
	ScheduleStatusCancelled ScheduleStatus = "cancelled"
)

// Global scheduler instance
var globalUpdateScheduler *UpdateScheduler
var schedulerOnce sync.Once

// GetUpdateScheduler returns the global UpdateScheduler instance
func GetUpdateScheduler() *UpdateScheduler {
	schedulerOnce.Do(func() {
		um := GetUpdateManager()
		if um != nil {
			globalUpdateScheduler = NewUpdateScheduler(um)
		}
	})
	return globalUpdateScheduler
}

// NewUpdateScheduler creates a new update scheduler instance
func NewUpdateScheduler(updateManager *UpdateManager) *UpdateScheduler {
	return &UpdateScheduler{
		scheduledTasks: make(map[string]*ScheduledUpdate),
		updateManager:  updateManager,
		i18nManager:    GetI18nManager(),
		running:        false,
	}
}

// Start begins the scheduler's background processing
func (us *UpdateScheduler) Start() error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	if us.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Start ticker to check for scheduled updates every minute
	us.ticker = time.NewTicker(1 * time.Minute)
	us.running = true

	go us.processScheduledUpdates()
	return nil
}

// Stop halts the scheduler's background processing
func (us *UpdateScheduler) Stop() error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	if !us.running {
		return fmt.Errorf("scheduler is not running")
	}

	if us.ticker != nil {
		us.ticker.Stop()
	}
	us.running = false
	return nil
}

// ScheduleUpdate schedules an update for a specific time
func (us *UpdateScheduler) ScheduleUpdate(version string, scheduledTime time.Time, options *ScheduleOptions) (*ScheduledUpdate, error) {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	if options == nil {
		options = &ScheduleOptions{
			AutoConfirm: false,
			MaxRetries:  3,
		}
	}

	// Generate unique ID
	id := fmt.Sprintf("update_%s_%d", version, time.Now().Unix())

	update := &ScheduledUpdate{
		ID:            id,
		Version:       version,
		ScheduledTime: scheduledTime,
		Status:        ScheduleStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Metadata:      make(map[string]string),
		RetryCount:    0,
		MaxRetries:    options.MaxRetries,
		AutoConfirm:   options.AutoConfirm,
		IsRecurring:   options.IsRecurring,
	}

	if options.CronExpression != "" {
		if err := us.validateCronExpression(options.CronExpression); err != nil {
			return nil, fmt.Errorf("invalid cron expression: %v", err)
		}
		update.CronExpression = options.CronExpression
		update.IsRecurring = true
	}

	us.scheduledTasks[id] = update
	return update, nil
}

// ScheduleOptions configures update scheduling behavior
type ScheduleOptions struct {
	AutoConfirm    bool
	MaxRetries     int
	CronExpression string
	IsRecurring    bool
	Metadata       map[string]string
}

// CancelScheduledUpdate cancels a scheduled update
func (us *UpdateScheduler) CancelScheduledUpdate(id string) error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	update, exists := us.scheduledTasks[id]
	if !exists {
		return fmt.Errorf("scheduled update with ID %s not found", id)
	}

	if update.Status == ScheduleStatusRunning {
		return fmt.Errorf("cannot cancel update that is currently running")
	}

	update.Status = ScheduleStatusCancelled
	update.UpdatedAt = time.Now()
	return nil
}

// GetScheduledUpdates returns all scheduled updates
func (us *UpdateScheduler) GetScheduledUpdates() []*ScheduledUpdate {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	updates := make([]*ScheduledUpdate, 0, len(us.scheduledTasks))
	for _, update := range us.scheduledTasks {
		updates = append(updates, update)
	}

	// Sort by scheduled time
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].ScheduledTime.Before(updates[j].ScheduledTime)
	})

	return updates
}

// GetPendingUpdates returns only pending scheduled updates
func (us *UpdateScheduler) GetPendingUpdates() []*ScheduledUpdate {
	updates := us.GetScheduledUpdates()
	var pending []*ScheduledUpdate

	for _, update := range updates {
		if update.Status == ScheduleStatusPending {
			pending = append(pending, update)
		}
	}

	return pending
}

// processScheduledUpdates runs in a goroutine to process scheduled updates
func (us *UpdateScheduler) processScheduledUpdates() {
	for {
		select {
		case <-us.ticker.C:
			us.checkAndExecuteUpdates()
		}

		// Check if scheduler should stop
		us.mutex.RLock()
		running := us.running
		us.mutex.RUnlock()

		if !running {
			break
		}
	}
}

// checkAndExecuteUpdates checks for due updates and executes them
func (us *UpdateScheduler) checkAndExecuteUpdates() {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	now := time.Now()

	for _, update := range us.scheduledTasks {
		if update.Status != ScheduleStatusPending {
			continue
		}

		// Check if it's time to execute
		if now.After(update.ScheduledTime) || now.Equal(update.ScheduledTime) {
			go us.executeScheduledUpdate(update)
		}
	}
}

// executeScheduledUpdate executes a single scheduled update
func (us *UpdateScheduler) executeScheduledUpdate(update *ScheduledUpdate) {
	us.mutex.Lock()
	update.Status = ScheduleStatusRunning
	update.UpdatedAt = time.Now()
	us.mutex.Unlock()

	fmt.Printf("⏰ Executing scheduled update to version %s...\n", update.Version)

	// Execute the update
	installResult, err := us.updateManager.DownloadAndInstallUpdate(update.Version)

	us.mutex.Lock()
	defer us.mutex.Unlock()

	if err != nil {
		update.Status = ScheduleStatusFailed
		update.LastError = err.Error()
		update.RetryCount++

		fmt.Printf("❌ Scheduled update failed: %v\n", err)

		// Check if we should retry
		if update.RetryCount < update.MaxRetries {
			// Schedule retry in 10 minutes
			update.ScheduledTime = time.Now().Add(10 * time.Minute)
			update.Status = ScheduleStatusPending
			fmt.Printf("🔄 Retrying in 10 minutes (attempt %d/%d)\n", update.RetryCount+1, update.MaxRetries)
		} else {
			fmt.Printf("❌ Max retries exceeded for scheduled update %s\n", update.ID)
		}
	} else {
		update.Status = ScheduleStatusCompleted
		update.LastError = ""
		fmt.Printf("✅ Scheduled update completed successfully!\n")
		
		if installResult != nil {
			fmt.Printf("   Old Version: %s\n", installResult.OldVersion)
			fmt.Printf("   New Version: %s\n", installResult.NewVersion)
		}

		// Handle recurring updates
		if update.IsRecurring && update.CronExpression != "" {
			nextTime := us.calculateNextCronTime(update.CronExpression, time.Now())
			if nextTime != nil {
				// Create new scheduled update for next occurrence
				newUpdate := &ScheduledUpdate{
					ID:             fmt.Sprintf("update_%s_%d", update.Version, time.Now().Unix()),
					Version:        update.Version,
					ScheduledTime:  *nextTime,
					Status:         ScheduleStatusPending,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
					Metadata:       update.Metadata,
					RetryCount:     0,
					MaxRetries:     update.MaxRetries,
					AutoConfirm:    update.AutoConfirm,
					CronExpression: update.CronExpression,
					IsRecurring:    true,
				}
				us.scheduledTasks[newUpdate.ID] = newUpdate
			}
		}
	}

	update.UpdatedAt = time.Now()
}

// validateCronExpression validates a cron expression (simplified validation)
func (us *UpdateScheduler) validateCronExpression(expr string) error {
	// Simple validation for now - in a real implementation you'd use a proper cron parser
	if expr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}
	
	// TODO: Implement proper cron expression validation
	// For now, accept common patterns
	validPatterns := []string{
		"@daily", "@weekly", "@monthly", "@yearly",
		"0 0 * * *", "0 0 * * 0", "0 0 1 * *", "0 0 1 1 *",
	}
	
	for _, pattern := range validPatterns {
		if expr == pattern {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported cron expression: %s", expr)
}

// calculateNextCronTime calculates the next execution time for a cron expression
func (us *UpdateScheduler) calculateNextCronTime(expr string, from time.Time) *time.Time {
	// Simplified cron calculation
	switch expr {
	case "@daily", "0 0 * * *":
		next := time.Date(from.Year(), from.Month(), from.Day()+1, 0, 0, 0, 0, from.Location())
		return &next
	case "@weekly", "0 0 * * 0":
		daysUntilSunday := (7 - int(from.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		next := time.Date(from.Year(), from.Month(), from.Day()+daysUntilSunday, 0, 0, 0, 0, from.Location())
		return &next
	case "@monthly", "0 0 1 * *":
		next := time.Date(from.Year(), from.Month()+1, 1, 0, 0, 0, 0, from.Location())
		return &next
	case "@yearly", "0 0 1 1 *":
		next := time.Date(from.Year()+1, 1, 1, 0, 0, 0, 0, from.Location())
		return &next
	default:
		return nil
	}
}

// CleanupCompletedTasks removes old completed/failed tasks
func (us *UpdateScheduler) CleanupCompletedTasks(olderThan time.Duration) int {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	for id, update := range us.scheduledTasks {
		if (update.Status == ScheduleStatusCompleted || update.Status == ScheduleStatusFailed || update.Status == ScheduleStatusCancelled) &&
			update.UpdatedAt.Before(cutoff) &&
			!update.IsRecurring {
			delete(us.scheduledTasks, id)
			removed++
		}
	}

	return removed
}

// IsRunning returns whether the scheduler is currently running
func (us *UpdateScheduler) IsRunning() bool {
	us.mutex.RLock()
	defer us.mutex.RUnlock()
	return us.running
}

// GetSchedulerStats returns statistics about the scheduler
func (us *UpdateScheduler) GetSchedulerStats() map[string]interface{} {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["running"] = us.running
	stats["total_tasks"] = len(us.scheduledTasks)

	// Count by status
	statusCounts := make(map[ScheduleStatus]int)
	for _, update := range us.scheduledTasks {
		statusCounts[update.Status]++
	}

	stats["pending"] = statusCounts[ScheduleStatusPending]
	stats["running"] = statusCounts[ScheduleStatusRunning]
	stats["completed"] = statusCounts[ScheduleStatusCompleted]
	stats["failed"] = statusCounts[ScheduleStatusFailed]
	stats["cancelled"] = statusCounts[ScheduleStatusCancelled]

	return stats
}