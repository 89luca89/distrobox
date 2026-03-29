- [Distrobox](README.md)
  - [Distrobox Go Rewrite: Architecture and Design](#distrobox-go-rewrite-architecture-and-design)
    - [Overview](#overview)
    - [Directory Structure](#directory-structure)
    - [Architecture Layers](#architecture-layers)
    - [Dependency Injection Pattern](#dependency-injection-pattern)
    - [Configuration System](#configuration-system)
    - [Shell Scripts](#shell-scripts)

---

# Distrobox Go Rewrite: Architecture and Design

This document describes the architecture of the Distrobox Go rewrite, explaining how
different layers interact and the design decisions behind the codebase. It's meant to
help contributors understand the system and know where to make changes.

## Overview

The Distrobox Go rewrite is designed with **clear separation of concerns** in mind.
The rewrite followed these principles:

- CLI layer handles command-line parsing and user interaction;
- Container manager implementations are interchangeable;
- UI components can evolve independently;
- The codebase must remains testable and maintainable;
- Dependencies must be kept at minimum.

## Directory Structure

```text
distrobox
├── cmd/distrobox/
│   └── main.go                          # Entry point
├── internal/
│   ├── cli/                             # CLI layer (command definitions)
│   │   ├── root.go                      # Root command with global flags
│   │   ├── create.go, list.go, etc.     # Individual commands
│   │   └── helpers.go
│   ├── config/                          # Configuration management
│   ├── inside-distrobox/                 
│   │   └── assets/                      # Embedded shell scripts
├── pkg/
│   ├── commands/                        # Business logic layer
│   │   ├── create.go, list.go, etc.     # Command implementations
│   ├── containermanager/                # Container abstraction
│   │   ├── containermanager.go          # Interface definitions
│   │   └── providers/                   # Implementations
│   │       ├── podman.go
│   │       └── docker.go
│   ├── ui/                              # UI components
│   │   ├── progress.go
│   │   ├── printer.go
│   │   └── prompt.go
│   └── manifest/                        # Manifest parsing
```

## Architecture Layers

### 1. CLI Layer (`internal/cli/`)

The CLI layer handles command-line argument parsing, global flag processing, and
command dispatch. It is also responsible for binding the application to the shell's stdin/stdout,
for loading the configuration, and for instantiating the components.

Among other things, the concerete `ContainerManager` implementation is selected and instantiated in the cli layer.

### 2. Command Layer (`pkg/commands/`)

The command layer contains the business logic for each distrobox operation. Commands
are independent of CLI specifics and can be tested and reused independently.

Commands should be **pure orchestrators**. They coordinate between
the container manager abstraction and UI components, but don't contain low-level
implementation details.

Each command is implemented by a `Execute` method that takes a context and options struct.

### 3. Container Manager Layer (`pkg/containermanager/`)

The container manager is the abstraction over different container runtimes. This design
allows distrobox to work with Docker, Podman, and other container managers without
duplicating logic.

### 4. UI Layer (`pkg/ui/`)

The UI layer provides simple components for user interaction and output formatting.
These are instantiated in the CLI layer and passed to commands.

- **Progress**: Tracks multi-step operations with status indicators
- **Printer**: Formats and displays structured output
- **Prompter**: Gets user confirmation or input

## Dependency Injection Pattern

The architecture uses **context-based dependency injection** to pass the container
manager from the root command to all subcommands.

**Flow:**

```text
main()
  ↓
LoadConfig()
  ↓
NewRootCommand().Run()
  ↓
beforeAction() [global hooks]
  ↓ Creates container manager
  ↓ Stores in context
  ↓
Specific command action (e.g., createAction)
  ↓ Extracts container manager from context
  ↓ Creates UI tools
  ↓ Delegates to command layer
```

This pattern ensures:

- Container manager is available to all commands without global state
- UI tools are created fresh for each invocation
- Testing can substitute different implementations via context

## Configuration System

Configuration is loaded once at startup in `main()`. Configuration sources (in order of precedence):

1. Command-line flags
2. Environment variables (prefixed with `DBX_`)
3. Config file (`~/.config/distrobox/distrobox.conf`)
4. Defaults

This centralized approach makes it easy to understand where values come from and ensures consistency across commands.

## Shell Scripts

When a container is created, part of the `Distrobox` application is loaded in the container
as it is meant to be executed inside it:

- `distrobox-init` serves as the container entrypoint
- `distrobox-export` to expose binaries and applications to the host
- `distrobox-host-exec` to execute host's commands from inside the distrobox

Such commands are POSIX shell scripts that are included as assets in `internal/inside-distrobox/assets`
