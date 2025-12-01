|              |                |
| :----------- | :------------- |
| Feature Name | Rewrite in GO  |
| Start Date   | Nov 28th, 2025 |
| State        | **ACCEPTED**   |

# Summary

[summary]: #summary

Rewrite Distrobox from `sh` to `GO`: motivation, core principles and architecture.

# Motivation

[motivation]: #motivation

We want Distrobox to live long and prosper. As the tool is being used by more and more people with diverse 
environment setup, the need to add support to more stuff is pressing. 
The key factor to not mess it up it's **maintainability**.

We believe that by rewriting Distrobox in `GO` we can:
* add new features safely, by having extensive testing;
* have a better developer experience, through the use of a modern coding and testing toolchain;
* engage more with the community, by using a popular programming language;

# Design

[design]: #design

## Core principles
* Keep feature parity with the actual Distrobox implementation.
* Keep dependency list short.
* Enforce conventions with strict linting.
* Isolate the shell from core Distrobox logic.
* Write humble code.
  
## Architecture

### `cmd/main.go`

Entrypoint of the application, build and execute the CLI root command.

### `internal/cli/root.go`

Root CLI command.

* Load global configuration from either file or environment variables, and define the defaults.
* Define the global flags that are valid for every sub command (example: `--verbose`). 
* Select, instantiate and run the appropriate sub command.

### `internal/cli/<command>.go`

Any command exposed through the CLI (example: `distrobox list`). It's the component that is aware of the shell, and it
 manipulates Distrobox input/output by parsing the command flags and printing logs appropriately.

* Define command-specific flags.
* Instantiate and execute the relative command package.
* Handle command termination.
* Define shell printing logic by injecting a custom logger to the command package.
* Handle verbosity of the logs to the `stdout`.

### `pkg/command/<command>.go`

Implement the Distrobox command logic. It has no knowledge about the environment in which it is executed.

* Exposes a `Execute()` function that accepts input params as arguments and can return either `error` or the execution result.
* A custom logger is injected; a command knows nothing about verbosity, log level and (eventually) pretty printing.

Commands may also have different needs depending on their behaviour:

* Some commands are meant to be immediate and return a result (example: `list`); for such commands, the output data is returned in the execution result object.
* Some are long-running and require to return progress feedback to the user (example: `create`); such commands will log writes to the provided logger, delegating the presentation to the caller.
* Finally, some are _interactive_ and need to interact with the shell (example: `enter`); such command will require as input instances of `io.Reader` and `io.Writer` (that, for the shell scenario, can be bound to `stdin` and `stdout` respectively).

### `pkg/containermanager/containermanager.go`

Module that defines the interface for interacting with the actual system container manager.

### `pkg/containermanager/providers/<provider>.go`

Actual implementation of a container manager (examples: `docker`, `podman`, `lxc`, etc).
