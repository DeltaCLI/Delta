package main

import (
	"fmt"
	"sync"
	"time"
)

type OllamaHealthMonitor struct {
	aiManager            *AIPredictionManager
	checkInterval        time.Duration
	isRunning            bool
	stopChan             chan struct{}
	mu                   sync.Mutex
	lastStatus           bool
	lastCheckTime        time.Time
	notificationShown    bool
	consecutiveFailures  int
	enabled              bool
	notificationsEnabled bool
	isFirstCheck         bool
}

func NewOllamaHealthMonitor(aiManager *AIPredictionManager) *OllamaHealthMonitor {
	return &OllamaHealthMonitor{
		aiManager:            aiManager,
		checkInterval:        30 * time.Second, // Default to 30 seconds
		stopChan:             make(chan struct{}),
		enabled:              true,
		notificationsEnabled: true,
		isFirstCheck:         true,
	}
}

func (m *OllamaHealthMonitor) Start() {
	m.mu.Lock()
	if m.isRunning || !m.enabled {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()

	go m.monitorLoop()
}

func (m *OllamaHealthMonitor) Stop() {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = false
	m.mu.Unlock()

	close(m.stopChan)
}

func (m *OllamaHealthMonitor) monitorLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	// Initial check after a short delay
	time.Sleep(5 * time.Second)
	m.checkOllamaStatus()

	for {
		select {
		case <-ticker.C:
			m.checkOllamaStatus()
		case <-m.stopChan:
			return
		}
	}
}

func (m *OllamaHealthMonitor) checkOllamaStatus() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastCheckTime = time.Now()
	currentStatus := m.aiManager.ollamaClient.IsAvailable()

	// Only process if AI is currently disabled
	if !m.aiManager.IsEnabled() {
		// Check if Ollama just became available
		if currentStatus && !m.lastStatus {
			m.consecutiveFailures = 0
			m.showAvailabilityNotification()
		} else if !currentStatus {
			m.consecutiveFailures++
			// After many failures, reduce check frequency
			if m.consecutiveFailures > 10 && m.checkInterval < 5*time.Minute {
				m.checkInterval = time.Duration(float64(m.checkInterval) * 1.5)
			}
		}
	} else {
		// AI is enabled, monitor for disconnections
		if !currentStatus && m.lastStatus {
			// Ollama just went offline while AI was enabled
			fmt.Println("\033[33m⚠ Ollama server connection lost. AI features temporarily unavailable.\033[0m")
		} else if currentStatus && !m.lastStatus && !m.isFirstCheck {
			// Ollama came back online (but not on first check)
			fmt.Println("\033[32m✓ Ollama server connection restored.\033[0m")
			// Reinitialize AI if needed
			m.aiManager.Initialize()
		}
	}

	m.lastStatus = currentStatus
	
	// After the first check, set the flag to false
	if m.isFirstCheck {
		m.isFirstCheck = false
	}
}

func (m *OllamaHealthMonitor) showAvailabilityNotification() {
	// Check if notifications are enabled
	if !m.notificationsEnabled {
		return
	}

	// Don't show notification too frequently
	if m.notificationShown && time.Since(m.lastCheckTime) < 10*time.Minute {
		return
	}

	// Check if the model is also available
	modelAvailable := false
	if models, err := m.aiManager.ollamaClient.ListModels(); err == nil {
		for _, model := range models {
			if model.Name == m.aiManager.ollamaClient.ModelName {
				modelAvailable = true
				break
			}
		}
	}

	if modelAvailable {
		fmt.Printf("\n\033[32m💡 Ollama is now available! Enable AI features with ':ai enable' for intelligent command predictions.\033[0m\n")
		m.notificationShown = true
		// Reset check interval to default
		m.checkInterval = 30 * time.Second
	} else {
		// Ollama is running but model is not available
		fmt.Printf("\n\033[33m💡 Ollama is running but model '%s' is not installed. Run 'ollama pull %s' to enable AI features.\033[0m\n", 
			m.aiManager.ollamaClient.ModelName, m.aiManager.ollamaClient.ModelName)
		m.notificationShown = true
	}
}

func (m *OllamaHealthMonitor) SetCheckInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkInterval = interval
}

func (m *OllamaHealthMonitor) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notificationsEnabled = enabled
}

func (m *OllamaHealthMonitor) SetMonitorEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
	
	if !enabled && m.isRunning {
		m.isRunning = false
		close(m.stopChan)
	} else if enabled && !m.isRunning {
		m.stopChan = make(chan struct{})
		m.isRunning = true
		go m.monitorLoop()
	}
}

func (m *OllamaHealthMonitor) GetStatus() (isRunning bool, lastCheck time.Time, lastStatus bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunning, m.lastCheckTime, m.lastStatus
}