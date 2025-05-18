package main

import (
	"fmt"
	"strings"
	"time"
)

// HandleAgentCommand processes agent-related commands
func HandleAgentCommand(args []string) bool {
	// Get the AgentManager instance
	am := GetAgentManager()
	if am == nil {
		fmt.Println("Failed to initialize agent manager")
		return true
	}

	// Process commands
	if len(args) == 0 {
		// Show agent status
		showAgentStatus(am)
		return true
	}

	// Handle subcommands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "enable":
			// Enable agent manager
			err := am.Initialize()
			if err != nil {
				fmt.Printf("Error initializing agent manager: %v\n", err)
				return true
			}

			err = am.Enable()
			if err != nil {
				fmt.Printf("Error enabling agent manager: %v\n", err)
			} else {
				fmt.Println("Agent manager enabled")
			}
			return true

		case "disable":
			// Disable agent manager
			err := am.Disable()
			if err != nil {
				fmt.Printf("Error disabling agent manager: %v\n", err)
			} else {
				fmt.Println("Agent manager disabled")
			}
			return true

		case "status":
			// Show status
			showAgentStatus(am)
			return true

		case "list":
			// List agents
			listAgents(am)
			return true

		case "show":
			// Show agent details
			if len(args) < 2 {
				fmt.Println("Usage: :agent show <agent_id>")
				return true
			}
			showAgent(am, args[1])
			return true

		case "run":
			// Run an agent
			if len(args) < 2 {
				fmt.Println("Usage: :agent run <agent_id> [--options]")
				return true
			}

			// Parse options
			options := parseAgentOptions(args[2:])
			runAgent(am, args[1], options)
			return true

		case "create":
			// Create a new agent
			if len(args) < 2 {
				fmt.Println("Usage: :agent create <name> [--template=<template>]")
				return true
			}

			// Parse options
			options := parseAgentOptions(args[2:])
			createAgent(am, args[1], options)
			return true

		case "edit":
			// Edit an agent
			if len(args) < 2 {
				fmt.Println("Usage: :agent edit <agent_id>")
				return true
			}
			editAgent(am, args[1])
			return true

		case "delete":
			// Delete an agent
			if len(args) < 2 {
				fmt.Println("Usage: :agent delete <agent_id>")
				return true
			}
			deleteAgent(am, args[1])
			return true

		case "learn":
			// Learn a new agent from command sequence
			if len(args) < 2 {
				fmt.Println("Usage: :agent learn <command_sequence>")
				return true
			}
			learnAgent(am, strings.Join(args[1:], " "))
			return true

		case "docker":
			// Docker-related commands
			if len(args) < 2 {
				fmt.Println("Usage: :agent docker <subcommand>")
				return true
			}
			handleDockerCommands(am, args[1:])
			return true

		case "stats":
			// Show statistics
			showAgentStats(am)
			return true

		case "help":
			// Show help
			showAgentHelp()
			return true
			
		case "errors":
			// Handle error solution commands
			handleErrorCommands(am, args[1:])
			return true

		default:
			fmt.Printf("Unknown agent command: %s\n", cmd)
			fmt.Println("Type :agent help for a list of available commands")
			return true
		}
	}

	return true
}

// showAgentStatus displays current status of the agent manager
func showAgentStatus(am *AgentManager) {
	fmt.Println("Agent Manager Status")
	fmt.Println("===================")

	if am.IsEnabled() {
		fmt.Println("Status: Enabled and active")

		// Show agent counts
		agents := am.ListAgents()
		enabledCount := 0
		for _, agent := range agents {
			if agent.Enabled {
				enabledCount++
			}
		}

		fmt.Printf("Agents: %d total, %d enabled\n", len(agents), enabledCount)

		// Show Docker status
		if am.config.UseDockerBuilds {
			fmt.Println("Docker Builds: Enabled")
			// Check docker availability
			err := checkDockerAvailability()
			if err != nil {
				fmt.Printf("Docker Status: Not available (%v)\n", err)
			} else {
				fmt.Println("Docker Status: Available")
			}
		} else {
			fmt.Println("Docker Builds: Disabled")
		}

		// Show AI assistance status
		if am.config.UseAIAssistance {
			fmt.Println("AI Assistance: Enabled")
			if am.aiManager != nil && am.aiManager.IsEnabled() {
				fmt.Printf("AI Status: Available (using %s)\n", am.aiManager.ollamaClient.ModelName)
			} else {
				fmt.Println("AI Status: Not available")
			}
		} else {
			fmt.Println("AI Assistance: Disabled")
		}
	} else {
		fmt.Println("Status: Disabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
	}
}

// listAgents displays a list of all agents
func listAgents(am *AgentManager) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	agents, err := am.ListAgents()
	if err != nil {
		fmt.Printf("Error listing agents: %v\n", err)
		return
	}
	
	if len(agents) == 0 {
		fmt.Println("No agents found")
		fmt.Println("Use ':agent create <name>' to create a new agent")
		return
	}

	fmt.Println("Available Agents")
	fmt.Println("===============")
	
	for _, agent := range agents {
		status := "Enabled"
		if !agent.Enabled {
			status = "Disabled"
		}
		
		fmt.Printf("%s (%s) - %s [%s]\n", agent.Name, agent.ID, agent.Description, status)
		fmt.Printf("  Types: %s\n", strings.Join(agent.TaskTypes, ", "))
		if agent.RunCount > 0 {
			fmt.Printf("  Runs: %d (%.1f%% success rate)\n", agent.RunCount, agent.SuccessRate*100)
		}
		fmt.Println()
	}
}

// showAgent displays detailed information about an agent
func showAgent(am *AgentManager, agentID string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Agent: %s (%s)\n", agent.Name, agent.ID)
	fmt.Println("==========================")
	fmt.Printf("Description: %s\n", agent.Description)
	fmt.Printf("Status: %s\n", getBoolText(agent.Enabled, "Enabled", "Disabled"))
	fmt.Printf("Task Types: %s\n", strings.Join(agent.TaskTypes, ", "))
	
	if len(agent.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(agent.Tags, ", "))
	}
	
	fmt.Printf("\nCreated: %s\n", agent.CreatedAt.Format(time.RFC1123))
	fmt.Printf("Updated: %s\n", agent.UpdatedAt.Format(time.RFC1123))
	
	if !agent.LastRunAt.IsZero() {
		fmt.Printf("Last Run: %s\n", agent.LastRunAt.Format(time.RFC1123))
	}
	
	if agent.RunCount > 0 {
		fmt.Printf("Run Count: %d\n", agent.RunCount)
		fmt.Printf("Success Rate: %.1f%%\n", agent.SuccessRate*100)
	}
	
	// Show commands
	fmt.Println("\nCommands:")
	for i, cmd := range agent.Commands {
		fmt.Printf("%d. %s\n", i+1, cmd.Command)
		fmt.Printf("   Working Directory: %s\n", cmd.WorkingDir)
		fmt.Printf("   Timeout: %d seconds\n", cmd.Timeout)
		if cmd.RetryCount > 0 {
			fmt.Printf("   Retry: %d times (delay: %d seconds)\n", cmd.RetryCount, cmd.RetryDelay)
		}
		if len(cmd.SuccessPatterns) > 0 {
			fmt.Printf("   Success Patterns: %s\n", strings.Join(cmd.SuccessPatterns, ", "))
		}
		if len(cmd.ErrorPatterns) > 0 {
			fmt.Printf("   Error Patterns: %s\n", strings.Join(cmd.ErrorPatterns, ", "))
		}
		fmt.Println()
	}
	
	// Show Docker config if present
	if agent.DockerConfig != nil {
		fmt.Println("Docker Configuration:")
		fmt.Printf("   Image: %s:%s\n", agent.DockerConfig.Image, agent.DockerConfig.Tag)
		if agent.DockerConfig.BuildContext != "" {
			fmt.Printf("   Build Context: %s\n", agent.DockerConfig.BuildContext)
		}
		if agent.DockerConfig.Dockerfile != "" {
			fmt.Printf("   Dockerfile: %s\n", agent.DockerConfig.Dockerfile)
		}
		if len(agent.DockerConfig.Volumes) > 0 {
			fmt.Printf("   Volumes: %s\n", strings.Join(agent.DockerConfig.Volumes, ", "))
		}
		fmt.Printf("   Use Cache: %s\n", getBoolText(agent.DockerConfig.UseCache, "Yes", "No"))
		fmt.Println()
	}
	
	// Show trigger patterns
	if len(agent.TriggerPatterns) > 0 {
		fmt.Println("Trigger Patterns:")
		for _, pattern := range agent.TriggerPatterns {
			fmt.Printf("   - %s\n", pattern)
		}
		fmt.Println()
	}
	
	// Show context
	if len(agent.Context) > 0 {
		fmt.Println("Context:")
		for k, v := range agent.Context {
			fmt.Printf("   %s: %s\n", k, v)
		}
		fmt.Println()
	}
	
	// Show AI prompt
	if agent.AIPrompt != "" {
		fmt.Println("AI Prompt:")
		fmt.Printf("   %s\n", agent.AIPrompt)
	}
	
	// Show recent run history
	history := am.GetRunHistory(agent.ID, 3)
	if len(history) > 0 {
		fmt.Println("\nRecent Runs:")
		for i, run := range history {
			fmt.Printf("%d. %s\n", i+1, run.StartTime.Format(time.RFC1123))
			fmt.Printf("   Result: %s\n", getBoolText(run.Success, "Success", "Failed"))
			fmt.Printf("   Duration: %.1f seconds\n", run.EndTime.Sub(run.StartTime).Seconds())
			fmt.Printf("   Commands Run: %d\n", run.CommandsRun)
			fmt.Println()
		}
	}
}

// runAgent runs an agent with optional parameters
func runAgent(am *AgentManager, agentID string, options map[string]string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	// Check if agent exists
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Running agent: %s (%s)\n", agent.Name, agent.ID)
	fmt.Println("================================")

	// Run agent
	fmt.Println("Starting agent execution...")
	ctx := context.Background()
	
	// Create timeout context if specified
	timeout, ok := options["timeout"]
	if ok {
		timeoutSec, err := time.ParseDuration(timeout + "s")
		if err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeoutSec)
			defer cancel()
		}
	}
	
	// Run the agent
	err = am.RunAgent(ctx, agentID, options)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Get agent status
	status, err := am.GetAgentStatus(agentID)
	if err != nil {
		fmt.Printf("Error getting agent status: %v\n", err)
		return
	}
	
	// Create result object
	result := AgentRunResult{
		AgentID:      agentID,
		StartTime:    status.StartTime,
		EndTime:      status.EndTime,
		Success:      status.LastError == nil,
		CommandsRun:  status.SuccessCount + status.ErrorCount,
		Output:       status.LastOutput,
		ExitCode:     0,
	}
	
	if status.LastError != nil {
		result.Errors = []string{status.LastError.Error()}
		result.ExitCode = 1
	}

	// Show result
	fmt.Printf("\nAgent execution %s\n", getBoolText(result.Success, "succeeded", "failed"))
	fmt.Printf("Duration: %.1f seconds\n", result.EndTime.Sub(result.StartTime).Seconds())
	fmt.Printf("Commands executed: %d\n", result.CommandsRun)
	fmt.Printf("Exit code: %d\n", result.ExitCode)
	
	// Show output
	if result.Output != "" {
		fmt.Println("\nOutput:")
		fmt.Println(result.Output)
	}
	
	// Show errors
	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("- %s\n", err)
		}
	}
	
	// Show artifacts
	if len(result.ArtifactsPaths) > 0 {
		fmt.Println("\nArtifacts:")
		for _, path := range result.ArtifactsPaths {
			fmt.Printf("- %s\n", path)
		}
	}
}

// createAgent creates a new agent
func createAgent(am *AgentManager, name string, options map[string]string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	// Generate a unique ID for the agent
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id = strings.ReplaceAll(id, "_", "-")
	id = fmt.Sprintf("%s-%d", id, time.Now().Unix())

	// Create agent
	agent := Agent{
		ID:              id,
		Name:            name,
		Description:     fmt.Sprintf("%s agent", name),
		TaskTypes:       []string{"general"},
		Commands:        []AgentCommand{},
		TriggerPatterns: []string{},
		Context:         make(map[string]string),
		Tags:            []string{},
		AIPrompt:        "",
		Enabled:         true,
	}

	// Apply template if specified
	template, ok := options["template"]
	if ok {
		applyAgentTemplate(&agent, template)
	}

	// Create agent
	err := am.CreateAgent(agent)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		return
	}

	fmt.Printf("Agent created: %s (%s)\n", agent.Name, agent.ID)
	fmt.Println("Edit the agent with ':agent edit " + agent.ID + "'")
}

// applyAgentTemplate applies a template to an agent
func applyAgentTemplate(agent *Agent, template string) {
	// Apply template based on name
	switch template {
	case "build":
		agent.Description = "Automated build agent"
		agent.TaskTypes = []string{"build", "compile"}
		agent.Commands = []AgentCommand{
			{
				Command:    "make build",
				WorkingDir: ".",
				Timeout:    3600,
				RetryCount: 3,
			},
		}
		agent.TriggerPatterns = []string{"build", "compile"}
		agent.Context["task"] = "build"
		agent.Tags = []string{"build", "automation"}
		agent.AIPrompt = "You are a build assistant for the " + agent.Name + " agent. Your task is to build the project and fix any build errors."
	
	case "test":
		agent.Description = "Automated test agent"
		agent.TaskTypes = []string{"test", "verify"}
		agent.Commands = []AgentCommand{
			{
				Command:    "make test",
				WorkingDir: ".",
				Timeout:    3600,
				RetryCount: 3,
			},
		}
		agent.TriggerPatterns = []string{"test", "verify"}
		agent.Context["task"] = "test"
		agent.Tags = []string{"test", "automation"}
		agent.AIPrompt = "You are a test assistant for the " + agent.Name + " agent. Your task is to run tests and analyze results."
	
	case "deploy":
		agent.Description = "Automated deployment agent"
		agent.TaskTypes = []string{"deploy", "release"}
		agent.Commands = []AgentCommand{
			{
				Command:    "make deploy",
				WorkingDir: ".",
				Timeout:    3600,
				RetryCount: 3,
			},
		}
		agent.TriggerPatterns = []string{"deploy", "release"}
		agent.Context["task"] = "deploy"
		agent.Tags = []string{"deploy", "automation"}
		agent.AIPrompt = "You are a deployment assistant for the " + agent.Name + " agent. Your task is to deploy the project and verify the deployment."
	
	case "docker":
		agent.Description = "Docker-based build agent"
		agent.TaskTypes = []string{"build", "docker"}
		agent.Commands = []AgentCommand{
			{
				Command:    "docker build -t " + strings.ToLower(agent.Name) + " .",
				WorkingDir: ".",
				Timeout:    3600,
				RetryCount: 3,
			},
		}
		agent.DockerConfig = &AgentDockerConfig{
			Image:       strings.ToLower(agent.Name),
			Tag:         "latest",
			BuildContext: ".",
			UseCache:    true,
		}
		agent.TriggerPatterns = []string{"build", "docker"}
		agent.Context["task"] = "docker-build"
		agent.Tags = []string{"docker", "build", "automation"}
		agent.AIPrompt = "You are a Docker build assistant for the " + agent.Name + " agent. Your task is to build Docker images and fix any build errors."
	
	case "deepfry":
		agent.Description = "DeepFry PocketPC Builder"
		agent.TaskTypes = []string{"build", "error-fix", "uproot"}
		agent.Commands = []AgentCommand{
			{
				Command:        "uproot",
				WorkingDir:     "$DEEPFRY_HOME",
				Timeout:        3600,
				RetryCount:     3,
				ErrorPatterns:  []string{"failed to uproot", "error:"},
				SuccessPatterns: []string{"uproot completed successfully"},
			},
			{
				Command:        "run all",
				WorkingDir:     "$DEEPFRY_HOME",
				Timeout:        7200,
				RetryCount:     5,
				ErrorPatterns:  []string{"build failed", "error:"},
			},
		}
		agent.DockerConfig = &AgentDockerConfig{
			Image:       "deepfry-builder",
			Tag:         "latest",
			Volumes:     []string{"$DEEPFRY_HOME:/src"},
			UseCache:    true,
			CacheFrom:   []string{"deepfry-builder:cache"},
		}
		agent.TriggerPatterns = []string{"deepfry build", "uproot failing", "pocketpc defconfig"}
		agent.Context = map[string]string{
			"project":    "deepfry",
			"build_type": "pocketpc",
		}
		agent.Tags = []string{"deepfry", "build", "pocketpc"}
		agent.AIPrompt = "You are a DeepFry build assistant. Your task is to execute uproot commands, run builds, and fix common build errors for the PocketPC defconfig."
	}
}

// editAgent opens an editor to edit an agent
func editAgent(am *AgentManager, agentID string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	// Get agent
	_, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// For now, just show agent details
	fmt.Println("Agent editing is not yet implemented")
	fmt.Println("Here are the current agent details:")
	showAgent(am, agentID)
}

// deleteAgent deletes an agent
func deleteAgent(am *AgentManager, agentID string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete agent '%s' (%s)? (y/n): ", agent.Name, agent.ID)
	var confirm string
	fmt.Scanln(&confirm)
	
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Deletion cancelled")
		return
	}

	// Delete agent
	err = am.DeleteAgent(agentID)
	if err != nil {
		fmt.Printf("Error deleting agent: %v\n", err)
		return
	}

	fmt.Printf("Agent '%s' (%s) deleted\n", agent.Name, agent.ID)
}

// learnAgent learns a new agent from command sequence
func learnAgent(am *AgentManager, commandSequence string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	// Check for AI manager
	if am.aiManager == nil || !am.aiManager.IsEnabled() {
		fmt.Println("AI assistance not available")
		fmt.Println("Enable AI with ':ai on' to use agent learning")
		return
	}

	// For now, just show what would happen
	fmt.Println("Agent learning is not yet implemented")
	fmt.Printf("Would learn from command sequence: %s\n", commandSequence)
	
	// Generate a name for the agent
	name := "Learned Agent"
	if commandSequence != "" {
		parts := strings.Fields(commandSequence)
		if len(parts) > 0 {
			name = fmt.Sprintf("%s Agent", strings.Title(parts[0]))
		}
	}
	
	// Generate a unique ID for the agent
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id = strings.ReplaceAll(id, "_", "-")
	id = fmt.Sprintf("%s-%d", id, time.Now().Unix())
	
	fmt.Printf("Would create agent with name '%s' and ID '%s'\n", name, id)
}

// handleDockerCommands handles Docker-related commands
func handleDockerCommands(am *AgentManager, args []string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	if len(args) == 0 {
		fmt.Println("Usage: :agent docker <subcommand>")
		fmt.Println("Available subcommands:")
		fmt.Println("  list         - List Docker builds")
		fmt.Println("  cache stats  - Show Docker cache statistics")
		fmt.Println("  cache prune  - Prune Docker cache")
		fmt.Println("  build <id>   - Build Docker image for agent")
		return
	}

	cmd := args[0]
	switch cmd {
	case "list":
		// List Docker builds
		listDockerBuilds(am)
		return
		
	case "cache":
		if len(args) < 2 {
			fmt.Println("Usage: :agent docker cache <stats|prune>")
			return
		}
		
		switch args[1] {
		case "stats":
			// Show Docker cache statistics
			showDockerCacheStats(am)
			return
		case "prune":
			// Prune Docker cache
			pruneDockerCache(am)
			return
		default:
			fmt.Printf("Unknown cache command: %s\n", args[1])
			fmt.Println("Available commands: stats, prune")
			return
		}
		
	case "build":
		if len(args) < 2 {
			fmt.Println("Usage: :agent docker build <agent_id>")
			return
		}
		
		// Build Docker image for agent
		buildDockerImage(am, args[1])
		return
		
	default:
		fmt.Printf("Unknown docker command: %s\n", cmd)
		fmt.Println("Available commands: list, cache, build")
		return
	}
}

// listDockerBuilds lists Docker builds
func listDockerBuilds(am *AgentManager) {
	// Get all agents with Docker configuration
	agents := am.ListAgents()
	
	var dockerAgents []*Agent
	for _, agent := range agents {
		if agent.DockerConfig != nil {
			dockerAgents = append(dockerAgents, agent)
		}
	}
	
	if len(dockerAgents) == 0 {
		fmt.Println("No Docker-enabled agents found")
		return
	}
	
	fmt.Println("Docker-enabled Agents")
	fmt.Println("====================")
	
	for _, agent := range dockerAgents {
		fmt.Printf("%s (%s)\n", agent.Name, agent.ID)
		fmt.Printf("  Image: %s:%s\n", agent.DockerConfig.Image, agent.DockerConfig.Tag)
		
		if agent.DockerConfig.BuildContext != "" {
			fmt.Printf("  Build Context: %s\n", agent.DockerConfig.BuildContext)
		}
		
		if agent.DockerConfig.Dockerfile != "" {
			fmt.Printf("  Dockerfile: %s\n", agent.DockerConfig.Dockerfile)
		}
		
		fmt.Printf("  Use Cache: %s\n", getBoolText(agent.DockerConfig.UseCache, "Yes", "No"))
		fmt.Println()
	}
}

// showDockerCacheStats shows Docker cache statistics
func showDockerCacheStats(am *AgentManager) {
	stats := am.GetDockerCacheStats()
	
	fmt.Println("Docker Cache Statistics")
	fmt.Println("======================")
	
	fmt.Printf("Cache Size: %.2f MB\n", stats["cache_size_mb"].(float64))
	fmt.Printf("Cache Hits: %d\n", stats["cache_hits"].(int))
	fmt.Printf("Cache Misses: %d\n", stats["cache_misses"].(int))
	fmt.Printf("Cache Efficiency: %.1f%%\n", stats["cache_efficiency"].(float64))
	fmt.Printf("Build Configs: %d\n", stats["build_configs"].(int))
	fmt.Printf("Max Cache Age: %.0f days\n", stats["max_cache_age"].(float64))
}

// pruneDockerCache prunes Docker cache
func pruneDockerCache(am *AgentManager) {
	// Confirm pruning
	fmt.Print("Are you sure you want to prune the Docker cache? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Pruning cancelled")
		return
	}
	
	// Prune cache
	err := am.ClearDockerCache()
	if err != nil {
		fmt.Printf("Error pruning Docker cache: %v\n", err)
		return
	}
	
	fmt.Println("Docker cache pruned successfully")
}

// buildDockerImage builds a Docker image for an agent
func buildDockerImage(am *AgentManager, agentID string) {
	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Check if agent has Docker configuration
	if agent.DockerConfig == nil {
		fmt.Printf("Agent '%s' (%s) does not have Docker configuration\n", agent.Name, agent.ID)
		return
	}
	
	// Check Docker availability
	err = checkDockerAvailability()
	if err != nil {
		fmt.Printf("Docker is not available: %v\n", err)
		return
	}
	
	// For now, just show what would happen
	fmt.Println("Docker image building is not yet implemented")
	fmt.Printf("Would build Docker image %s:%s for agent '%s' (%s)\n",
		agent.DockerConfig.Image, agent.DockerConfig.Tag, agent.Name, agent.ID)
	
	// Show Docker configuration
	fmt.Println("\nDocker Configuration:")
	fmt.Printf("  Image: %s:%s\n", agent.DockerConfig.Image, agent.DockerConfig.Tag)
	
	if agent.DockerConfig.BuildContext != "" {
		fmt.Printf("  Build Context: %s\n", agent.DockerConfig.BuildContext)
	}
	
	if agent.DockerConfig.Dockerfile != "" {
		fmt.Printf("  Dockerfile: %s\n", agent.DockerConfig.Dockerfile)
	}
	
	if len(agent.DockerConfig.Volumes) > 0 {
		fmt.Printf("  Volumes: %s\n", strings.Join(agent.DockerConfig.Volumes, ", "))
	}
	
	if len(agent.DockerConfig.CacheFrom) > 0 {
		fmt.Printf("  Cache From: %s\n", strings.Join(agent.DockerConfig.CacheFrom, ", "))
	}
	
	fmt.Printf("  Use Cache: %s\n", getBoolText(agent.DockerConfig.UseCache, "Yes", "No"))
}

// showAgentStats shows statistics about agents
func showAgentStats(am *AgentManager) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}
	
	stats := am.GetAgentStats()
	
	fmt.Println("Agent Statistics")
	fmt.Println("===============")
	
	fmt.Printf("Total Agents: %d\n", stats["total_agents"].(int))
	fmt.Printf("Enabled Agents: %d\n", stats["enabled_agents"].(int))
	fmt.Printf("Total Runs: %d\n", stats["total_runs"].(int))
	fmt.Printf("Successful Runs: %d\n", stats["successful_runs"].(int))
	fmt.Printf("Success Rate: %.1f%%\n", stats["success_rate"].(float64))
	fmt.Printf("Average Run Time: %.1f seconds\n", stats["avg_run_time"].(float64))
	
	// Show Docker cache statistics if Docker builds are enabled
	if am.config.UseDockerBuilds {
		fmt.Println("\nDocker Cache Statistics")
		fmt.Println("----------------------")
		
		dockerStats := am.GetDockerCacheStats()
		fmt.Printf("Cache Size: %.2f MB\n", dockerStats["cache_size_mb"].(float64))
		fmt.Printf("Cache Hits: %d\n", dockerStats["cache_hits"].(int))
		fmt.Printf("Cache Misses: %d\n", dockerStats["cache_misses"].(int))
		fmt.Printf("Cache Efficiency: %.1f%%\n", dockerStats["cache_efficiency"].(float64))
	}
}

// showAgentHelp displays help for agent commands
func showAgentHelp() {
	fmt.Println("Agent Commands")
	fmt.Println("=============")
	fmt.Println("  :agent                - Show agent manager status")
	fmt.Println("  :agent enable         - Initialize and enable agent manager")
	fmt.Println("  :agent disable        - Disable agent manager")
	fmt.Println("  :agent list           - List all agents")
	fmt.Println("  :agent show <id>      - Show agent details")
	fmt.Println("  :agent run <id>       - Run an agent")
	fmt.Println("  :agent create <name>  - Create a new agent")
	fmt.Println("  :agent edit <id>      - Edit agent configuration")
	fmt.Println("  :agent delete <id>    - Delete an agent")
	fmt.Println("  :agent learn <cmds>   - Learn a new agent from command sequence")
	fmt.Println("  :agent stats          - Show agent statistics")
	fmt.Println("")
	fmt.Println("Docker-related Commands:")
	fmt.Println("  :agent docker list           - List Docker builds")
	fmt.Println("  :agent docker cache stats    - Show Docker cache statistics")
	fmt.Println("  :agent docker cache prune    - Prune Docker cache")
	fmt.Println("  :agent docker build <id>     - Build Docker image for agent")
	fmt.Println("")
	fmt.Println("Error Management Commands:")
	fmt.Println("  :agent errors list           - List learned error solutions")
	fmt.Println("  :agent errors export         - Export learned solutions to pattern file")
	fmt.Println("")
	fmt.Println("Agent Templates:")
	fmt.Println("  :agent create <name> --template=build    - Create a build agent")
	fmt.Println("  :agent create <name> --template=test     - Create a test agent")
	fmt.Println("  :agent create <name> --template=deploy   - Create a deployment agent")
	fmt.Println("  :agent create <name> --template=docker   - Create a Docker-based agent")
	fmt.Println("  :agent create <name> --template=deepfry  - Create a DeepFry builder agent")
}

// parseAgentOptions parses agent command options
func parseAgentOptions(args []string) map[string]string {
	options := make(map[string]string)
	
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2)
			if len(parts) == 2 {
				options[parts[0]] = parts[1]
			} else {
				options[parts[0]] = "true"
			}
		}
	}
	
	return options
}

// getBoolText returns textual representation of a boolean value
func getBoolText(value bool, trueText, falseText string) string {
	if value {
		return trueText
	}
	return falseText
}

// GetRunHistory returns the run history for an agent
func (am *AgentManager) GetRunHistory(agentID string, limit int) []AgentRunResult {
	// This is a placeholder implementation
	// In a real implementation, this would retrieve the run history from storage
	return []AgentRunResult{}
}

// GetDockerCacheStats returns Docker cache statistics
func (am *AgentManager) GetDockerCacheStats() map[string]interface{} {
	// This is a placeholder implementation
	return map[string]interface{}{
		"cache_size_mb":   float64(0),
		"cache_hits":      int(0),
		"cache_misses":    int(0),
		"cache_efficiency": float64(0),
		"build_configs":   int(0),
		"max_cache_age":   float64(0),
	}
}

// ClearDockerCache clears the Docker cache
func (am *AgentManager) ClearDockerCache() error {
	// This is a placeholder implementation
	return nil
}

// GetAgentStats returns agent statistics
func (am *AgentManager) GetAgentStats() map[string]interface{} {
	// This is a placeholder implementation
	return map[string]interface{}{
		"total_agents":    int(0),
		"enabled_agents":  int(0),
		"total_runs":      int(0),
		"successful_runs": int(0),
		"success_rate":    float64(0),
		"avg_run_time":    float64(0),
	}
}

// checkDockerAvailability checks if Docker is available
func checkDockerAvailability() error {
	// Check if Docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Docker not found: %v", err)
	}
	
	// Check Docker version
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get Docker version: %v", err)
	}
	
	return nil
}

// handleErrorCommands handles error solution commands
func handleErrorCommands(am *AgentManager, args []string) {
	if !am.IsEnabled() {
		fmt.Println("Agent manager not enabled")
		fmt.Println("Run ':agent enable' to enable agent manager")
		return
	}

	// Get error learning manager
	errorLearningMgr := GetErrorLearningManager()
	if errorLearningMgr == nil {
		fmt.Println("Error learning system not available")
		return
	}

	if len(args) == 0 {
		fmt.Println("Usage: :agent errors <subcommand>")
		fmt.Println("Available subcommands:")
		fmt.Println("  list      - List learned error solutions")
		fmt.Println("  export    - Export learned error solutions to patterns file")
		fmt.Println("  learn     - Learn a new error solution")
		fmt.Println("  fix       - Fix an error using learned solutions")
		fmt.Println("  stats     - Show error learning statistics")
		return
	}

	cmd := args[0]
	switch cmd {
	case "list":
		// List learned error solutions
		solutions := errorLearningMgr.ListSolutions()
		
		if len(solutions) == 0 {
			fmt.Println("No learned error solutions found")
			return
		}
		
		fmt.Println("Learned Error Solutions")
		fmt.Println("======================")
		
		for _, solution := range solutions {
			// Calculate success rate
			total := solution.SuccessCount + solution.FailureCount
			var successRate float64
			if total > 0 {
				successRate = float64(solution.SuccessCount) / float64(total) * 100
			}
			
			fmt.Printf("Error Pattern: %s\n", solution.Pattern)
			fmt.Printf("Solution: %s\n", solution.Solution)
			if solution.Description != "" {
				fmt.Printf("Description: %s\n", solution.Description)
			}
			fmt.Printf("Success Rate: %.1f%% (%d/%d)\n", successRate, solution.SuccessCount, total)
			fmt.Printf("Source: %s\n", solution.Source)
			fmt.Println()
		}
		return

	case "export":
		// Export learned error solutions to patterns file
		err := errorLearningMgr.ExportToErrorPatterns()
		if err != nil {
			fmt.Printf("Error exporting learned solutions: %v\n", err)
			return
		}
		fmt.Println("Learned error solutions exported to patterns file")
		return

	case "learn":
		// Learn a new error solution
		if len(args) < 3 {
			fmt.Println("Usage: :agent errors learn <error_pattern> <solution>")
			return
		}
		
		errorPattern := args[1]
		solution := args[2]
		
		// Get optional description
		description := ""
		if len(args) > 3 {
			description = args[3]
		}
		
		// Add error solution
		errorLearningMgr.AddErrorSolution(errorPattern, solution, description, "", true, "user")
		fmt.Println("Error solution added successfully")
		return

	case "fix":
		// Fix an error using learned solutions
		if len(args) < 2 {
			fmt.Println("Usage: :agent errors fix <error_message>")
			return
		}
		
		// Get error message
		errorMsg := args[1]
		
		// Get context
		context := ""
		if len(args) > 2 {
			context = args[2]
		}
		
		// Try to fix the error
		knownSolution, solution, err := errorLearningMgr.FixErrorAutomatically(errorMsg, context)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		
		if knownSolution {
			fmt.Println("Found a solution from learning history:")
		} else {
			fmt.Println("Generated a solution using AI:")
		}
		
		fmt.Println(solution)
		return

	case "stats":
		// Show error learning statistics
		fmt.Println("Error Learning Statistics")
		fmt.Println("========================")
		
		solutions := errorLearningMgr.ListSolutions()
		
		// Calculate statistics
		totalSolutions := len(solutions)
		aiGeneratedCount := 0
		userDefinedCount := 0
		systemCount := 0
		successfulCount := 0
		
		for _, sol := range solutions {
			switch sol.Source {
			case "ai":
				aiGeneratedCount++
			case "user":
				userDefinedCount++
			case "system":
				systemCount++
			}
			
			if sol.SuccessCount > 0 {
				successfulCount++
			}
		}
		
		fmt.Printf("Total Solutions: %d\n", totalSolutions)
		fmt.Printf("AI-Generated: %d\n", aiGeneratedCount)
		fmt.Printf("User-Defined: %d\n", userDefinedCount)
		fmt.Printf("System-Provided: %d\n", systemCount)
		fmt.Printf("Successful Solutions: %d\n", successfulCount)
		
		if totalSolutions > 0 {
			fmt.Printf("Success Rate: %.1f%%\n", float64(successfulCount)/float64(totalSolutions)*100)
		}
		return

	default:
		fmt.Printf("Unknown errors command: %s\n", cmd)
		fmt.Println("Available commands: list, export, learn, fix, stats")
		return
	}
}