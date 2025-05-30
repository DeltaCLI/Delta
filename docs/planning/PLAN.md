# Tab Completion Implementation Plan

This document outlines the approach for implementing tab completion functionality in the Delta CLI. The implementation will leverage the chzyer/readline library's AutoCompleter interface to provide a rich tab completion experience.

## Current Understanding

- Delta CLI uses the chzyer/readline library for command-line input processing
- The readline library supports tab completion through the AutoCompleter interface
- Tab completion needs to be implemented for several contexts:
  - Command history-based completion
  - File path completion
  - Executable command completion

## Implementation Strategy

### 1. Custom Completer Structure

Create a custom completer that implements the readline.AutoCompleter interface:

```go
type DeltaCompleter struct {
    history     *EncryptedHistoryHandler
    maxHistoryItems int
}

func (c *DeltaCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
    // Implement completion logic here
}
```

### 2. Command History-Based Completion

- Extract unique command prefixes from the user's command history
- Match the current input against historical commands
- Sort by frequency/recency for better suggestions

### 3. File Path Completion

- Implement file/directory path completion for commands that accept file paths
- Support relative and absolute paths
- Differentiate between files and directories in completion suggestions
- Implement proper path expansion for ~ (home directory)

### 4. Executable Command Completion

- Build and maintain a list of executable commands from PATH
- Refresh the list periodically or when PATH changes
- Match input against available commands
- Consider using prefix tree (trie) for efficient matching

### 5. Context-Aware Completion

- Determine the current completion context (command name, argument, file path)
- Apply the appropriate completion strategy based on context
- Support command-specific argument completion where applicable

## Integration Plan

1. Extend the readline.Config initialization in main():
   ```go
   completer := NewDeltaCompleter(historyHandler, historyLimit)
   rl, err := readline.NewEx(&readline.Config{
       Prompt:            "∆ ",
       InterruptPrompt:   "^C",
       EOFPrompt:         "exit",
       HistoryLimit:      historyLimit,
       HistorySearchFold: true,
       AutoComplete:      completer,
   })
   ```

2. Implement the DeltaCompleter with progressively enhanced features:
   - Start with basic history-based completion
   - Add file path completion
   - Add executable command completion
   - Refine with context awareness

## Testing Strategy

- Test each completion type individually
- Verify correct behavior with various input patterns
- Ensure completion works with special characters and spaces
- Test performance with large history and file sets

## Future Enhancements

- Command-specific argument completion
- Customizable completion behavior through configuration
- Support for custom aliases and user-defined completions
- Smarter context detection for better suggestions