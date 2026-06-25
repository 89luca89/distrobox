package userenv_test

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/internal/userenv"
)

func TestLoadUserEnvironment_EnvironmentVariables(t *testing.T) {
	// Save original env vars
	origUser := os.Getenv("USER")
	origHome := os.Getenv("HOME")
	origShell := os.Getenv("SHELL")

	// Cleanup
	defer func() {
		t.Setenv("USER", origUser)
		t.Setenv("HOME", origHome)
		t.Setenv("SHELL", origShell)
	}()

	// Set test env vars
	t.Setenv("USER", "testuser")
	t.Setenv("HOME", "/test/home")
	t.Setenv("SHELL", "/test/shell")

	ctx := context.Background()
	env := userenv.LoadUserEnvironment(ctx)

	// Verify env vars took precedence
	assert.Equal(t, "testuser", env.User)
	assert.Equal(t, "/test/home", env.Home)
	assert.Equal(t, "/test/shell", env.Shell)

	// UserID and GroupID should still be populated from system
	assert.NotEmpty(t, env.UserID, "UserID should not be empty")
	assert.NotEmpty(t, env.GroupID, "GroupID should not be empty")

	// Verify they're numeric strings
	_, err := strconv.Atoi(env.UserID)
	require.NoError(t, err, "UserID should be numeric, got '%s'", env.UserID)
	_, err = strconv.Atoi(env.GroupID)
	assert.NoError(t, err, "GroupID should be numeric, got '%s'", env.GroupID)
}

func TestLoadUserEnvironment_NoEnvironmentVariables(t *testing.T) {
	// Save and clear env vars
	origUser := os.Getenv("USER")
	origHome := os.Getenv("HOME")
	origShell := os.Getenv("SHELL")

	os.Unsetenv("USER")
	os.Unsetenv("HOME")
	os.Unsetenv("SHELL")

	defer func() {
		if origUser != "" {
			t.Setenv("USER", origUser)
		}
		if origHome != "" {
			t.Setenv("HOME", origHome)
		}
		if origShell != "" {
			t.Setenv("SHELL", origShell)
		}
	}()

	ctx := context.Background()
	env := userenv.LoadUserEnvironment(ctx)

	// Should have found values from system
	assert.NotEmpty(t, env.User, "User should not be empty")

	if env.User == "nobody" {
		t.Log("Warning: User defaulted to 'nobody', system lookups might have failed")
	}

	assert.NotEmpty(t, env.Home, "Home should not be empty")

	// Shell should at least have the default
	assert.NotEmpty(t, env.Shell, "Shell should not be empty")
	assert.NotEmpty(t, env.UserID, "UserID should not be empty")
	assert.NotEmpty(t, env.GroupID, "GroupID should not be empty")
}

func TestLoadUserEnvironment_ContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Clear USER to force getent call
	origUser := os.Getenv("USER")
	os.Unsetenv("USER")
	defer func() {
		if origUser != "" {
			t.Setenv("USER", origUser)
		}
	}()

	env := userenv.LoadUserEnvironment(ctx)

	// Should still work but getent might fail due to cancelled context
	// The function should handle this gracefully
	assert.NotEmpty(t, env.User, "User should not be empty even with cancelled context")

	// Should have defaults or fallbacks
	assert.NotEmpty(t, env.Home, "Home should not be empty")
	assert.NotEmpty(t, env.Shell, "Shell should not be empty")
}

func TestLoadUserEnvironment_ContextTimeout(t *testing.T) {
	// Very short timeout to potentially interrupt getent
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure timeout has passed
	time.Sleep(10 * time.Millisecond)

	env := userenv.LoadUserEnvironment(ctx)

	// Function should still return valid data through fallbacks
	require.NotNil(t, env, "LoadUserEnvironment should never return nil")

	// Basic fields should still be populated
	assert.NotEmpty(t, env.UserID, "UserID should be populated even with timeout")
	assert.NotEmpty(t, env.GroupID, "GroupID should be populated even with timeout")
}

func TestLoadUserEnvironment_PartialEnvironment(t *testing.T) {
	origUser := os.Getenv("USER")
	origHome := os.Getenv("HOME")
	origShell := os.Getenv("SHELL")

	defer func() {
		if origUser != "" {
			t.Setenv("USER", origUser)
		} else {
			os.Unsetenv("USER")
		}
		if origHome != "" {
			t.Setenv("HOME", origHome)
		} else {
			os.Unsetenv("HOME")
		}
		if origShell != "" {
			t.Setenv("SHELL", origShell)
		} else {
			os.Unsetenv("SHELL")
		}
	}()

	// Set only USER, let HOME and SHELL come from system
	t.Setenv("USER", "testuser")
	os.Unsetenv("HOME")
	os.Unsetenv("SHELL")

	ctx := context.Background()
	env := userenv.LoadUserEnvironment(ctx)

	assert.Equal(t, "testuser", env.User)

	// HOME should be populated (either from passwd or default)
	assert.NotEmpty(t, env.Home, "Home should not be empty")

	// If getent fails for testuser, should have default home
	if env.Home == "/home/testuser" {
		t.Log("Home correctly defaulted to /home/testuser")
	}

	// Shell should have a value (from passwd or default)
	assert.NotEmpty(t, env.Shell, "Shell should not be empty")

	// Default shell should be /bin/sh if user doesn't exist
	if env.Shell == "/bin/sh" {
		t.Log("Shell correctly defaulted to /bin/sh")
	}
}

func TestLoadUserEnvironment_IDs(t *testing.T) {
	ctx := context.Background()
	env := userenv.LoadUserEnvironment(ctx)

	// These should always be populated on a Unix system
	assert.NotEmpty(t, env.UserID, "UserID should not be empty")
	assert.NotEmpty(t, env.GroupID, "GroupID should not be empty")

	// Verify they're valid integers
	uid, err := strconv.Atoi(env.UserID)
	require.NoError(t, err, "UserID should be a valid integer, got '%s'", env.UserID)

	gid, err := strconv.Atoi(env.GroupID)
	require.NoError(t, err, "GroupID should be a valid integer, got '%s'", env.GroupID)

	// Verify they match os.Getuid() and os.Getgid()
	assert.Equal(t, os.Getuid(), uid, "UserID mismatch")
	assert.Equal(t, os.Getgid(), gid, "GroupID mismatch")
}

func TestLoadUserEnvironment_SystemCommands(t *testing.T) {
	// Test that our system commands actually work
	commands := []struct {
		name string
		cmd  string
		args []string
	}{
		{"id -run", "id", []string{"-run"}},
		{"id -ru", "id", []string{"-ru"}},
		{"id -rg", "id", []string{"-rg"}},
		{"getent passwd", "getent", []string{"passwd", os.Getenv("USER")}},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			// Skip if USER not set for getent test
			if tc.cmd == "getent" && os.Getenv("USER") == "" {
				t.Skip("USER env var not set")
			}

			output, err := exec.Command(tc.cmd, tc.args...).Output()
			if err != nil {
				t.Logf("Warning: %s command failed: %v", tc.name, err)
				// Don't fail the test as command might not be available
				return
			}

			result := strings.TrimSpace(string(output))
			assert.NotEmpty(t, result, "%s returned empty output", tc.name)

			t.Logf("%s output: %s", tc.name, result)
		})
	}
}
