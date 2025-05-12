# Delta Agent Definition YAML Format

This document defines the YAML format for Delta agents, enabling automatic agent creation from GitHub repositories and other external projects.

## Overview

The Delta Agent Definition YAML format allows project developers to include agent definitions directly in their repositories. Delta CLI can automatically discover and create agents from these definitions, providing seamless integration between Delta and external projects.

## File Location

Agent definition files should be placed in the following locations within a repository:

- `.delta/agents.yml` - Main agent definition file
- `.delta/agents/*.yml` - Individual agent definition files (for projects with multiple agents)

## YAML Schema

### Basic Structure

```yaml
# .delta/agents.yml
version: "1.0"  # Delta agent definition format version
project:
  name: "Project Name"
  repository: "https://github.com/username/project"
  description: "Project description"

# Global settings applied to all agents
settings:
  docker:
    enabled: true
    cache_dir: "${HOME}/.delta/cache/project"
    volumes:
      - "${PROJECT_ROOT}:/src"
    environment:
      PROJECT_VERSION: "1.0.0"
  
  # Global error patterns and solutions
  error_handling:
    patterns:
      - pattern: "fatal error: .+: No such file or directory"
        solution: "apt-get update && apt-get install -y build-essential"
        description: "Missing build essentials"

# List of agents for this project
agents:
  - id: "project-builder"
    name: "Project Builder"
    description: "Builds the main project components"
    enabled: true
    task_types: ["build", "test"]
    
    commands:
      - id: "build"
        command: "make all"
        working_dir: "${PROJECT_ROOT}"
        timeout: 3600
        retry_count: 3
        error_patterns:
          - "build failed"
          - "error:"
        success_patterns:
          - "build completed successfully"
        environment:
          BUILD_TYPE: "release"
    
    docker:
      enabled: true  # Override global setting if needed
      image: "project-builder"
      tag: "latest"
      dockerfile: "${PROJECT_ROOT}/.delta/Dockerfile.builder"
      volumes:
        - "${PROJECT_ROOT}:/src"
        - "project-cache:/cache"
      environment:
        CCACHE_DIR: "/cache/ccache"
    
    # Agent-specific error handling
    error_handling:
      auto_fix: true
      patterns:
        - pattern: "error: unknown type name 'uint'"
          solution: "sed -i 's/uint/unsigned int/g' $FILE"
          description: "Type definition error"
    
    # Project-specific metadata
    metadata:
      priority: "high"
      category: "build"
      tags: ["ci", "release"]
    
    # Trigger configuration
    triggers:
      patterns:
        - "build project"
        - "run tests"
      paths:
        - "src/**/*.c"
        - "include/**/*.h"
      schedules:
        - "0 2 * * *"  # Cron schedule format
```

### Multiple Agent Example

```yaml
# .delta/agents.yml
version: "1.0"
project:
  name: "DeepFry"
  repository: "https://github.com/username/deepfry"
  description: "DeepFry Project"

settings:
  docker:
    enabled: true
    cache_dir: "${HOME}/.delta/cache/deepfry"

agents:
  - id: "uproot-agent"
    name: "DeepFry Uproot"
    description: "Handles uproot commands"
    import: "agents/uproot.yml"  # Import detailed configuration from separate file

  - id: "build-agent"
    name: "DeepFry Builder"
    description: "Builds DeepFry components"
    import: "agents/builder.yml"
```

```yaml
# .delta/agents/builder.yml
id: "build-agent"  # Must match the ID in agents.yml
name: "DeepFry Builder"
description: "Builds DeepFry components with waterfall build system"
enabled: true
task_types: ["build", "release"]

commands:
  - id: "build-all"
    command: "run all"
    working_dir: "${DEEPFRY_HOME}"
    timeout: 7200
    retry_count: 5
    error_patterns:
      - "build failed"
      - "error:"

docker:
  enabled: true
  waterfall:
    compose_file: "${DELTA_CONFIG}/agents/deepfry/docker-compose.yml"
    stages:
      - "base"
      - "kernel" 
      - "fs"
      - "final"
    dependencies:
      base: []
      kernel: ["base"]
      fs: ["base"]
      final: ["kernel", "fs"]
  volumes:
    - "${DEEPFRY_HOME}:/src"
    - "deepfry-cache:/cache"
  environment:
    DEFCONFIG: "pocketpc"
```

## Required Sections

### Project Information

```yaml
project:
  name: string          # Required: Project name
  repository: string    # Required: Project repository URL
  description: string   # Optional: Project description
  website: string       # Optional: Project website
  docs: string          # Optional: Documentation URL
```

### Agent Definition

```yaml
agents:
  - id: string            # Required: Unique agent identifier
    name: string          # Required: Display name
    description: string   # Required: Description
    enabled: boolean      # Optional: Enable/disable agent (default: true)
    task_types: [string]  # Required: Agent task types
    commands: [Command]   # Required: Command definitions
    docker: DockerConfig  # Optional: Docker configuration
    error_handling: ErrorHandling  # Optional: Error handling configuration
    metadata: {string: string}     # Optional: Custom metadata
    triggers: TriggerConfig        # Optional: Trigger configuration
```

### Command Definition

```yaml
commands:
  - id: string            # Required: Command identifier
    command: string       # Required: Command to execute
    working_dir: string   # Required: Working directory
    timeout: integer      # Optional: Timeout in seconds (default: 3600)
    retry_count: integer  # Optional: Retry count (default: 0)
    error_patterns: [string]  # Optional: Error detection patterns
    success_patterns: [string]  # Optional: Success detection patterns
    is_interactive: boolean  # Optional: Interactive command (default: false)
    environment: {string: string}  # Optional: Environment variables
```

### Docker Configuration

```yaml
docker:
  enabled: boolean      # Optional: Enable Docker (default: true)
  image: string         # Optional: Docker image name
  tag: string           # Optional: Docker image tag
  dockerfile: string    # Optional: Dockerfile path
  compose_file: string  # Optional: Docker Compose file path
  waterfall:            # Optional: Waterfall build configuration
    stages: [string]    # Required if waterfall: Build stages
    dependencies: {string: [string]}  # Required if waterfall: Stage dependencies
  volumes: [string]     # Optional: Volume mounts
  networks: [string]    # Optional: Networks
  environment: {string: string}  # Optional: Environment variables
  build_args: {string: string}   # Optional: Build arguments
  use_cache: boolean    # Optional: Use Docker layer caching (default: true)
```

### Error Handling Configuration

```yaml
error_handling:
  auto_fix: boolean     # Optional: Auto-fix errors (default: true)
  patterns:             # Optional: Error patterns and solutions
    - pattern: string   # Required: Regex pattern
      solution: string  # Required: Command to fix
      description: string  # Optional: Description
      file_pattern: string  # Optional: File pattern to search
```

### Trigger Configuration

```yaml
triggers:
  patterns: [string]    # Optional: Command patterns to trigger agent
  paths: [string]       # Optional: File paths to trigger agent
  schedules: [string]   # Optional: Cron schedules
  events: [string]      # Optional: Events (push, pull_request, etc.)
```

## Environment Variables

Delta will automatically provide several environment variables that can be referenced in the YAML definition:

- `${PROJECT_ROOT}` - Root directory of the project repository
- `${DELTA_CONFIG}` - Delta configuration directory
- `${HOME}` - User's home directory
- `${USER}` - Current username
- Custom project-specific variables (e.g., `${DEEPFRY_HOME}`)

## Discovering and Using Repository Agents

Delta will automatically discover agent definitions in GitHub repositories:

1. When using `:jump` to a repository with `.delta/agents.yml`
2. When initially cloning a repository with `.delta/agents.yml` 
3. When manually requesting with `:agent discover [repository_path]`

### Automatic Agent Creation

```
:agent discover /path/to/repository
```

This command will:
1. Look for `.delta/agents.yml` in the repository
2. Parse the YAML definition
3. Create or update agents in Delta
4. Report the discovered agents

### Running Repository Agents

```
:agent run [agent_id]
```

### Listing Repository Agents

```
:agent list --repository=/path/to/repository
```

## Example: DeepFry Agent Definition

```yaml
# .delta/agents.yml
version: "1.0"
project:
  name: "DeepFry"
  repository: "https://github.com/username/deepfry"
  description: "DeepFry embedded system platform"

settings:
  docker:
    enabled: true
    cache_dir: "${HOME}/.delta/cache/deepfry"
    volumes:
      - "${DEEPFRY_HOME}:/src"
    environment:
      CCACHE_DIR: "/cache/ccache"

error_handling:
  patterns:
    - pattern: "fatal error: .+: No such file or directory"
      solution: "apt-get update && apt-get install -y build-essential"
      description: "Missing build essentials"
    - pattern: "make\\[\\d+\\]: \\*\\*\\* \\[.+\\] Error \\d+"
      solution: "make clean && make $TARGET"
      description: "Make build error requiring clean"

agents:
  - id: "deepfry-pocketpc-builder"
    name: "DeepFry PocketPC Builder"
    description: "Automated build system for DeepFry PocketPC defconfig"
    enabled: true
    task_types: ["build", "error-fix", "uproot"]
    
    commands:
      - id: "uproot"
        command: "uproot"
        working_dir: "${DEEPFRY_HOME}"
        timeout: 3600
        retry_count: 3
        error_patterns:
          - "failed to uproot"
          - "error:"
        success_patterns:
          - "uproot completed successfully"
        environment:
          UPROOT_OPTS: "--force --clean"
      
      - id: "run-all"
        command: "run all"
        working_dir: "${DEEPFRY_HOME}"
        timeout: 7200
        retry_count: 5
        error_patterns:
          - "build failed"
          - "error:"
        environment:
          BUILD_PARALLEL: "auto"
          BUILD_VERBOSE: "1"
    
    docker:
      enabled: true
      compose_file: "${DELTA_CONFIG}/agents/deepfry/docker-compose.yml"
      waterfall:
        stages: ["base", "kernel", "fs", "final"]
        dependencies:
          base: []
          kernel: ["base"]
          fs: ["base"]
          final: ["kernel", "fs"]
      volumes:
        - "${DEEPFRY_HOME}:/src"
        - "deepfry-cache:/cache"
      environment:
        DEFCONFIG: "pocketpc"
    
    error_handling:
      auto_fix: true
      patterns:
        - pattern: "error: unknown type name 'uint'"
          solution: "sed -i 's/uint/unsigned int/g' $FILE"
          description: "Type definition error"
    
    metadata:
      priority: "high"
      category: "build"
      defconfig: "pocketpc"
    
    triggers:
      patterns:
        - "deepfry build"
        - "uproot failing"
        - "pocketpc defconfig"
```

## Integration with GitHub Actions

Repositories using Delta Agent definitions can also integrate with GitHub Actions for CI/CD:

```yaml
# .github/workflows/build.yml
name: Build with Delta Agent

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Delta CLI
        uses: delta-cli/setup-action@v1
      
      - name: Discover agents
        run: delta agent discover .
      
      - name: Run build agent
        run: delta agent run deepfry-pocketpc-builder
```

## Conclusion

The Delta Agent Definition YAML format enables seamless integration between Delta CLI and external projects. By including agent definitions in repositories, project maintainers can provide automated build, test, and deployment tools that Delta users can immediately leverage, creating a powerful ecosystem for development automation.