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

## Development Workflow
When making changes, follow these steps:
1. Write code
2. Build with `make build`
3. Test manually
4. Use `go fmt` before committing

For larger changes, consider adding Go tests using the standard testing package.