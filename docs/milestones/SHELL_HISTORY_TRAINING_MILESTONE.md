# Shell History Training Milestone

## Overview
Implement functionality to detect, parse, and train on existing bash/zsh history files from the user's home directory to enhance Delta's command prediction and pattern recognition capabilities.

## Current State
- Delta only tracks commands executed within its own shell interface
- No integration with existing shell history files (.bash_history, .zsh_history)
- Training data limited to Delta's internal command feedback system

## Milestone Goals

### Phase 1: Detection and Discovery
- [ ] Implement shell history file detection in user's home directory
- [ ] Support common history file locations:
  - `~/.bash_history`
  - `~/.zsh_history` 
  - `~/.history`
  - Custom HISTFILE locations from shell configuration

### Phase 2: Parsing and Processing
- [ ] Create parser for bash history format (simple newline-separated commands)
- [ ] Create parser for zsh history format (timestamp + command format)
- [ ] Handle extended history formats with timestamps and session info
- [ ] Sanitize and validate history entries

### Phase 3: User Interaction
- [ ] Add interactive prompt asking user permission to train on existing history
- [ ] Provide options to:
  - Train on all history
  - Train on recent history (last N commands)
  - Skip certain commands/patterns
  - Review commands before training
- [ ] Show statistics about discovered history files

### Phase 4: Training Integration
- [ ] Convert shell history entries to Delta's training data format
- [ ] Integrate with existing training pipeline in `training_commands.go`
- [ ] Preserve command context and frequency information
- [ ] Update knowledge extraction to include shell history patterns

## Technical Implementation

### New Files/Components
1. `shell_history_detector.go` - Detect and locate shell history files
2. `shell_history_parser.go` - Parse different history file formats
3. `shell_history_trainer.go` - Convert history to training data
4. New command: `delta history import` - Interactive import workflow

### Modified Files
- `training_commands.go` - Add history import command
- `knowledge_extractor.go` - Include shell history patterns
- `cli.go` - Add new command registration

### Command Interface
```
delta history import [options]
  --auto-detect    Automatically detect and import all found history files
  --file PATH      Import specific history file
  --limit N        Only import last N commands
  --interactive    Interactive mode with user confirmation
  --dry-run        Show what would be imported without actually doing it
```

## Success Criteria
- [ ] Successfully detect shell history files in user's home directory
- [ ] Parse bash and zsh history formats correctly
- [ ] Provide user-friendly import workflow with clear permissions
- [ ] Integrate imported history into Delta's training pipeline
- [ ] Show measurable improvement in command prediction accuracy
- [ ] Maintain user privacy and control over imported data

## Privacy Considerations
- Always ask user permission before accessing history files
- Provide options to exclude sensitive commands
- Allow user to review commands before import
- Respect shell history settings (HISTIGNORE patterns)
- Store imported data securely within Delta's data directory

## Timeline
- Phase 1-2: Core detection and parsing functionality
- Phase 3: User interaction and permission system
- Phase 4: Training pipeline integration and testing

## Testing Strategy
- Unit tests for history file parsers
- Integration tests with sample history files
- Manual testing with real user history files
- Privacy and permission flow testing