# DeltaCLI Jump Functionality

This feature provides a directory navigation system similar to the external `jump.sh` script but integrated natively into the DeltaCLI shell.

## Features

- Fast directory navigation with named locations
- Persistent location storage in `~/.config/delta/jump_locations.json`
- Ability to import locations from external `jump.sh` script
- Tab completion for location names

## Usage

### Basic Navigation

```
∆ jump <location>
```

This will change the current directory to the saved location.

### Adding Locations

```
∆ jump add <name> [path]
```

If path is not provided, the current directory will be used.

### Removing Locations

```
∆ jump remove <name>
```

or

```
∆ jump rm <name>
```

### Listing Locations

```
∆ jump
```

or

```
∆ jump list
```

### Importing Locations from jump.sh

```
∆ jump import jumpsh
```

This will scan the jump.sh script at `/home/bleepbloop/black/bin/jump.sh` and import any locations it finds.

## Shortcuts

You can use `:j` as a shorthand for the jump command:

```
∆ :j <location>
```

## Configuration

Jump locations are stored in JSON format at `~/.config/delta/jump_locations.json`.

## Advanced Usage

### Native Command vs External Script

The native `jump` command within DeltaCLI will take precedence over any external `jump.sh` script. This means that typing `jump delta` will use the internal implementation rather than the external script.

### Tab Completion

Press Tab after typing `:jump ` or `:j ` to see available locations.

## Implementation Details

- The jump functionality is implemented in `jump_manager.go` as a separate module
- Locations are stored in a JSON file for persistence
- `jump_helper.go` provides the integration with the main CLI
- It respects separation of concerns by keeping the jump functionality isolated