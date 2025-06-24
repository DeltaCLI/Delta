# Delta CLI v0.4.6-alpha Release Notes

## 🎉 New Features

### Natural Language Command Suggestions 🗣️
Delta now understands plain English! Simply describe what you want to do, and Delta will suggest the right commands.

- **New `:suggest` command** (alias `:s`) - Get command suggestions from natural language
- **Smart Pattern Matching** - Recognizes common command patterns for files, git, docker, and more
- **Project-Aware** - Detects your project type and suggests relevant commands
- **AI-Powered** - Enhanced suggestions when Ollama is enabled
- **History Learning** - Learns from your command usage patterns
- **Interactive Selection** - Choose commands with safety indicators (✓ safe, ⚡ caution, ⚠️ dangerous)
- **Command Explanations** - Use `:suggest explain <command>` to understand what commands do

#### Examples:
- `:suggest list all files` → `ls -la`, `ls -lh`, `tree`
- `:suggest install dependencies` → `npm install`, `go mod download`, `pip install -r requirements.txt`
- `:suggest create new git branch` → `git checkout -b new-branch`
- `:suggest find text in files` → `grep -r "text" .`, `rg "pattern"`

### Ollama Health Monitoring 🏥
Never miss when Ollama becomes available! Delta now monitors Ollama connectivity in the background.

- **Automatic Detection** - Periodically checks if Ollama is available
- **Smart Notifications** - Alerts you when Ollama comes online
- **Configurable Monitoring** - Adjust check intervals and notification preferences
- **New Health Commands**:
  - `:ai health` - View monitoring status
  - `:ai health monitor on/off` - Enable/disable monitoring
  - `:ai health interval <seconds>` - Set check frequency
  - `:ai health notify on/off` - Toggle notifications

## 🔧 Improvements

- Enhanced AI status display with health monitoring information
- Better error handling for AI features
- Improved help system with new command documentation

## 📦 Installation

### Download Binaries
Pre-built binaries are available for:
- Linux (amd64)
- macOS (Intel & Apple Silicon)
- Windows (amd64)

### Build from Source
```bash
git clone https://github.com/sourcegraph/delta.git
cd delta/deltacli
make build
```

## 🙏 Acknowledgments

Thanks to all contributors and users who provided feedback for this release!

---

**Full Changelog**: https://github.com/sourcegraph/delta/compare/v0.4.5-alpha...v0.4.6-alpha