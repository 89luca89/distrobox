// Package rootful provides utilities for running operations that require
// root privileges via a configurable sudo command.
package rootful

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

//nolint:gochecknoglobals // singleton: process-wide memoization is the intent
var (
	validateOnce      sync.Once
	errValidate       error
	cachedSudoCommand string
)

// Validate ensures that sudoCommand is available and the user can elevate.
// It runs `<sudoCommand> -v` at most once per process: the first call performs
// the check and caches the result; subsequent calls return the cached result
// without re-running the command. Passing a different sudoCommand after the
// first call is a programming error and returns an explicit error.
func Validate(ctx context.Context, sudoCommand string) error {
	if cachedSudoCommand != "" && cachedSudoCommand != sudoCommand {
		return fmt.Errorf("sudoCommand mismatch: already validated with %q, got %q", cachedSudoCommand, sudoCommand)
	}
	validateOnce.Do(func() {
		cachedSudoCommand = sudoCommand
		cmd := exec.CommandContext(ctx, sudoCommand, "-v")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			errValidate = fmt.Errorf("failed to validate %q: %w", sudoCommand, err)
		}
	})
	return errValidate
}
