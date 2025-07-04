package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

// ManPageGenerator generates Unix man pages from command documentation
type ManPageGenerator struct {
	Registry *CommandRegistry
	Version  string
}

// NewManPageGenerator creates a new man page generator
func NewManPageGenerator() *ManPageGenerator {
	if globalCommandRegistry == nil {
		InitializeCommandDocs()
	}
	return &ManPageGenerator{
		Registry: globalCommandRegistry,
		Version:  getCurrentVersion(),
	}
}

// GenerateMainManPage generates the main delta.1 man page
func (g *ManPageGenerator) GenerateMainManPage() (string, error) {
	tmpl := `.\\" Manpage for delta
.\\" Contact: support@deltacli.dev
.TH DELTA 1 "{{.Date}}" "{{.Version}}" "Delta CLI Manual"
.SH NAME
delta \- AI-powered, context-aware shell enhancement
.SH SYNOPSIS
.B delta
[\fI\,OPTIONS\/\fR] [\fI\,COMMAND\/\fR]
.SH DESCRIPTION
Delta CLI is an AI-powered shell enhancement that makes your terminal safer, smarter, and more intuitive.
It provides intelligent command suggestions, safety validation, and enhanced navigation features
while maintaining full compatibility with your existing shell.
.PP
Delta operates as an interactive shell wrapper that intercepts commands, provides AI-powered
suggestions, and adds safety checks before execution. All features are optional and can be
configured to match your workflow.
.SH OPTIONS
{{range .CLIFlags -}}
.TP
{{if .Short -}}
\fB\-{{.Short}}\fR, \fB\-\-{{.Name}}\fR
{{- else -}}
\fB\-\-{{.Name}}\fR
{{- end}}
{{.Description}}{{if .Default}} (default: {{.Default}}){{end}}
{{end}}
.SH COMMANDS
Delta uses a colon prefix (:) for internal commands to distinguish them from shell commands.
.PP
{{range .Categories -}}
.SS {{.}}
{{range $.GetCommandsByCategory . -}}
.TP
.B :{{.Name}}
{{.Synopsis}}
{{end}}
{{end}}
.SH EXAMPLES
.TP
.B delta
Start Delta in interactive mode
.TP
.B delta -c "ls -la"
Execute a single command and exit
.TP
.B delta -c ":ai on"
Enable AI features and exit
.TP
.B echo "git status" | delta
Run a command through Delta (legacy method)
.SH ENVIRONMENT
.TP
.B DELTA_CONFIG_DIR
Override the default configuration directory (default: ~/.config/delta)
.TP
.B DELTA_LOCALE
Set the interface language (e.g., en, es, fr, de, ja)
.TP
.B DELTA_UPDATE_ENABLED
Enable or disable automatic update checks (true/false)
.TP
.B DELTA_AI_MODEL
Override the default AI model for predictions
.TP
.B SHELL
The shell to use for command execution (default: current shell)
.SH FILES
.TP
.I ~/.config/delta/
Configuration directory containing all Delta settings and data
.TP
.I ~/.config/delta/system_config.json
Main configuration file
.TP
.I ~/.config/delta/memory.db
Command history and learning database
.TP
.I ~/.config/delta/i18n/locales/
Internationalization files for different languages
.TP
.I ~/.config/delta/logs/
Log files for debugging and audit trails
.SH EXIT STATUS
Delta exits with the status of the last executed command. Special exit codes:
.TP
.B 0
Success or normal exit
.TP
.B 1
General error or command failure
.TP
.B 2
Configuration or initialization error
.TP
.B 130
Interrupted by Ctrl+C (SIGINT)
.SH SECURITY
Delta implements multiple security features:
.PP
- Command validation with risk assessment
.br
- Interactive safety prompts for dangerous operations
.br
- Automatic rollback for failed updates
.br
- SHA256 verification for all downloads
.br
- Local-only AI processing (no cloud dependencies)
.br
- Configurable privacy settings
.SH SEE ALSO
Full documentation is available at https://delta.dev/docs
.PP
Report bugs to: https://github.com/DeltaCLI/delta/issues
.SH AUTHORS
Delta CLI is developed by the Delta Team and the open source community.
.PP
This manual page was automatically generated from Delta's command documentation.
`

	data := struct {
		Date                 string
		Version              string
		CLIFlags             []CommandFlag
		Categories           []CommandCategory
		GetCommandsByCategory func(CommandCategory) []*CommandDoc
	}{
		Date:                 GetManPageDate(),
		Version:              g.Version,
		CLIFlags:             g.Registry.CLIFlags,
		Categories:           g.Registry.GetAllCategories(),
		GetCommandsByCategory: g.Registry.GetCommandsByCategory,
	}

	t, err := template.New("manpage").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var builder strings.Builder
	if err := t.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

// GenerateCommandManPage generates a man page for a specific command
func (g *ManPageGenerator) GenerateCommandManPage(commandName string) (string, error) {
	doc, exists := g.Registry.GetCommand(commandName)
	if !exists {
		return "", fmt.Errorf("command not found: %s", commandName)
	}

	tmpl := `.\\" Manpage for delta {{.Name}} command
.\\" Contact: support@deltacli.dev
.TH DELTA-{{.UpperName}} 1 "{{.Date}}" "{{.Version}}" "Delta CLI Manual"
.SH NAME
delta\-{{.Name}} \- {{.Synopsis}}
.SH SYNOPSIS
.B {{.Usage}}
.SH DESCRIPTION
{{.FormattedDescription}}
{{if .Flags -}}
.SH OPTIONS
{{range .Flags -}}
.TP
{{if .Short -}}
\fB\-{{.Short}}\fR, \fB\-\-{{.Name}}\fR
{{- else -}}
\fB\-\-{{.Name}}\fR
{{- end}}
{{.Description}}{{if .Default}} (default: {{.Default}}){{end}}{{if .Required}} [required]{{end}}
{{end}}
{{- end}}
{{if .Examples -}}
.SH EXAMPLES
{{range .Examples -}}
.TP
.B {{.Command}}
{{.Description}}
{{if .Output -}}
.br
Output: {{.Output}}
{{- end}}
{{end}}
{{- end}}
{{if .SeeAlso -}}
.SH SEE ALSO
{{range $i, $cmd := .SeeAlso -}}
{{if $i}}, {{end}}.BR delta\-{{$cmd}} (1)
{{- end}}
{{- end}}
.SH AVAILABILITY
Available since Delta CLI {{.Since}}
.SH AUTHORS
Delta CLI is developed by the Delta Team and the open source community.
`

	// Format description for man page (break long lines)
	formattedDesc := formatDescriptionForMan(doc.Description)

	data := struct {
		*CommandDoc
		Date                 string
		Version              string
		UpperName            string
		FormattedDescription string
	}{
		CommandDoc:           doc,
		Date:                 GetManPageDate(),
		Version:              g.Version,
		UpperName:            strings.ToUpper(doc.Name),
		FormattedDescription: formattedDesc,
	}

	t, err := template.New("command").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var builder strings.Builder
	if err := t.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

// GenerateAllManPages generates all man pages and saves them to a directory
func (g *ManPageGenerator) GenerateAllManPages(outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate main man page
	mainContent, err := g.GenerateMainManPage()
	if err != nil {
		return fmt.Errorf("failed to generate main man page: %w", err)
	}

	mainPath := filepath.Join(outputDir, "delta.1")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("failed to write main man page: %w", err)
	}

	// Generate command-specific man pages for complex commands
	complexCommands := []string{"ai", "update", "memory", "config", "validation"}
	for _, cmd := range complexCommands {
		content, err := g.GenerateCommandManPage(cmd)
		if err != nil {
			// Skip if command doesn't exist
			continue
		}

		cmdPath := filepath.Join(outputDir, fmt.Sprintf("delta-%s.1", cmd))
		if err := os.WriteFile(cmdPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write man page for %s: %w", cmd, err)
		}
	}

	return nil
}

// InstallManPages installs man pages to the system
func (g *ManPageGenerator) InstallManPages(manDir string) error {
	// Default to /usr/local/share/man/man1 if not specified
	if manDir == "" {
		manDir = "/usr/local/share/man/man1"
	}

	// Check if we have permission to write
	if err := os.MkdirAll(manDir, 0755); err != nil {
		return fmt.Errorf("failed to create man directory (may need sudo): %w", err)
	}

	// Generate to temporary directory first
	tempDir := filepath.Join(os.TempDir(), "delta-man-"+time.Now().Format("20060102150405"))
	if err := g.GenerateAllManPages(tempDir); err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Copy files to man directory
	files, err := filepath.Glob(filepath.Join(tempDir, "*.1"))
	if err != nil {
		return fmt.Errorf("failed to list man pages: %w", err)
	}

	for _, file := range files {
		basename := filepath.Base(file)
		dest := filepath.Join(manDir, basename)

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("failed to install %s (may need sudo): %w", basename, err)
		}
	}

	// Update man database if mandb is available
	if _, err := os.Stat("/usr/bin/mandb"); err == nil {
		// Run mandb to update the database
		// Note: In a real implementation, we'd use os/exec to run this
		fmt.Println("Run 'sudo mandb' to update the man page database")
	}

	return nil
}

// PreviewManPage generates and displays a man page preview
func (g *ManPageGenerator) PreviewManPage(commandName string) error {
	var content string
	var err error

	if commandName == "" || commandName == "delta" {
		content, err = g.GenerateMainManPage()
	} else {
		content, err = g.GenerateCommandManPage(commandName)
	}

	if err != nil {
		return err
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "delta-man-*.1")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Display using man command
	// Note: In a real implementation, we'd use os/exec to run this
	fmt.Printf("Preview: man %s\n", tmpFile.Name())
	fmt.Println("\nRaw content:")
	fmt.Println(content)

	return nil
}

// formatDescriptionForMan formats description text for man page display
func formatDescriptionForMan(desc string) string {
	// Replace newlines with man page line breaks
	desc = strings.ReplaceAll(desc, "\n", "\n.br\n")
	
	// Ensure sentences end with proper spacing
	desc = strings.ReplaceAll(desc, ". ", ".\n.br\n")
	
	return desc
}

// GenerateCompletions generates shell completion files from command docs
func (g *ManPageGenerator) GenerateCompletions(shell string) (string, error) {
	switch shell {
	case "bash":
		return g.generateBashCompletions()
	case "zsh":
		return g.generateZshCompletions()
	case "fish":
		return g.generateFishCompletions()
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

func (g *ManPageGenerator) generateBashCompletions() (string, error) {
	var cmds []string
	for name := range g.Registry.Commands {
		cmds = append(cmds, ":"+name)
	}
	sort.Strings(cmds)

	completion := fmt.Sprintf(`# Delta CLI Bash Completion
# Generated on %s

_delta_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # CLI flags
    opts="--help --version --command --debug"
    
    # Internal commands
    commands="%s"
    
    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    elif [[ ${cur} == :* ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
    fi
    
    return 0
}

complete -F _delta_completions delta
`, time.Now().Format("2006-01-02"), strings.Join(cmds, " "))

	return completion, nil
}

func (g *ManPageGenerator) generateZshCompletions() (string, error) {
	var cmds []string
	var cmdDescriptions []string
	
	// Build list of commands with descriptions
	for name, doc := range g.Registry.Commands {
		cmds = append(cmds, ":"+name)
		// Get first line of description
		desc := doc.Synopsis
		if desc == "" && doc.Description != "" {
			lines := strings.Split(doc.Description, "\n")
			if len(lines) > 0 {
				desc = strings.TrimSpace(lines[0])
			}
		}
		// Escape single quotes in description
		desc = strings.ReplaceAll(desc, "'", "'\"'\"'")
		cmdDescriptions = append(cmdDescriptions, fmt.Sprintf("':%s:%s'", name, desc))
	}
	sort.Strings(cmds)
	sort.Strings(cmdDescriptions)

	// For now, we'll handle subcommands manually for known commands
	// This can be enhanced later when subcommands are added to CommandDoc
	subcommandMap := map[string][]string{
		"ai": {"on", "off", "status", "model", "feedback"},
		"memory": {"enable", "disable", "status", "stats", "config", "list", "export", "clear", "train"},
		"learning": {"enable", "disable", "status", "feedback", "train", "patterns", "process", "stats", "config"},
		"update": {"status", "check", "install", "config", "history", "channel"},
		"validation": {"check", "safety", "stats", "history", "config"},
		"man": {"generate", "preview", "install", "view", "completions"},
	}
	
	var subcommandCases []string
	for cmd, subs := range subcommandMap {
		var subList []string
		for _, sub := range subs {
			subList = append(subList, fmt.Sprintf("'%s:%s'", sub, sub))
		}
		subcommandCases = append(subcommandCases, fmt.Sprintf(`      %s)
        _arguments \
          '1: :->subcmd' \
          '*:: :->args' && return 0
        case $state in
          subcmd)
            local subcmds=(
              %s
            )
            _describe -t subcmds 'subcommand' subcmds && return 0
            ;;
        esac
        ;;`, cmd, strings.Join(subList, "\n              ")))
	}

	completion := fmt.Sprintf(`#compdef delta
# Delta CLI Zsh Completion
# Generated on %s

_delta() {
  local context state state_descr line
  typeset -A opt_args

  _arguments -C \
    '(-h --help)'{-h,--help}'[Show help information]' \
    '(-v --version)'{-v,--version}'[Show version information]' \
    '(-c --command)'{-c,--command}'[Execute a single command and exit]:command:_command_names' \
    '--debug[Enable debug mode]' \
    '1: :->cmd' \
    '*:: :->args' && return 0

  case $state in
    cmd)
      if [[ "$words[1]" == ":" || "$words[1]" == ":*" ]]; then
        # Internal Delta commands
        local commands=(
          %s
        )
        _describe -t commands 'Delta command' commands && return 0
      else
        # External shell commands
        _command_names && return 0
      fi
      ;;
    args)
      # Handle subcommands for specific Delta commands
      local cmd="${words[1]#:}"
      case $cmd in
%s
        *)
          # Default file completion for other commands
          _files
          ;;
      esac
      ;;
  esac

  return 1
}

# Helper function for internal commands
_delta_commands() {
  local commands=(
    %s
  )
  _describe -t commands 'Delta command' commands
}

_delta "$@"
`, 
		time.Now().Format("2006-01-02"),
		strings.Join(cmdDescriptions, "\n    "),
		strings.Join(subcommandCases, "\n"),
		strings.Join(cmdDescriptions, "\n    "))

	return completion, nil
}

func (g *ManPageGenerator) generateFishCompletions() (string, error) {
	var completions []string
	
	// Add basic flags
	completions = append(completions,
		"# Delta CLI Fish Completion",
		fmt.Sprintf("# Generated on %s", time.Now().Format("2006-01-02")),
		"",
		"# Disable file completion by default",
		"complete -c delta -f",
		"",
		"# Global flags", 
		"complete -c delta -s h -l help -d 'Show help information'",
		"complete -c delta -s v -l version -d 'Show version information'",
		"complete -c delta -s c -l command -d 'Execute a single command and exit' -r",
		"complete -c delta -l debug -d 'Enable debug mode'",
		"",
		"# Enable file completion for external commands",
		"complete -c delta -n '__fish_is_first_token; and not __fish_seen_colon_command' -F",
		"",
		"# Internal Delta commands",
	)
	
	// Get all commands sorted
	var cmdNames []string
	for name := range g.Registry.Commands {
		cmdNames = append(cmdNames, name)
	}
	sort.Strings(cmdNames)
	
	// Add main commands
	for _, name := range cmdNames {
		doc := g.Registry.Commands[name]
		desc := doc.Synopsis
		if desc == "" && doc.Description != "" {
			lines := strings.Split(doc.Description, "\n")
			if len(lines) > 0 {
				desc = strings.TrimSpace(lines[0])
			}
		}
		// Escape single quotes for fish
		desc = strings.ReplaceAll(desc, "'", "\\'")
		
		// Add the main command
		completions = append(completions,
			fmt.Sprintf("complete -c delta -n '__fish_use_subcommand' -a ':%s' -d '%s'", name, desc))
		
		// Add subcommands based on our manual mapping
		subcommandMap := map[string][]string{
			"ai": {"on", "off", "status", "model", "feedback"},
			"memory": {"enable", "disable", "status", "stats", "config", "list", "export", "clear", "train"},
			"learning": {"enable", "disable", "status", "feedback", "train", "patterns", "process", "stats", "config"},
			"update": {"status", "check", "install", "config", "history", "channel"},
			"validation": {"check", "safety", "stats", "history", "config"},
			"man": {"generate", "preview", "install", "view", "completions"},
		}
		
		if subs, hasSubs := subcommandMap[name]; hasSubs {
			completions = append(completions, "")
			completions = append(completions, fmt.Sprintf("# Subcommands for :%s", name))
			
			for _, sub := range subs {
				condition := fmt.Sprintf("__fish_seen_subcommand_from :%s; and not __fish_seen_subcommand_from %s", 
					name, strings.Join(subs, " "))
				
				completions = append(completions,
					fmt.Sprintf("complete -c delta -n '%s' -a '%s' -d 'Subcommand for %s'", 
						condition, sub, name))
			}
		}
		
		// Add flag completions for commands that have them
		if len(doc.Flags) > 0 {
			completions = append(completions, "")
			completions = append(completions, fmt.Sprintf("# Flags for :%s", name))
			
			for _, flag := range doc.Flags {
				flagDesc := strings.ReplaceAll(flag.Description, "'", "\\'")
				condition := fmt.Sprintf("__fish_seen_subcommand_from :%s", name)
				
				if flag.Short != "" && flag.Name != "" {
					completions = append(completions,
						fmt.Sprintf("complete -c delta -n '%s' -s %s -l %s -d '%s'",
							condition, flag.Short, flag.Name, flagDesc))
				} else if flag.Name != "" {
					completions = append(completions,
						fmt.Sprintf("complete -c delta -n '%s' -l %s -d '%s'",
							condition, flag.Name, flagDesc))
				} else if flag.Short != "" {
					completions = append(completions,
						fmt.Sprintf("complete -c delta -n '%s' -s %s -d '%s'",
							condition, flag.Short, flagDesc))
				}
			}
		}
	}
	
	// Add help function for detecting subcommands
	completions = append(completions, "", "# Helper functions", `
# Check if we're completing the first argument
function __fish_use_subcommand
    set -l cmd (commandline -opc)
    test (count $cmd) -eq 1
end

# Check if a colon command has been used
function __fish_seen_colon_command
    set -l cmd (commandline -opc)
    for c in $cmd
        if string match -q ':*' -- $c
            return 0
        end
    end
    return 1
end`)

	return strings.Join(completions, "\n"), nil
}

