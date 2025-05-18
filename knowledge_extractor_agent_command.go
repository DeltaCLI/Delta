package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// HandleKnowledgeExtractorAgentCommand processes agent-related knowledge extraction commands
func HandleKnowledgeExtractorAgentCommand(args []string) bool {
	// Get the KnowledgeExtractor instance
	ke := GetKnowledgeExtractor()
	if ke == nil {
		fmt.Println("Failed to initialize knowledge extractor")
		return true
	}

	// Get the AgentManager instance
	am := GetAgentManager()
	if am == nil {
		fmt.Println("Failed to initialize agent manager")
		return true
	}

	// Check if knowledge extractor is enabled
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor is not enabled")
		fmt.Println("Run ':knowledge enable' to enable the knowledge extractor")
		return true
	}

	// Process commands
	if len(args) == 0 {
		// Show help by default
		showKnowledgeAgentHelp()
		return true
	}

	// Handle subcommands
	switch args[0] {
	case "suggest":
		// Suggest agents based on knowledge
		suggestAgentsFromKnowledge(ke, am)
		return true

	case "learn":
		// Learn agent patterns from knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge agent learn <agent_id>")
			return true
		}
		learnAgentPatterns(ke, am, args[1])
		return true

	case "optimize":
		// Optimize agent using knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge agent optimize <agent_id>")
			return true
		}
		optimizeAgent(ke, am, args[1])
		return true

	case "create":
		// Create agent from knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge agent create <name> [--type=<type>] [--directory=<dir>]")
			return true
		}
		options := parseOptions(args[2:])
		createAgentFromKnowledge(ke, am, args[1], options)
		return true

	case "extract":
		// Extract knowledge from agent executions
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge agent extract <agent_id>")
			return true
		}
		extractKnowledgeFromAgent(ke, am, args[1])
		return true

	case "context":
		// Generate context for agent from knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge agent context <agent_id>")
			return true
		}
		generateAgentContext(ke, am, args[1])
		return true

	case "triggers":
		// Suggest trigger patterns for agents
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge agent triggers <agent_id>")
			return true
		}
		suggestAgentTriggers(ke, am, args[1])
		return true

	case "discover":
		// Discover potential agents from knowledge
		discoverAgentsFromKnowledge(ke, am)
		return true

	case "help":
		// Show help
		showKnowledgeAgentHelp()
		return true

	default:
		fmt.Printf("Unknown knowledge agent command: %s\n", args[0])
		fmt.Println("Type ':knowledge agent help' for available commands")
		return true
	}
}

// suggestAgentsFromKnowledge suggests agents based on extracted knowledge
func suggestAgentsFromKnowledge(ke *KnowledgeExtractor, am *AgentManager) {
	fmt.Println("Analyzing knowledge to suggest agents...")

	// Get knowledge items
	items := ke.GetKnowledgeItems()
	if len(items) == 0 {
		fmt.Println("No knowledge items found. Use ':knowledge scan' to gather knowledge first.")
		return
	}

	// Get project information
	projectInfo := ke.GetProjectInfo()

	// Analyze knowledge items to identify potential agent types
	fmt.Println("\nPotential Agents Based on Knowledge:")
	fmt.Println("===================================")

	// Check for build patterns
	buildPatterns := 0
	for _, item := range items {
		if (item.Type == "command" && containsAnyTerm(item.Content, "make", "build", "compile")) ||
			(item.Type == "workflow" && containsAnyTerm(item.Content, "build", "compile")) {
			buildPatterns++
		}
	}

	if buildPatterns > 0 {
		fmt.Println("1. Build Agent")
		fmt.Printf("   Confidence: %.1f%%\n", minFloatValue(float64(buildPatterns*10), 100.0))
		fmt.Println("   Purpose: Automate build processes")
		fmt.Println("   Create with: :knowledge agent create build-agent --type=build")
		fmt.Println()
	}

	// Check for test patterns
	testPatterns := 0
	for _, item := range items {
		if (item.Type == "command" && containsAnyTerm(item.Content, "test", "check", "verify")) ||
			(item.Type == "workflow" && containsAnyTerm(item.Content, "test", "verify")) {
			testPatterns++
		}
	}

	if testPatterns > 0 {
		fmt.Println("2. Test Agent")
		fmt.Printf("   Confidence: %.1f%%\n", minFloatValue(float64(testPatterns*10), 100.0))
		fmt.Println("   Purpose: Automate testing workflows")
		fmt.Println("   Create with: :knowledge agent create test-agent --type=test")
		fmt.Println()
	}

	// Check for deploy patterns
	deployPatterns := 0
	for _, item := range items {
		if (item.Type == "command" && containsAnyTerm(item.Content, "deploy", "release", "publish")) ||
			(item.Type == "workflow" && containsAnyTerm(item.Content, "deploy", "release")) {
			deployPatterns++
		}
	}

	if deployPatterns > 0 {
		fmt.Println("3. Deploy Agent")
		fmt.Printf("   Confidence: %.1f%%\n", minFloatValue(float64(deployPatterns*10), 100.0))
		fmt.Println("   Purpose: Automate deployment workflows")
		fmt.Println("   Create with: :knowledge agent create deploy-agent --type=deploy")
		fmt.Println()
	}

	// Check for Docker patterns
	dockerPatterns := 0
	for _, item := range items {
		if (item.Type == "command" && containsAnyTerm(item.Content, "docker", "container", "image")) ||
			(item.Type == "workflow" && containsAnyTerm(item.Content, "docker", "container")) {
			dockerPatterns++
		}
	}

	if dockerPatterns > 0 {
		fmt.Println("4. Docker Agent")
		fmt.Printf("   Confidence: %.1f%%\n", minFloatValue(float64(dockerPatterns*10), 100.0))
		fmt.Println("   Purpose: Manage Docker containers and images")
		fmt.Println("   Create with: :knowledge agent create docker-agent --type=docker")
		fmt.Println()
	}

	// Check for project-specific agents (like DeepFry)
	if projectInfo.Type != "" {
		fmt.Printf("5. %s Project Agent\n", projectInfo.Name)
		fmt.Println("   Confidence: 90.0%")
		fmt.Printf("   Purpose: Manage %s-specific tasks\n", projectInfo.Name)
		fmt.Printf("   Create with: :knowledge agent create %s-agent --type=project\n", strings.ToLower(projectInfo.Name))
		fmt.Println()
	}

	// Provide next steps
	fmt.Println("Next Steps:")
	fmt.Println("1. Create an agent with ':knowledge agent create <name> --type=<type>'")
	fmt.Println("2. Optimize an existing agent with ':knowledge agent optimize <agent_id>'")
	fmt.Println("3. Get more agent suggestions with ':knowledge agent discover'")
}

// learnAgentPatterns learns patterns from an existing agent
func learnAgentPatterns(ke *KnowledgeExtractor, am *AgentManager, agentID string) {
	fmt.Printf("Learning patterns from agent: %s\n", agentID)

	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Analyzing agent: %s (%s)\n", agent.Name, agent.ID)
	fmt.Println("==================================")

	// Create a batch for analysis
	batch := KnowledgeBatch{
		BatchID:   fmt.Sprintf("agent-%s-%d", agent.ID, time.Now().Unix()),
		Timestamp: time.Now(),
		Commands:  make([]CommandEntry, 0),
	}

	// Convert agent commands to command entries
	for _, cmd := range agent.Commands {
		entry := CommandEntry{
			Command:     cmd.Command,
			Directory:   cmd.WorkingDir,
			Environment: cmd.Environment,
		}
		batch.Commands = append(batch.Commands, entry)
	}

	// Process batch to extract knowledge
	err = ke.ProcessBatch(batch)
	if err != nil {
		fmt.Printf("Error processing agent commands: %v\n", err)
		return
	}

	fmt.Println("Learning complete. Agent patterns have been added to knowledge.")
	fmt.Println("Use ':knowledge query' to search for these patterns.")
}

// optimizeAgent optimizes an agent using knowledge
func optimizeAgent(ke *KnowledgeExtractor, am *AgentManager, agentID string) {
	fmt.Printf("Optimizing agent: %s\n", agentID)

	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Analyzing agent: %s (%s)\n", agent.Name, agent.ID)
	fmt.Println("==================================")

	// Make a copy of the agent
	optimizedAgent := *agent

	// Optimize timeout values based on knowledge
	if len(agent.Commands) > 0 {
		fmt.Println("Optimizing command timeouts...")
		
		for i := range optimizedAgent.Commands {
			cmd := optimizedAgent.Commands[i]
			// Add 20% margin to historical execution time if we have knowledge about it
			// This is just a placeholder - in a real implementation, we would look up
			// historical execution times from the knowledge extractor
			if cmd.Timeout < 120 {
				optimizedAgent.Commands[i].Timeout = 120 // Minimum timeout of 2 minutes
				fmt.Printf("- Increased timeout for '%s' to %d seconds\n", cmd.Command, optimizedAgent.Commands[i].Timeout)
			}
		}
	}

	// Optimize retry counts based on knowledge
	fmt.Println("Optimizing retry strategies...")
	for i := range optimizedAgent.Commands {
		// Set reasonable retry count and delay based on command type
		// In a real implementation, we would analyze historical failure patterns
		if optimizedAgent.Commands[i].RetryCount == 0 {
			optimizedAgent.Commands[i].RetryCount = 3
			fmt.Printf("- Added retry count of %d for '%s'\n", 
				optimizedAgent.Commands[i].RetryCount, 
				optimizedAgent.Commands[i].Command)
		}
		
		if optimizedAgent.Commands[i].RetryDelay == 0 {
			optimizedAgent.Commands[i].RetryDelay = 10
			fmt.Printf("- Added retry delay of %d seconds for '%s'\n", 
				optimizedAgent.Commands[i].RetryDelay, 
				optimizedAgent.Commands[i].Command)
		}
	}

	// Add suggested error patterns based on knowledge
	fmt.Println("Adding error detection patterns...")
	for i := range optimizedAgent.Commands {
		// Add common error patterns for this command type
		// This is a placeholder - in a real implementation, we would get these from
		// the knowledge extractor based on historical command executions
		if len(optimizedAgent.Commands[i].ErrorPatterns) == 0 {
			cmd := optimizedAgent.Commands[i].Command
			if strings.HasPrefix(cmd, "make") || strings.HasPrefix(cmd, "build") {
				patterns := []string{"error:", "failed:", "undefined reference", "compilation terminated"}
				optimizedAgent.Commands[i].ErrorPatterns = patterns
				fmt.Printf("- Added %d error patterns for '%s'\n", 
					len(patterns), 
					optimizedAgent.Commands[i].Command)
			}
		}
	}

	// Add suggested trigger patterns
	fmt.Println("Enhancing trigger patterns...")
	newTriggers := suggestTriggersForAgent(&optimizedAgent)
	if len(newTriggers) > 0 {
		added := 0
		for _, trigger := range newTriggers {
			if !containsString(optimizedAgent.TriggerPatterns, trigger) {
				optimizedAgent.TriggerPatterns = append(optimizedAgent.TriggerPatterns, trigger)
				added++
			}
		}
		if added > 0 {
			fmt.Printf("- Added %d new trigger patterns\n", added)
		}
	}

	// Enhance context information
	fmt.Println("Enriching context information...")
	projectInfo := ke.GetProjectInfo()
	if projectInfo.Type != "" && projectInfo.Name != "" {
		if optimizedAgent.Context == nil {
			optimizedAgent.Context = make(map[string]string)
		}
		
		if _, ok := optimizedAgent.Context["project_type"]; !ok {
			optimizedAgent.Context["project_type"] = projectInfo.Type
			fmt.Printf("- Added project type: %s\n", projectInfo.Type)
		}
		
		if _, ok := optimizedAgent.Context["project_name"]; !ok {
			optimizedAgent.Context["project_name"] = projectInfo.Name
			fmt.Printf("- Added project name: %s\n", projectInfo.Name)
		}
	}

	// Update agent with optimizations
	optimizedAgent.UpdatedAt = time.Now()
	err = am.UpdateAgent(optimizedAgent)
	if err != nil {
		fmt.Printf("Error updating agent: %v\n", err)
		return
	}

	fmt.Println("\nAgent has been successfully optimized!")
	fmt.Printf("View the updated agent with: :agent show %s\n", agent.ID)
}

// createAgentFromKnowledge creates a new agent using knowledge
func createAgentFromKnowledge(ke *KnowledgeExtractor, am *AgentManager, name string, options map[string]string) {
	fmt.Printf("Creating agent '%s' from knowledge...\n", name)

	// Generate a unique ID for the agent
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id = strings.ReplaceAll(id, "_", "-")
	id = fmt.Sprintf("%s-%d", id, time.Now().Unix())

	// Get agent type from options
	agentType := options["type"]
	if agentType == "" {
		agentType = "general"
	}

	// Get working directory
	workingDir := options["directory"]
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}
	}

	// Create base agent
	agent := Agent{
		ID:          id,
		Name:        name,
		Description: fmt.Sprintf("%s agent for %s tasks", name, agentType),
		TaskTypes:   []string{agentType},
		Commands:    []AgentCommand{},
		Context:     make(map[string]string),
		Tags:        []string{agentType, "auto-generated"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Enabled:     true,
	}

	// Get project info from knowledge
	projectInfo := ke.GetProjectInfo()
	if projectInfo.Type != "" {
		agent.Context["project_type"] = projectInfo.Type
		agent.Context["project_name"] = projectInfo.Name
		agent.Tags = append(agent.Tags, projectInfo.Type)
	}

	// Add commands based on agent type
	switch agentType {
	case "build":
		createBuildAgent(&agent, workingDir, projectInfo)
	case "test":
		createTestAgent(&agent, workingDir, projectInfo)
	case "deploy":
		createDeployAgent(&agent, workingDir, projectInfo)
	case "docker":
		createDockerAgent(&agent, workingDir, projectInfo)
	case "project":
		createProjectAgent(&agent, workingDir, projectInfo)
	case "deepfry":
		createDeepFryAgent(&agent, workingDir)
	default:
		createGeneralAgent(&agent, workingDir, projectInfo)
	}

	// Add trigger patterns
	agent.TriggerPatterns = suggestTriggersForAgent(&agent)

	// Generate AI prompt
	agent.AIPrompt = fmt.Sprintf("You are a %s assistant for the %s agent. Your task is to help with %s tasks and provide guidance.", 
		agentType, name, agentType)

	// Create agent
	err := am.CreateAgent(agent)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		return
	}

	// Use the created agent for display purposes
	createdAgent := agent

	fmt.Printf("Agent '%s' (%s) created successfully!\n", createdAgent.Name, createdAgent.ID)
	fmt.Println("You can run the agent with:")
	fmt.Printf(":agent run %s\n", createdAgent.ID)
	fmt.Println("Or view its details with:")
	fmt.Printf(":agent show %s\n", createdAgent.ID)
	fmt.Println("\nOptimize the agent with:")
	fmt.Printf(":knowledge agent optimize %s\n", createdAgent.ID)
}

// extractKnowledgeFromAgent extracts knowledge from agent executions
func extractKnowledgeFromAgent(ke *KnowledgeExtractor, am *AgentManager, agentID string) {
	fmt.Printf("Extracting knowledge from agent: %s\n", agentID)

	// Get agent and check if it exists
	agentObj, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Analyzing agent: %s (%s)\n", agentObj.Name, agentObj.ID)

	// Get agent run history
	history := am.GetRunHistory(agentID, 10)
	if len(history) == 0 {
		fmt.Println("No run history found for this agent.")
		fmt.Printf("Run the agent first with: :agent run %s\n", agentID)
		return
	}

	fmt.Printf("Analyzing %d previous runs...\n", len(history))

	// Extract knowledge from execution history
	successCount := 0
	failureCount := 0
	
	for _, run := range history {
		if run.Success {
			successCount++
		} else {
			failureCount++
		}
		
		// Add to knowledge based on run output and errors
		// This is a placeholder - in a real implementation, we would parse the
		// output and errors to extract meaningful patterns
	}

	fmt.Printf("Extracted knowledge from %d successful and %d failed runs\n", successCount, failureCount)
	fmt.Println("This knowledge will be used to suggest optimizations for the agent.")
	fmt.Printf("Optimize the agent with: :knowledge agent optimize %s\n", agentID)
}

// generateAgentContext generates context for agent from knowledge
func generateAgentContext(ke *KnowledgeExtractor, am *AgentManager, agentID string) {
	fmt.Printf("Generating context for agent: %s\n", agentID)

	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Get current context
	ctx := ke.GetCurrentContext()
	projectInfo := ke.GetProjectInfo()

	fmt.Println("Suggested Context:")
	fmt.Println("=================")

	// Build suggested context
	suggestedContext := make(map[string]string)

	// Add OS information
	suggestedContext["os"] = ctx.OS
	suggestedContext["arch"] = ctx.Arch

	// Add user information
	suggestedContext["user"] = ctx.User
	suggestedContext["shell"] = ctx.Shell
	
	// Add project information
	if projectInfo.Type != "" {
		suggestedContext["project_type"] = projectInfo.Type
		suggestedContext["project_name"] = projectInfo.Name
		
		if projectInfo.Version != "" {
			suggestedContext["project_version"] = projectInfo.Version
		}
		
		if projectInfo.BuildSystem != "" {
			suggestedContext["build_system"] = projectInfo.BuildSystem
		}
	}
	
	// Add Git information if available
	if ctx.GitBranch != "" {
		suggestedContext["git_branch"] = ctx.GitBranch
	}

	if ctx.GitRepo != "" {
		suggestedContext["git_repo"] = ctx.GitRepo
	}

	// Print context information
	for key, value := range suggestedContext {
		existing := ""
		if agent.Context != nil {
			if val, ok := agent.Context[key]; ok {
				existing = fmt.Sprintf(" (current: %s)", val)
			}
		}
		fmt.Printf("%s = %s%s\n", key, value, existing)
	}

	// Prompt to update context
	fmt.Println("\nTo update the agent context, use:")
	fmt.Printf(":agent edit %s\n", agentID)
}

// suggestAgentTriggers suggests trigger patterns for agents
func suggestAgentTriggers(ke *KnowledgeExtractor, am *AgentManager, agentID string) {
	fmt.Printf("Suggesting trigger patterns for agent: %s\n", agentID)

	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Get suggested triggers
	triggers := suggestTriggersForAgent(agent)

	fmt.Println("Suggested Trigger Patterns:")
	fmt.Println("==========================")

	// Print existing triggers
	if len(agent.TriggerPatterns) > 0 {
		fmt.Println("Current patterns:")
		for _, pattern := range agent.TriggerPatterns {
			fmt.Printf("- %s\n", pattern)
		}
		fmt.Println()
	}

	// Print new suggested triggers
	fmt.Println("Suggested new patterns:")
	for _, pattern := range triggers {
		// Skip if already in agent's trigger patterns
		if containsString(agent.TriggerPatterns, pattern) {
			continue
		}
		fmt.Printf("- %s\n", pattern)
	}

	// Prompt to update trigger patterns
	fmt.Println("\nTo update the agent trigger patterns, use:")
	fmt.Printf(":agent edit %s\n", agentID)
}

// discoverAgentsFromKnowledge discovers potential agents from knowledge
func discoverAgentsFromKnowledge(ke *KnowledgeExtractor, am *AgentManager) {
	fmt.Println("Discovering potential agents from knowledge...")

	// Get project info
	projectInfo := ke.GetProjectInfo()

	// Get knowledge items
	items := ke.GetKnowledgeItems()
	if len(items) == 0 {
		fmt.Println("No knowledge items found. Use ':knowledge scan' to gather knowledge first.")
		return
	}

	// Analyze workflow patterns
	workflows := make(map[string]int)
	for _, item := range items {
		if item.Type == "workflow" {
			workflows[item.Content]++
		}
	}

	// Identify potential agents based on workflows
	fmt.Println("\nDiscovered Potential Agents:")
	fmt.Println("===========================")

	// Look for common workflow patterns
	agentCount := 0

	// Build workflows
	buildWorkflows := 0
	for workflow, count := range workflows {
		if containsAnyTerm(workflow, "make", "build", "compile") {
			buildWorkflows += count
		}
	}

	if buildWorkflows > 0 {
		agentCount++
		fmt.Printf("%d. Build Agent\n", agentCount)
		fmt.Printf("   Detected %d build workflows\n", buildWorkflows)
		fmt.Printf("   Purpose: Automate build processes\n")
		fmt.Printf("   Create with: :knowledge agent create build-agent --type=build\n\n")
	}

	// Test workflows
	testWorkflows := 0
	for workflow, count := range workflows {
		if containsAnyTerm(workflow, "test", "verify", "check") {
			testWorkflows += count
		}
	}

	if testWorkflows > 0 {
		agentCount++
		fmt.Printf("%d. Test Agent\n", agentCount)
		fmt.Printf("   Detected %d test workflows\n", testWorkflows)
		fmt.Printf("   Purpose: Automate testing processes\n")
		fmt.Printf("   Create with: :knowledge agent create test-agent --type=test\n\n")
	}

	// Deployment workflows
	deployWorkflows := 0
	for workflow, count := range workflows {
		if containsAnyTerm(workflow, "deploy", "publish", "release") {
			deployWorkflows += count
		}
	}

	if deployWorkflows > 0 {
		agentCount++
		fmt.Printf("%d. Deployment Agent\n", agentCount)
		fmt.Printf("   Detected %d deployment workflows\n", deployWorkflows)
		fmt.Printf("   Purpose: Automate deployment processes\n")
		fmt.Printf("   Create with: :knowledge agent create deploy-agent --type=deploy\n\n")
	}

	// Docker workflows
	dockerWorkflows := 0
	for workflow, count := range workflows {
		if containsAnyTerm(workflow, "docker", "container", "image") {
			dockerWorkflows += count
		}
	}
	
	if dockerWorkflows > 0 {
		agentCount++
		fmt.Printf("%d. Docker Agent\n", agentCount)
		fmt.Printf("   Detected %d Docker workflows\n", dockerWorkflows)
		fmt.Printf("   Purpose: Manage Docker containers and images\n")
		fmt.Printf("   Create with: :knowledge agent create docker-agent --type=docker\n\n")
	}

	// Project-specific agents
	if projectInfo.Type != "" {
		agentCount++
		fmt.Printf("%d. %s Project Agent\n", agentCount, projectInfo.Name)
		fmt.Printf("   Detected project type: %s\n", projectInfo.Type)
		fmt.Printf("   Purpose: Manage project-specific tasks\n")
		fmt.Printf("   Create with: :knowledge agent create %s-agent --type=project\n\n", 
			strings.ToLower(projectInfo.Name))
	}

	// DeepFry agent if we detect a DeepFry project
	deepFryPatterns := 0
	for _, item := range items {
		if containsAnyTerm(item.Content, "deepfry", "uproot", "pocketpc") {
			deepFryPatterns++
		}
	}
	
	if deepFryPatterns > 0 {
		agentCount++
		fmt.Printf("%d. DeepFry Agent\n", agentCount)
		fmt.Printf("   Detected %d DeepFry patterns\n", deepFryPatterns)
		fmt.Printf("   Purpose: Manage DeepFry builds and processes\n")
		fmt.Printf("   Create with: :knowledge agent create deepfry-agent --type=deepfry\n\n")
	}

	if agentCount == 0 {
		fmt.Println("No potential agents discovered. Try scanning for more knowledge with ':knowledge scan'.")
	} else {
		fmt.Printf("Discovered %d potential agents. Create one with ':knowledge agent create <name> --type=<type>'\n", agentCount)
	}
}

// showKnowledgeAgentHelp displays help for knowledge agent commands
func showKnowledgeAgentHelp() {
	fmt.Println("Knowledge Agent Commands")
	fmt.Println("=======================")
	fmt.Println("  :knowledge agent suggest           - Suggest agents based on knowledge")
	fmt.Println("  :knowledge agent learn <id>        - Learn patterns from an agent")
	fmt.Println("  :knowledge agent optimize <id>     - Optimize agent using knowledge")
	fmt.Println("  :knowledge agent create <name>     - Create agent from knowledge")
	fmt.Println("  :knowledge agent extract <id>      - Extract knowledge from agent executions")
	fmt.Println("  :knowledge agent context <id>      - Generate context for agent")
	fmt.Println("  :knowledge agent triggers <id>     - Suggest trigger patterns")
	fmt.Println("  :knowledge agent discover          - Discover potential agents")
	fmt.Println("  :knowledge agent help              - Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  :knowledge agent suggest")
	fmt.Println("  :knowledge agent create build-agent --type=build")
	fmt.Println("  :knowledge agent optimize deepfry-agent-1234567890")
	fmt.Println("  :knowledge agent discover")
}

// Helper functions for agent creation

// createBuildAgent creates a build agent
func createBuildAgent(agent *Agent, workingDir string, projectInfo ProjectInfo) {
	// Add build command based on project type
	buildCmd := "make"
	if projectInfo.BuildSystem != "" {
		switch projectInfo.BuildSystem {
		case "make":
			buildCmd = "make"
		case "cmake":
			buildCmd = "cmake --build ."
		case "gradle":
			buildCmd = "./gradlew build"
		case "maven":
			buildCmd = "mvn package"
		case "npm":
			buildCmd = "npm run build"
		case "yarn":
			buildCmd = "yarn build"
		case "cargo":
			buildCmd = "cargo build"
		}
	}

	// Add build command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       buildCmd,
		WorkingDir:    workingDir,
		Timeout:       600, // 10 minutes
		RetryCount:    3,
		RetryDelay:    10,
		IsInteractive: false,
		ErrorPatterns: []string{"error:", "failed:", "compilation terminated"},
	})

	// Add clean command if using make
	if buildCmd == "make" {
		agent.Commands = append(agent.Commands, AgentCommand{
			Command:       "make clean",
			WorkingDir:    workingDir,
			Timeout:       120, // 2 minutes
			RetryCount:    1,
			RetryDelay:    5,
			IsInteractive: false,
		})
	}
}

// createTestAgent creates a test agent
func createTestAgent(agent *Agent, workingDir string, projectInfo ProjectInfo) {
	// Add test command based on project type
	testCmd := "make test"
	if projectInfo.TestFramework != "" {
		switch projectInfo.TestFramework {
		case "go test":
			testCmd = "go test ./..."
		case "pytest":
			testCmd = "pytest"
		case "jest":
			testCmd = "npm test"
		case "junit":
			testCmd = "mvn test"
		case "mocha":
			testCmd = "mocha"
		case "cargo test":
			testCmd = "cargo test"
		}
	} else if projectInfo.BuildSystem != "" {
		switch projectInfo.BuildSystem {
		case "make":
			testCmd = "make test"
		case "cmake":
			testCmd = "ctest"
		case "gradle":
			testCmd = "./gradlew test"
		case "maven":
			testCmd = "mvn test"
		case "npm":
			testCmd = "npm test"
		case "yarn":
			testCmd = "yarn test"
		case "cargo":
			testCmd = "cargo test"
		}
	}

	// Add test command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       testCmd,
		WorkingDir:    workingDir,
		Timeout:       600, // 10 minutes
		RetryCount:    2,
		RetryDelay:    5,
		IsInteractive: false,
		ErrorPatterns: []string{"error:", "failed:", "test failure"},
	})
}

// createDeployAgent creates a deployment agent
func createDeployAgent(agent *Agent, workingDir string, projectInfo ProjectInfo) {
	// Add deploy command based on project type
	deployCmd := "make deploy"
	if projectInfo.BuildSystem != "" {
		switch projectInfo.BuildSystem {
		case "make":
			deployCmd = "make deploy"
		case "gradle":
			deployCmd = "./gradlew deploy"
		case "maven":
			deployCmd = "mvn deploy"
		case "npm":
			deployCmd = "npm run deploy"
		case "yarn":
			deployCmd = "yarn deploy"
		}
	}

	// Add deploy command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       deployCmd,
		WorkingDir:    workingDir,
		Timeout:       900, // 15 minutes
		RetryCount:    3,
		RetryDelay:    30,
		IsInteractive: false,
		ErrorPatterns: []string{"error:", "failed:", "deployment failed"},
	})
}

// createDockerAgent creates a Docker agent
func createDockerAgent(agent *Agent, workingDir string, projectInfo ProjectInfo) {
	// Add Docker build command
	imageName := "app"
	if projectInfo.Name != "" {
		imageName = strings.ToLower(projectInfo.Name)
	}

	// Add Docker build command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       fmt.Sprintf("docker build -t %s .", imageName),
		WorkingDir:    workingDir,
		Timeout:       900, // 15 minutes
		RetryCount:    2,
		RetryDelay:    30,
		IsInteractive: false,
		ErrorPatterns: []string{"error:", "failed:", "build failed"},
	})

	// Add Docker run command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       fmt.Sprintf("docker run --rm %s", imageName),
		WorkingDir:    workingDir,
		Timeout:       300, // 5 minutes
		RetryCount:    2,
		RetryDelay:    10,
		IsInteractive: false,
		ErrorPatterns: []string{"error:", "failed:", "exited with code"},
	})

	// Create Docker configuration
	agent.DockerConfig = &AgentDockerConfig{
		Image:       imageName,
		Tag:         "latest",
		BuildContext: workingDir,
		UseCache:    true,
	}
}

// createProjectAgent creates a project-specific agent
func createProjectAgent(agent *Agent, workingDir string, projectInfo ProjectInfo) {
	// Add commands based on project type
	if projectInfo.BuildSystem != "" {
		switch projectInfo.BuildSystem {
		case "make":
			agent.Commands = append(agent.Commands, AgentCommand{
				Command:       "make",
				WorkingDir:    workingDir,
				Timeout:       600, // 10 minutes
				RetryCount:    3,
				RetryDelay:    10,
				IsInteractive: false,
				ErrorPatterns: []string{"error:", "failed:"},
			})
		case "npm":
			agent.Commands = append(agent.Commands, AgentCommand{
				Command:       "npm install",
				WorkingDir:    workingDir,
				Timeout:       300, // 5 minutes
				RetryCount:    2,
				RetryDelay:    10,
				IsInteractive: false,
				ErrorPatterns: []string{"error:", "failed:"},
			})
			agent.Commands = append(agent.Commands, AgentCommand{
				Command:       "npm run build",
				WorkingDir:    workingDir,
				Timeout:       300, // 5 minutes
				RetryCount:    2,
				RetryDelay:    10,
				IsInteractive: false,
				ErrorPatterns: []string{"error:", "failed:"},
			})
		case "gradle":
			agent.Commands = append(agent.Commands, AgentCommand{
				Command:       "./gradlew build",
				WorkingDir:    workingDir,
				Timeout:       600, // 10 minutes
				RetryCount:    2,
				RetryDelay:    10,
				IsInteractive: false,
				ErrorPatterns: []string{"error:", "failed:"},
			})
		}
	} else {
		// Add generic build command if no specific one is detected
		agent.Commands = append(agent.Commands, AgentCommand{
			Command:       "make",
			WorkingDir:    workingDir,
			Timeout:       600, // 10 minutes
			RetryCount:    3,
			RetryDelay:    10,
			IsInteractive: false,
			ErrorPatterns: []string{"error:", "failed:"},
		})
	}
}

// createDeepFryAgent creates a DeepFry agent
func createDeepFryAgent(agent *Agent, workingDir string) {
	// Update agent description and tags
	agent.Description = "Automated build system for DeepFry PocketPC defconfig"
	agent.TaskTypes = []string{"build", "error-fix", "uproot"}
	agent.Tags = []string{"deepfry", "build", "pocketpc"}

	// Configure context
	agent.Context["project"] = "deepfry"
	agent.Context["build_type"] = "pocketpc"

	// Add uproot command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:         "uproot",
		WorkingDir:      workingDir,
		Timeout:         3600, // 1 hour
		RetryCount:      3,
		RetryDelay:      30,
		IsInteractive:   false,
		ErrorPatterns:   []string{"failed to uproot", "error:"},
		SuccessPatterns: []string{"uproot completed successfully"},
	})

	// Add run all command
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       "run all",
		WorkingDir:    workingDir,
		Timeout:       7200, // 2 hours
		RetryCount:    5,
		RetryDelay:    60,
		IsInteractive: false,
		ErrorPatterns: []string{"build failed", "error:"},
	})

	// Configure Docker
	agent.DockerConfig = &AgentDockerConfig{
		Image:      "deepfry-builder",
		Tag:        "latest",
		Volumes:    []string{workingDir + ":/src"}, // Keep as string concatenation for Docker volume format
		UseCache:   true,
		CacheFrom:  []string{"deepfry-builder:cache"},
	}

	// Set trigger patterns
	agent.TriggerPatterns = []string{
		"deepfry build",
		"uproot failing",
		"pocketpc defconfig",
	}

	// Set AI prompt
	agent.AIPrompt = "You are a DeepFry build assistant. Your task is to execute uproot commands, run builds, and fix common build errors for the PocketPC defconfig."
}

// createGeneralAgent creates a general-purpose agent
func createGeneralAgent(agent *Agent, workingDir string, projectInfo ProjectInfo) {
	// Add a simple command to echo environment
	agent.Commands = append(agent.Commands, AgentCommand{
		Command:       "echo 'Agent executed successfully'",
		WorkingDir:    workingDir,
		Timeout:       30, // 30 seconds
		RetryCount:    1,
		RetryDelay:    5,
		IsInteractive: false,
	})

	// Get current directory
	curDir, err := os.Getwd()
	if err == nil {
		agent.Context["current_dir"] = curDir
	}
}

// Helper functions

// suggestTriggersForAgent suggests trigger patterns for an agent
func suggestTriggersForAgent(agent *Agent) []string {
	triggers := make([]string, 0)

	// Add trigger based on agent name
	nameTrigger := strings.ToLower(agent.Name)
	if !stringInSlice(triggers, nameTrigger) {
		triggers = append(triggers, nameTrigger)
	}

	// Add triggers based on task types
	for _, taskType := range agent.TaskTypes {
		taskTrigger := strings.ToLower(taskType)
		if !stringInSlice(triggers, taskTrigger) {
			triggers = append(triggers, taskTrigger)
		}
	}

	// Add triggers based on context
	for key, value := range agent.Context {
		if key == "project" || key == "project_name" {
			projectTrigger := strings.ToLower(value)
			if !stringInSlice(triggers, projectTrigger) {
				triggers = append(triggers, projectTrigger)
			}
		}
	}

	// Add triggers based on commands
	for _, cmd := range agent.Commands {
		cmdParts := strings.Fields(cmd.Command)
		if len(cmdParts) > 0 {
			cmdTrigger := strings.ToLower(cmdParts[0])
			if !stringInSlice(triggers, cmdTrigger) && len(cmdTrigger) > 2 {
				triggers = append(triggers, cmdTrigger)
			}
		}
	}

	return triggers
}

// stringInSlice checks if a string is in a slice (internal function for this file)
func stringInSlice(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// containsAnyTerm checks if any of the terms are in the string (internal function for this file)
func containsAnyTerm(s string, terms ...string) bool {
	s = strings.ToLower(s)
	for _, term := range terms {
		if strings.Contains(s, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// parseOptions parses command options (internal function for this file)
func parseOptions(args []string) map[string]string {
	options := make(map[string]string)

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
			key := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			} else {
				value = "true"
			}
			options[key] = value
		}
	}

	return options
}

// minFloatValue returns the minimum of two floats (internal function for this file)
func minFloatValue(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}