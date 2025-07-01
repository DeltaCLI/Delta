package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CommandCategory represents a category of commands
type CommandCategory string

const (
	CategoryCore         CommandCategory = "Core Commands"
	CategoryAI           CommandCategory = "AI & Machine Learning"
	CategoryMemory       CommandCategory = "Memory & Learning"
	CategoryUtility      CommandCategory = "Utility Commands"
	CategoryDevelopment  CommandCategory = "Development Tools"
	CategorySystem       CommandCategory = "System Management"
	CategoryCollaboration CommandCategory = "Collaboration"
)

// CommandFlag represents a command-line flag
type CommandFlag struct {
	Name        string
	Short       string
	Description string
	Default     string
	Required    bool
}

// CommandExample represents a usage example
type CommandExample struct {
	Command     string
	Description string
	Output      string // Optional expected output
}

// CommandDoc represents structured documentation for a command
type CommandDoc struct {
	Name        string
	Category    CommandCategory
	Synopsis    string
	Description string
	Usage       string
	Flags       []CommandFlag
	Examples    []CommandExample
	SeeAlso     []string
	Since       string // Version when introduced
}

// CommandRegistry holds all command documentation
type CommandRegistry struct {
	Commands    map[string]*CommandDoc
	CLIFlags    []CommandFlag
	Version     string
	Description string
}

// Global command registry
var globalCommandRegistry *CommandRegistry

// InitializeCommandDocs initializes the command documentation
func InitializeCommandDocs() {
	globalCommandRegistry = &CommandRegistry{
		Commands:    make(map[string]*CommandDoc),
		Version:     getCurrentVersion(),
		Description: "Delta CLI - An AI-powered, context-aware shell enhancement that makes your terminal safer, smarter, and more intuitive.",
	}

	// Register CLI flags
	globalCommandRegistry.CLIFlags = []CommandFlag{
		{
			Name:        "command",
			Short:       "c",
			Description: "Execute a single command and exit",
			Required:    false,
		},
		{
			Name:        "version",
			Short:       "v",
			Description: "Show version information",
			Required:    false,
		},
		{
			Name:        "help",
			Short:       "h",
			Description: "Show help message",
			Required:    false,
		},
		{
			Name:        "debug",
			Short:       "d",
			Description: "Enable debug mode",
			Required:    false,
		},
	}

	// Register all commands
	registerCoreCommands()
	registerAICommands()
	registerMemoryCommands()
	registerUtilityCommands()
	registerSystemCommands()
}

func registerCoreCommands() {
	globalCommandRegistry.Register(&CommandDoc{
		Name:     "help",
		Category: CategoryCore,
		Synopsis: "Display help information",
		Description: `The help command displays comprehensive help information about Delta CLI
and its available commands. Without arguments, it shows a general overview.
With a command name, it shows detailed help for that specific command.`,
		Usage: ":help [command]",
		Examples: []CommandExample{
			{
				Command:     ":help",
				Description: "Show general help",
			},
			{
				Command:     ":help ai",
				Description: "Show help for the AI command",
			},
		},
		SeeAlso: []string{"version", "docs"},
		Since:   "v0.1.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "exit",
		Category: CategoryCore,
		Synopsis: "Exit Delta CLI",
		Description: `Exits the Delta CLI shell and returns to the parent shell.
Any unsaved state is automatically saved before exit.`,
		Usage: ":exit",
		Examples: []CommandExample{
			{
				Command:     ":exit",
				Description: "Exit Delta CLI",
			},
		},
		Since: "v0.1.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "version",
		Category: CategoryCore,
		Synopsis: "Show version information",
		Description: `Displays detailed version information including the Delta CLI version,
build date, git commit hash, and platform information.`,
		Usage: ":version",
		Examples: []CommandExample{
			{
				Command:     ":version",
				Description: "Show version details",
				Output:      "Delta CLI v0.4.6-alpha (2024-07-01, commit: abc123)",
			},
		},
		Since: "v0.1.0-alpha",
	})
}

func registerAICommands() {
	globalCommandRegistry.Register(&CommandDoc{
		Name:     "ai",
		Category: CategoryAI,
		Synopsis: "Manage AI features and predictions",
		Description: `The AI command controls Delta's AI-powered features including command
predictions, natural language processing, and intelligent suggestions.
Requires Ollama to be running with appropriate models installed.`,
		Usage: ":ai <subcommand> [options]",
		Examples: []CommandExample{
			{
				Command:     ":ai on",
				Description: "Enable AI predictions",
			},
			{
				Command:     ":ai off",
				Description: "Disable AI predictions",
			},
			{
				Command:     ":ai status",
				Description: "Show AI system status",
			},
			{
				Command:     ":ai model llama3.3:8b",
				Description: "Switch to a different AI model",
			},
		},
		SeeAlso: []string{"inference", "suggest"},
		Since:   "v0.2.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "suggest",
		Category: CategoryAI,
		Synopsis: "Get AI-powered command suggestions",
		Description: `Uses AI to suggest commands based on natural language descriptions.
The suggest command translates your intent into executable shell commands,
making it easier to accomplish tasks without memorizing syntax.`,
		Usage: ":suggest <description>",
		Examples: []CommandExample{
			{
				Command:     ":suggest find large files in home directory",
				Description: "Get command to find large files",
				Output:      "find ~ -type f -size +100M",
			},
			{
				Command:     ":suggest compress all jpg files",
				Description: "Get compression command",
				Output:      "find . -name '*.jpg' -exec jpegoptim {} \\;",
			},
		},
		SeeAlso: []string{"ai", "inference"},
		Since:   "v0.3.0-alpha",
	})
}

func registerMemoryCommands() {
	globalCommandRegistry.Register(&CommandDoc{
		Name:     "memory",
		Category: CategoryMemory,
		Synopsis: "Manage command history and learning",
		Description: `The memory command manages Delta's learning system, which collects
and analyzes command patterns to improve predictions and suggestions.
All data is stored locally and privacy settings can be configured.`,
		Usage: ":memory <subcommand> [options]",
		Examples: []CommandExample{
			{
				Command:     ":memory status",
				Description: "Show memory system status",
			},
			{
				Command:     ":memory clear",
				Description: "Clear collected command history",
			},
			{
				Command:     ":memory config privacy high",
				Description: "Set privacy level to high",
			},
			{
				Command:     ":memory export",
				Description: "Export collected data",
			},
		},
		SeeAlso: []string{"history", "pattern"},
		Since:   "v0.2.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "history",
		Category: CategoryMemory,
		Synopsis: "Analyze command history",
		Description: `Provides advanced analysis of your command history including usage
patterns, frequency analysis, and trend detection. Helps identify
opportunities for automation and improvement.`,
		Usage: ":history <subcommand> [options]",
		Examples: []CommandExample{
			{
				Command:     ":history stats",
				Description: "Show command statistics",
			},
			{
				Command:     ":history top 10",
				Description: "Show top 10 most used commands",
			},
			{
				Command:     ":history search git",
				Description: "Search history for git commands",
			},
		},
		SeeAlso: []string{"memory", "pattern"},
		Since:   "v0.3.0-alpha",
	})
}

func registerUtilityCommands() {
	globalCommandRegistry.Register(&CommandDoc{
		Name:     "jump",
		Category: CategoryUtility,
		Synopsis: "Smart directory navigation",
		Description: `Jump provides intelligent directory navigation by learning from your
usage patterns. It allows quick navigation to frequently used directories
using partial matches and fuzzy search.`,
		Usage: ":jump <pattern>",
		Examples: []CommandExample{
			{
				Command:     ":jump proj",
				Description: "Jump to a directory matching 'proj'",
			},
			{
				Command:     ":jump --list",
				Description: "List all jump targets",
			},
			{
				Command:     ":jump --add /path/to/dir",
				Description: "Add a directory to jump database",
			},
		},
		SeeAlso: []string{"cd"},
		Since:   "v0.1.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "validation",
		Category: CategoryUtility,
		Synopsis: "Command validation and safety checking",
		Description: `The validation system provides real-time syntax checking and safety
analysis for commands before execution. It can detect dangerous patterns,
suggest safer alternatives, and prevent accidental data loss.`,
		Usage: ":validation <subcommand> [options]",
		Examples: []CommandExample{
			{
				Command:     ":validation check rm -rf /",
				Description: "Check if a command is safe",
			},
			{
				Command:     ":validation stats",
				Description: "Show validation statistics",
			},
			{
				Command:     ":validation config interactive true",
				Description: "Enable interactive safety prompts",
			},
		},
		SeeAlso: []string{"validate"},
		Since:   "v0.4.5-alpha",
	})
}

func registerSystemCommands() {
	globalCommandRegistry.Register(&CommandDoc{
		Name:     "update",
		Category: CategorySystem,
		Synopsis: "Manage Delta CLI updates",
		Description: `The update command manages Delta CLI's auto-update system. It can check
for new versions, download updates, and manage update preferences. Updates
are verified with SHA256 checksums and include automatic rollback on failure.`,
		Usage: ":update <subcommand> [options]",
		Examples: []CommandExample{
			{
				Command:     ":update check",
				Description: "Check for available updates",
			},
			{
				Command:     ":update install",
				Description: "Install available updates",
			},
			{
				Command:     ":update config channel beta",
				Description: "Switch to beta update channel",
			},
			{
				Command:     ":update history",
				Description: "Show update history",
			},
		},
		SeeAlso: []string{"version", "config"},
		Since:   "v0.3.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "config",
		Category: CategorySystem,
		Synopsis: "Manage Delta CLI configuration",
		Description: `The config command provides access to Delta's configuration system.
Settings can be viewed, modified, exported, and imported. Configuration
is stored in ~/.config/delta/ and can be overridden with environment variables.`,
		Usage: ":config <subcommand> [key] [value]",
		Examples: []CommandExample{
			{
				Command:     ":config show",
				Description: "Show all configuration",
			},
			{
				Command:     ":config set theme dark",
				Description: "Set the theme to dark",
			},
			{
				Command:     ":config export config.json",
				Description: "Export configuration to file",
			},
		},
		SeeAlso: []string{"update", "i18n"},
		Since:   "v0.2.0-alpha",
	})

	globalCommandRegistry.Register(&CommandDoc{
		Name:     "i18n",
		Category: CategorySystem,
		Synopsis: "Internationalization settings",
		Description: `Manages language and localization settings for Delta CLI. Supports
over 11 languages with advanced pluralization rules. Language files are
automatically downloaded when switching languages.`,
		Usage: ":i18n <subcommand> [options]",
		Examples: []CommandExample{
			{
				Command:     ":i18n list",
				Description: "List available languages",
			},
			{
				Command:     ":i18n locale es",
				Description: "Switch to Spanish",
			},
			{
				Command:     ":i18n install",
				Description: "Install/update language files",
			},
		},
		SeeAlso: []string{"config"},
		Since:   "v0.2.0-alpha",
	})
}

// Register adds a command to the registry
func (r *CommandRegistry) Register(doc *CommandDoc) {
	r.Commands[doc.Name] = doc
}

// GetCommand retrieves a command documentation
func (r *CommandRegistry) GetCommand(name string) (*CommandDoc, bool) {
	doc, exists := r.Commands[name]
	return doc, exists
}

// GetCommandsByCategory returns all commands in a category
func (r *CommandRegistry) GetCommandsByCategory(category CommandCategory) []*CommandDoc {
	var commands []*CommandDoc
	for _, doc := range r.Commands {
		if doc.Category == category {
			commands = append(commands, doc)
		}
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})
	return commands
}

// GetAllCategories returns all command categories in order
func (r *CommandRegistry) GetAllCategories() []CommandCategory {
	return []CommandCategory{
		CategoryCore,
		CategoryAI,
		CategoryMemory,
		CategoryUtility,
		CategoryDevelopment,
		CategorySystem,
		CategoryCollaboration,
	}
}

// FormatCommandHelp formats help text for a specific command
func (doc *CommandDoc) FormatCommandHelp() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Command: :%s\n", doc.Name))
	builder.WriteString(fmt.Sprintf("Category: %s\n", doc.Category))
	builder.WriteString(fmt.Sprintf("\n%s\n", doc.Description))
	builder.WriteString(fmt.Sprintf("\nUsage: %s\n", doc.Usage))

	if len(doc.Flags) > 0 {
		builder.WriteString("\nOptions:\n")
		for _, flag := range doc.Flags {
			if flag.Short != "" {
				builder.WriteString(fmt.Sprintf("  -%s, --%s", flag.Short, flag.Name))
			} else {
				builder.WriteString(fmt.Sprintf("  --%s", flag.Name))
			}
			builder.WriteString(fmt.Sprintf("\t%s", flag.Description))
			if flag.Default != "" {
				builder.WriteString(fmt.Sprintf(" (default: %s)", flag.Default))
			}
			if flag.Required {
				builder.WriteString(" [required]")
			}
			builder.WriteString("\n")
		}
	}

	if len(doc.Examples) > 0 {
		builder.WriteString("\nExamples:\n")
		for _, example := range doc.Examples {
			builder.WriteString(fmt.Sprintf("  %s\n", example.Command))
			if example.Description != "" {
				builder.WriteString(fmt.Sprintf("    %s\n", example.Description))
			}
			if example.Output != "" {
				builder.WriteString(fmt.Sprintf("    Output: %s\n", example.Output))
			}
		}
	}

	if len(doc.SeeAlso) > 0 {
		builder.WriteString("\nSee also: ")
		for i, cmd := range doc.SeeAlso {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf(":%s", cmd))
		}
		builder.WriteString("\n")
	}

	if doc.Since != "" {
		builder.WriteString(fmt.Sprintf("\nAvailable since: %s\n", doc.Since))
	}

	return builder.String()
}

// GetManPageDate returns the current date in man page format
func GetManPageDate() string {
	return time.Now().Format("January 2006")
}