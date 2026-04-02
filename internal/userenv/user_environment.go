package userenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

type UserEnvironment struct {
	User    string
	UserID  string
	GroupID string
	Home    string
	Shell   string
}

// LoadUserEnvironment loads the user environment variables
// Data is sourced from environment variables, and falls back to system calls
// if the environment variables are not set
// Expected variables are:
// - USER
// - HOME
// - SHELL
func LoadUserEnvironment(ctx context.Context) *UserEnvironment {
	env := &UserEnvironment{}

	// USER
	env.User = os.Getenv("USER")
	if env.User == "" {
		// Try id -run (most reliable)
		if output, err := exec.CommandContext(ctx, "id", "-run").Output(); err == nil {
			env.User = strings.TrimSpace(string(output))
		} else if u, err := user.Current(); err == nil {
			// Fallback to os/user
			env.User = u.Username
		} else {
			env.User = "nobody"
		}
	}

	// Get passwd entry once for HOME and SHELL
	passwdEntry := getPasswdFields(ctx, env.User)

	// HOME
	env.Home = os.Getenv("HOME")
	if env.Home == "" {
		if len(passwdEntry) > 5 && passwdEntry[5] != "" {
			env.Home = passwdEntry[5]
		} else if u, err := user.Lookup(env.User); err == nil {
			env.Home = u.HomeDir
		} else {
			env.Home = fmt.Sprintf("/home/%s", env.User)
		}
	}

	// SHELL
	env.Shell = os.Getenv("SHELL")
	if env.Shell == "" {
		if len(passwdEntry) > 6 && passwdEntry[6] != "" {
			env.Shell = passwdEntry[6]
		} else {
			env.Shell = "/bin/sh"
		}
	}

	// USER ID
	if uid := os.Getuid(); uid >= 0 {
		env.UserID = strconv.Itoa(uid)
	} else if uid, err := exec.CommandContext(ctx, "id", "-ru").Output(); err == nil {
		env.UserID = strings.TrimSpace(string(uid))
	}

	// GROUP ID
	if gid := os.Getgid(); gid >= 0 {
		env.GroupID = strconv.Itoa(gid)
	} else if gid, err := exec.CommandContext(ctx, "id", "-rg").Output(); err == nil {
		env.GroupID = strings.TrimSpace(string(gid))
	}

	return env
}

// getPasswdFields returns all fields from passwd entry for a user
func getPasswdFields(ctx context.Context, userName string) []string {
	// Use getent passwd (handles LDAP, NIS, local users)
	output, err := exec.CommandContext(ctx, "getent", "passwd", userName).Output()
	if err != nil {
		return nil
	}

	return strings.Split(strings.TrimSpace(string(output)), ":")
}
