// Package rootful provides utilities for running operations that require
// root privileges via sudo.
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
	validateOnce sync.Once
	errValidate  error
)

// Validate ensures that sudo is available and the user can elevate.
// It runs `sudo -v` at most once per process: the first call performs the
// check and caches the result; subsequent calls return the cached result
// without re-running the command.
func Validate(ctx context.Context) error {
	validateOnce.Do(func() {
		cmd := exec.CommandContext(ctx, "sudo", "-v")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			errValidate = fmt.Errorf("failed to validate sudo: %w", err)
		}
	})
	return errValidate
}
