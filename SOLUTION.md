# DeltaCLI Jump.sh Integration Issue

## Problem Analysis

The issue is that when `jump.sh delta` is executed through DeltaCLI, the directory doesn't change in the DeltaCLI shell. This happens because:

1. `jump.sh` uses the `jumpto` function that calls `cd` to change directories
2. When executed as a subprocess from DeltaCLI, the directory change only affects the subprocess
3. The directory change doesn't propagate back to the parent DeltaCLI process

This is a fundamental limitation of Unix process hierarchies: child processes cannot change the working directory of their parent process.

## Solution Options

### Option 1: Shell Function Integration

Integrate the `jump.sh` functionality directly into DeltaCLI as an internal command:

```go
// Add to internalCmds map in NewDeltaCompleter
"jump": {},

// Add to handleInternalCommand function
case "jump":
    if len(args) > 0 {
        handleJumpCommand(args[0])
    } else {
        fmt.Println("Usage: :jump <location>")
    }
    return true

// Implement the jump command handler
func handleJumpCommand(location string) {
    // Map of location shortcuts
    jumpLocations := map[string]string{
        "delta": "/home/bleepbloop/deltacli",
        "bin":   "/home/bleepbloop/black/bin",
        // Add more from jump.sh as needed
    }
    
    // Look up the location
    if path, ok := jumpLocations[location]; ok {
        // Change directory in the DeltaCLI process
        err := os.Chdir(path)
        if err != nil {
            fmt.Printf("Error: Failed to jump to %s: %v\n", location, err)
            return
        }
        fmt.Printf("Jumped to %s\n", path)
    } else {
        fmt.Printf("Unknown location: %s\n", location)
    }
}
```

### Option 2: Shell Alias Approach

Create a shell alias or function in your .bashrc/.zshrc that works with DeltaCLI:

```bash
# Add to .zshrc or .bashrc
function j() {
    # First try to jump
    source "$HOME/black/bin/jump.sh" "$@"
    
    # If in deltacli, also send the command to change directory there
    if [[ -n "$DELTACLI_RUNNING" ]]; then
        echo ":cd $(pwd)" > "$DELTACLI_COMMAND_FIFO"
    fi
}
```

This would require adding FIFO-based command injection to DeltaCLI, which is more complex.

### Option 3: Custom Shell Protocol

Add a shell protocol to DeltaCLI that detects when a directory has changed in a subprocess and updates the parent accordingly:

```go
// Modify executeCommand to capture directory changes
func executeCommand(cmd *exec.Cmd, sigChan chan os.Signal) error {
    // ... existing code ...
    
    // After command completes
    if err == nil {
        // Check if the subprocess was attempting to change directory
        if strings.HasPrefix(cmd.Args[len(cmd.Args)-1], "cd ") ||
           strings.Contains(cmd.Args[len(cmd.Args)-1], "jump.sh") {
            // Run pwd in the same shell to get the target directory
            pwdCmd := exec.Command(shell, "-c", "pwd")
            output, err := pwdCmd.Output()
            if err == nil {
                targetDir := strings.TrimSpace(string(output))
                // Change directory in the parent process
                os.Chdir(targetDir)
                fmt.Printf("Directory changed to: %s\n", targetDir)
            }
        }
    }
    
    return err
}
```

## Recommended Solution

The cleanest solution is Option 1 - implement the jump functionality natively in DeltaCLI. This:

1. Maintains a consistent user experience within DeltaCLI
2. Doesn't depend on external scripts
3. Allows future customization specific to DeltaCLI

However, it requires maintaining the location map in synchronization with jump.sh.

Implement it as a `:jump` or `:j` command to avoid conflicts with the existing jump.sh script.