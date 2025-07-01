# Release Notes for v0.4.7-alpha

## 🚀 Highlights

Delta CLI v0.4.7-alpha introduces **man page generation** for Unix systems, allowing you to access Delta's documentation through the standard `man` command. We've also fixed the startup experience by eliminating misleading "connection restored" messages.

## 📦 What's New

### Man Page Generation System
- **Standard Unix Documentation**: Generate and install man pages for Delta CLI
- **`:man` Command**: New command for managing documentation
  - `:man generate` - Create man pages in troff format
  - `:man preview` - Preview before installing
  - `:man install` - Install to system (may need sudo)
  - `:man view` - View installed pages
  - `:man completions` - Generate shell completions
- **Build Integration**: `make man` and PowerShell equivalents
- **Structured Docs**: Consistent command documentation across all features

### Improved Startup Experience
- Fixed misleading "Ollama connection restored" messages on boot
- Connection status now only shows for actual state changes
- Better first-run experience with accurate status reporting

### Repository Cleanup
- Moved internal planning documents to private repository
- Cleaner public repository focused on production code

## 💻 Usage Examples

### Generate and Install Man Pages
```bash
# Generate man pages
delta :man generate

# Preview before installing
delta :man preview ai

# Install system-wide
sudo delta :man install

# View the documentation
man delta
man delta-ai
```

### Shell Completions
```bash
# Generate bash completions
delta :man completions bash > ~/.delta-completion.bash
source ~/.delta-completion.bash
```

### Build System
```bash
# Using make
make man
sudo make install-man

# Using PowerShell
.\build.ps1 man
```

## 🔧 Technical Details

- Man pages follow standard troff format
- Compatible with `man`, `apropos`, and `whatis` commands
- Structured command documentation ensures consistency
- First-check detection prevents false connection messages

## 📋 Requirements

- For man pages: Unix-like system with man-db or similar
- For shell completions: bash (zsh/fish support coming soon)
- No changes to existing Delta CLI requirements

## 🐛 Bug Fixes

- Fixed "Ollama server connection restored" showing on every startup
- Improved connection status accuracy for AI features

## 📝 Notes

- Man page installation may require sudo privileges
- Windows users can generate man pages for use on Unix systems
- Shell completions currently support bash only

---

**Full Changelog**: https://github.com/DeltaCLI/delta/compare/v0.4.6-alpha...v0.4.7-alpha