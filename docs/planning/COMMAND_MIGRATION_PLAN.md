# Command Migration Plan

## Overview
This document outlines the plan to migrate all Delta CLI command handlers into a centralized `cmds/` directory for better organization and maintainability.

## Goals
1. **Better Organization**: Group all command implementations in one place
2. **Easier Navigation**: Find commands quickly without searching through the root directory
3. **Cleaner Root**: Reduce clutter in the project root
4. **Consistent Structure**: All commands follow the same pattern and location

## Current Command Files to Migrate

### Priority 1 - Core Commands
- [ ] `agent_commands.go` → `cmds/agent_commands.go`
- [ ] `config_commands.go` → `cmds/config_commands.go`
- [ ] `memory_commands.go` → `cmds/memory_commands.go`
- [ ] `inference_commands.go` → `cmds/inference_commands.go`
- [ ] `knowledge_commands.go` → `cmds/knowledge_commands.go`

### Priority 2 - Feature Commands
- [ ] `pattern_commands.go` → `cmds/pattern_commands.go`
- [ ] `spellcheck_commands.go` → `cmds/spellcheck_commands.go`
- [ ] `history_commands.go` → `cmds/history_commands.go`
- [ ] `tokenizer_commands.go` → `cmds/tokenizer_commands.go`
- [ ] `training_commands.go` → `cmds/training_commands.go`

### Priority 3 - Advanced Commands
- [ ] `embedding_commands.go` → `cmds/embedding_commands.go`
- [ ] `evaluation_commands.go` → `cmds/evaluation_commands.go`
- [ ] `deployment_commands.go` → `cmds/deployment_commands.go`
- [ ] `speculative_commands.go` → `cmds/speculative_commands.go`
- [ ] `vector_commands.go` → `cmds/vector_commands.go`

### New Commands
- [x] `cmds/docs_commands.go` - Already created

## Migration Steps

### Phase 1: Setup (Completed)
1. ✅ Create `cmds/` directory
2. ✅ Create first command in new structure (`docs_commands.go`)
3. ✅ Create this migration plan

### Phase 2: Migration Process
For each command file:

1. **Update Package Declaration**
   ```go
   // Change from:
   package main
   
   // To:
   package cmds
   ```

2. **Move File**
   ```bash
   mv <command>_commands.go cmds/
   ```

3. **Update Imports in cli.go**
   ```go
   import (
       // ... other imports
       "github.com/yourusername/deltacli/cmds"
   )
   ```

4. **Update Function Calls**
   ```go
   // Change from:
   case "agent":
       return HandleAgentCommand(args)
   
   // To:
   case "agent":
       return cmds.HandleAgentCommand(args)
   ```

5. **Update Manager Access**
   - Commands that access global managers (like `GetMemoryManager()`) will need refactoring
   - Options:
     a. Pass managers as parameters
     b. Create a command context struct
     c. Use dependency injection pattern

### Phase 3: Refactoring Considerations

1. **Common Command Interface**
   ```go
   type Command interface {
       Execute(args []string) bool
       Help()
       Name() string
   }
   ```

2. **Command Registry**
   ```go
   type CommandRegistry struct {
       commands map[string]Command
   }
   ```

3. **Manager Context**
   ```go
   type CommandContext struct {
       MemoryManager     *MemoryManager
       AIManager         *AIPredictionManager
       InferenceManager  *InferenceManager
       // ... other managers
   }
   ```

## Testing Strategy

1. **Before Migration**: Document current command behavior
2. **During Migration**: Test each command after moving
3. **After Migration**: Run full command suite tests

## Benefits After Migration

1. **Modular Structure**: Each command is self-contained
2. **Easy to Add New Commands**: Clear pattern to follow
3. **Better Testing**: Commands can be tested in isolation
4. **Plugin Ready**: Structure supports future plugin system
5. **Documentation**: Commands can be auto-documented

## Timeline

- Phase 1: ✅ Complete (Setup)
- Phase 2: Week 1-2 (Migration)
- Phase 3: Week 3-4 (Refactoring)

## Notes

- Start with less complex commands first
- Maintain backward compatibility during migration
- Update documentation as commands are moved
- Consider creating command aliases for commonly used commands