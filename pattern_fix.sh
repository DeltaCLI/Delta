#!/bin/bash

# Read all lines before the problematic section
head -n 613 /home/bleepbloop/deltacli/cli.go > /tmp/cli_fixed.go

# Add our fixed section
cat << 'EOF' >> /tmp/cli_fixed.go
		case "knowledge", "know":
			return HandleKnowledgeCommand(args)
		case "agent":
			return HandleAgentCommand(args)
		case "config":
			return HandleConfigCommand(args)
		case "pattern", "pat":
			return HandlePatternCommand(args)
		case "spellcheck", "spell":
			return HandleSpellCheckCommand(args)
		case "history", "hist":
			return HandleHistoryCommand(args)
EOF

# Read all lines after the problematic section
tail -n +625 /home/bleepbloop/deltacli/cli.go >> /tmp/cli_fixed.go

# Replace the original file
mv /tmp/cli_fixed.go /home/bleepbloop/deltacli/cli.go