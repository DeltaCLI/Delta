# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Build: `make build`
- Run: `make run`
- Clean: `make clean`
- Install: `make install`

## Code Style Guidelines
1. **Formatting**: Follow Go standard formatting using `gofmt`
2. **Imports**: Group standard library imports first, then third-party packages
3. **Error Handling**: Always check for errors and handle them appropriately
4. **Naming Conventions**:
   - Use CamelCase for exported functions/variables
   - Use camelCase for unexported functions/variables
   - Package names should be short and lowercase
5. **Types**: Always specify types explicitly when not obvious from context
6. **Comments**: Document public functions with meaningful comments
7. **No Hardcoded Values**:
   - Avoid hardcoded shell paths (use $SHELL environment variable)
   - Avoid hardcoded file paths (use os.UserHomeDir() or similar)
   - Use environment variables or configuration when possible
   - DO NOT hardcode command names or special handling for specific applications

## Development Workflow
When making changes, follow these steps:
1. Write code
2. Build with `make build`
3. Test manually
4. Use `go fmt` before committing

For larger changes, consider adding Go tests using the standard testing package.

## Interactive Application Support
When working with subprocess execution:
1. **Signal Handling**:
   - Properly reset signal handlers before starting subprocesses
   - Forward signals directly to subprocesses
   - Don't use process groups for command execution unless necessary
   - Restore shell signal handlers after subprocess completes
   - IMPORTANT: Do NOT use the `-i` flag when executing shells as it breaks signal handling
2. **Terminal Support**:
   - Test interactive applications (e.g., htop, vim, nano) to ensure proper handling
   - Ensure proper TTY setup for subprocesses
   - Source appropriate shell profile files (.bashrc, .zshrc) explicitly rather than relying on `-i` flag