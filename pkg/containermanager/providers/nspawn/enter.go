// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of the distrobox project:
//    https://github.com/89luca89/distrobox
//
// Copyright (C) 2021 distrobox contributors
//
// distrobox is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License version 3
// as published by the Free Software Foundation.
//
// distrobox is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with distrobox; if not, see <http://www.gnu.org/licenses/>.

package nspawn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

const (
	setupPollInterval = 500 * time.Millisecond
	// setupTimeout bounds the wait for distrobox-init to report readiness;
	// first boots install packages, so it is generous.
	setupTimeout = 30 * time.Minute
)

func (n *Nspawn) Enter(
	ctx context.Context,
	options containermanager.EnterOptions,
	progress *ui.Progress,
	printer *ui.Printer,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)

	m, err := loadMachine(n.dirs(), options.ContainerName)
	if err != nil {
		if !options.DryRun {
			return err
		}
		// Dry-run without an existing machine: show what an exec into a
		// default machine would look like, matching the podman provider.
		m = &machine{
			Name:        options.ContainerName,
			MachineName: sanitizeMachineName(options.ContainerName),
			UnitName:    unitName(sanitizeMachineName(options.ContainerName)),
			Home:        userEnv.Home,
		}
	}

	env, workdir, err := buildEnterEnv(m, options)
	if err != nil {
		return err
	}

	unshareGroups := m.Labels["distrobox.unshare_groups"] == "1"
	commandArgs := containermanager.BuildCommandArgs(options.CustomCommand, userEnv.User, options.NoTTY, unshareGroups)

	if options.DryRun {
		command := n.execArgvInit(m, userEnv, workdir, env, commandArgs, options)
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(strings.Join(command, " "))
		return nil
	}

	if !n.root {
		if err := n.probeRootless(ctx); err != nil {
			return err
		}
	}

	if n.machineStatus(ctx, m) != containermanager.RunningStatus {
		if err := n.startMachine(ctx, m, progress, printer); err != nil {
			return err
		}
		progress.Finalize("Container Setup Complete!")
	}

	command, err := n.execArgv(ctx, m, userEnv, workdir, env, commandArgs, options)
	if err != nil {
		return err
	}
	if _, err := n.run(ctx, command, runOptions{Interactive: true, Privileged: true}); err != nil {
		return err
	}
	return nil
}

// buildEnterEnv computes the environment and working directory for the
// entered session, mirroring the podman/docker enter command generators.
func buildEnterEnv(m *machine, options containermanager.EnterOptions) ([]string, string, error) {
	home := m.Home
	if m.CustomHome != "" {
		home = m.CustomHome
	}

	workdir, err := containermanager.GetWorkDir(home, options.NoWorkDir)
	if err != nil {
		return nil, "", fmt.Errorf("error getting the workdir: %w", err)
	}

	executablePath, err := os.Executable()
	if err != nil {
		return nil, "", fmt.Errorf("error getting the executable path: %w", err)
	}

	env := []string{
		"PWD=" + workdir,
		"CONTAINER_ID=" + m.Name,
		"DISTROBOX_PATH=" + executablePath,
	}
	env = append(env, containermanager.FilterEnvVars()...)

	// The container PATH is unknown before entering; the empty container
	// path makes BuildContainerPath fall back to the standard paths.
	env = append(env, "PATH="+containermanager.BuildContainerPath(options.CleanPath, os.Getenv("PATH"), ""))
	env = append(env, "XDG_DATA_DIRS="+containermanager.BuildXDGPaths("XDG_DATA_DIRS", []string{"/usr/local/share", "/usr/share"}))
	env = append(env, "XDG_CONFIG_DIRS="+containermanager.BuildXDGPaths("XDG_CONFIG_DIRS", []string{"/etc/xdg"}))
	env = append(env,
		"XDG_CACHE_HOME="+home+"/.cache",
		"XDG_CONFIG_HOME="+home+"/.config",
		"XDG_DATA_HOME="+home+"/.local/share",
		"XDG_STATE_HOME="+home+"/.local/state",
	)
	env = append(env, "HOME="+home)
	env = append(env, "SHELL="+os.Getenv("SHELL"))
	env = append(env, "TERM="+os.Getenv("TERM"))
	return env, workdir, nil
}

// execArgv builds the command that runs a session inside the running
// machine. Initful machines boot systemd, so systemd-run -M can reach
// their manager; non-init machines have distrobox-init as PID 1 with no
// D-Bus, so the session enters through nsenter on the machine leader.
func (n *Nspawn) execArgv(
	ctx context.Context,
	m *machine,
	userEnv *userenv.UserEnvironment,
	workdir string,
	env []string,
	commandArgs []string,
	options containermanager.EnterOptions,
) ([]string, error) {
	if m.Init {
		return n.execArgvInit(m, userEnv, workdir, env, commandArgs, options), nil
	}

	leader, err := n.machineLeader(ctx, m)
	if err != nil {
		return nil, err
	}
	return n.execArgvNsenter(m, userEnv, leader, workdir, env, commandArgs, options), nil
}

// execArgvInit runs the session as a transient unit inside the machine's
// systemd, which maps almost 1:1 onto docker exec semantics.
func (n *Nspawn) execArgvInit(
	m *machine,
	userEnv *userenv.UserEnvironment,
	workdir string,
	env []string,
	commandArgs []string,
	options containermanager.EnterOptions,
) []string {
	cmd := []string{"systemd-run"}
	cmd = append(cmd,
		"--machine="+m.MachineName,
		"--quiet",
		"--collect",
		"--wait",
		"--service-type=exec",
	)

	if !options.NoTTY && containermanager.IsTTY() {
		cmd = append(cmd, "--pty")
	} else {
		cmd = append(cmd, "--pipe")
	}

	if m.Labels["distrobox.unshare_groups"] == "1" {
		cmd = append(cmd, "--uid=0")
	} else {
		cmd = append(cmd, "--uid="+userEnv.UserID)
	}

	cmd = append(cmd, "--working-directory="+workdir)
	for _, e := range env {
		cmd = append(cmd, "--setenv="+e)
	}

	// Additional enter flags use systemd-run syntax with this backend.
	if len(options.AdditionalFlags) > 0 {
		cmd = append(cmd, strings.Fields(options.AdditionalFlags)...)
	}

	cmd = append(cmd, "--")
	cmd = append(cmd, commandArgs...)
	return cmd
}

// execArgvNsenter joins the machine's namespaces directly via the leader
// PID. TTY handling needs no allocation: the interactive run inherits the
// caller's terminal.
func (n *Nspawn) execArgvNsenter(
	m *machine,
	userEnv *userenv.UserEnvironment,
	leader string,
	workdir string,
	env []string,
	commandArgs []string,
	options containermanager.EnterOptions,
) []string {
	cmd := []string{
		"nsenter",
		"--target", leader,
		"--mount", "--uts", "--ipc", "--pid",
	}
	// Rootful machines share the host network unless unshared; rootless
	// machines always have their own (pasta-backed) namespace to join.
	if m.Unshare.NetNS || !n.root {
		cmd = append(cmd, "--net")
	}
	if !n.root {
		// Unprivileged machines live in a caller-owned user namespace;
		// join it first so the setuid below happens inside the mapping.
		cmd = append(cmd, "--user")
	}

	if m.Labels["distrobox.unshare_groups"] != "1" {
		cmd = append(cmd, "--setuid", userEnv.UserID, "--setgid", userEnv.GroupID)
	}

	// --wdns resolves the directory inside the entered mount namespace.
	cmd = append(cmd, "--wdns="+workdir)

	if len(options.AdditionalFlags) > 0 {
		cmd = append(cmd, strings.Fields(options.AdditionalFlags)...)
	}

	cmd = append(cmd, "env", "-i")
	cmd = append(cmd, env...)
	cmd = append(cmd, commandArgs...)
	return cmd
}

// machineLeader returns the PID of the machine's PID 1 as seen from the
// host, preferring machined and falling back to the transient unit's
// process tree.
func (n *Nspawn) machineLeader(ctx context.Context, m *machine) (string, error) {
	out, err := n.run(ctx, []string{"machinectl", "show", m.MachineName, "--property=Leader", "--value"}, runOptions{})
	if leader := strings.TrimSpace(out); err == nil && leader != "" && leader != "0" {
		return leader, nil
	}

	// Fallback: the unit's MainPID is the nspawn supervisor; the machine
	// leader is its only child.
	argv := []string{"systemctl"}
	if !n.root {
		argv = append(argv, "--user")
	}
	argv = append(argv, "show", m.UnitName, "--property=MainPID", "--value")
	out, err = n.run(ctx, argv, runOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot determine machine leader for %s: %w", m.Name, err)
	}
	supervisor := strings.TrimSpace(out)
	if supervisor == "" || supervisor == "0" {
		return "", fmt.Errorf("container %s is not running", m.Name)
	}

	childrenPath := fmt.Sprintf("/proc/%s/task/%s/children", supervisor, supervisor)
	raw, err := os.ReadFile(childrenPath)
	if err != nil {
		return "", fmt.Errorf("cannot determine machine leader for %s: %w", m.Name, err)
	}
	children := strings.Fields(string(raw))
	if len(children) == 0 {
		return "", fmt.Errorf("cannot determine machine leader for %s: no child processes", m.Name)
	}
	return children[0], nil
}

// cleanStaleRuntime removes leftover per-machine nspawn runtime state
// (best-effort; only called when the machine's unit is inactive).
func (n *Nspawn) cleanStaleRuntime(ctx context.Context, m *machine) {
	if n.root {
		_ = n.runHelper(ctx, nil, "clean-runtime", "--name", m.MachineName)
		return
	}
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		cleanUnixExport(filepath.Join(runtimeDir, "systemd", "nspawn", "unix-export"), m.MachineName)
	}
}

// attachNetwork connects the machine's private network namespace to the
// outside world with pasta, which daemonizes and lives until the namespace
// goes away.
func (n *Nspawn) attachNetwork(ctx context.Context, m *machine) error {
	leader, err := n.waitForLeader(ctx, m)
	if err != nil {
		return err
	}
	if _, err := n.run(ctx, []string{"pasta", "--config-net", "--quiet", leader}, runOptions{}); err != nil {
		return fmt.Errorf("cannot attach pasta to the container (is the passt package installed?): %w", err)
	}
	return nil
}

// waitForLeader polls for the machine's leader PID, which appears once
// machined registration completes shortly after the unit starts.
func (n *Nspawn) waitForLeader(ctx context.Context, m *machine) (string, error) {
	var lastErr error
	//nolint:mnd // ~10s of polling
	for range 20 {
		leader, err := n.machineLeader(ctx, m)
		if err == nil {
			return leader, nil
		}
		lastErr = err
		time.Sleep(setupPollInterval)
	}
	return "", lastErr
}

// startMachine launches the machine's transient unit and waits for
// distrobox-init to finish the container setup, following init.log the way
// the podman/docker providers follow the engine logs.
func (n *Nspawn) startMachine(ctx context.Context, m *machine, progress *ui.Progress, printer *ui.Printer) error {
	d := n.dirs()
	logPath := d.initLogPath(m.Name)

	// Remember where the log ends now: everything after this offset belongs
	// to this boot.
	var offset int64
	if info, err := os.Stat(logPath); err == nil {
		offset = info.Size()
	}

	// An unclean previous shutdown can leave nspawn's per-machine runtime
	// directory behind, which makes the next start refuse with "Mount
	// point ... exists already". The unit is inactive here, so the state
	// is stale by definition.
	n.cleanStaleRuntime(ctx, m)

	if _, err := n.run(ctx, m.StartCommand, runOptions{Privileged: true}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Unprivileged machines always get a private network namespace (the
	// managed user namespace cannot own the host's), so networking comes
	// from pasta attached to the machine's namespaces — the same user-mode
	// networking rootless podman uses.
	if !n.root {
		if err := n.attachNetwork(ctx, m); err != nil {
			printer.PrintWarning("no network in container %s: %v", m.Name, err)
		}
	}

	progress.Next("Starting container...")

	userEnv := userenv.LoadUserEnvironment(ctx)
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(userEnv.Home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "distrobox")
	if err := os.MkdirAll(cacheDir, 0755); err != nil { //nolint:gosec // we need this writable by everybody
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return n.waitForSetup(ctx, m, logPath, offset, progress, printer)
}

// waitForSetup tails the machine's init.log from offset until
// distrobox-init reports completion, mirroring the log line protocol of the
// podman/docker providers.
func (n *Nspawn) waitForSetup(
	ctx context.Context,
	m *machine,
	logPath string,
	offset int64,
	progress *ui.Progress,
	printer *ui.Printer,
) error {
	deadline := time.Now().Add(setupTimeout)
	var pending string

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("nspawn: %w", err)
		}
		if time.Now().After(deadline) {
			progress.Fail()
			return errors.New("timed out waiting for container setup")
		}

		chunk, newOffset, err := readLogChunk(logPath, offset)
		if err == nil {
			offset = newOffset
			pending += chunk

			lines := strings.Split(pending, "\n")
			// The last element is either empty (chunk ended on a newline)
			// or an incomplete line to carry over.
			pending = lines[len(lines)-1]
			for _, line := range lines[:len(lines)-1] {
				done, err := processSetupLine(line, progress, printer)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			}
		}

		// The unit dying during setup means the machine failed to boot.
		if n.machineStatus(ctx, m) != containermanager.RunningStatus {
			printer.PrintError("\nContainer Setup Failure!")
			logs, _ := os.ReadFile(logPath)
			return fmt.Errorf("container stopped during setup:\n%s", tailOf(string(logs)))
		}

		time.Sleep(setupPollInterval)
	}
}

// processSetupLine interprets one distrobox-init log line, mirroring the
// line protocol of the podman/docker providers. It reports whether setup
// completed.
func processSetupLine(line string, progress *ui.Progress, printer *ui.Printer) (bool, error) {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "+"):
		// Ignore logging commands

	case strings.HasPrefix(line, "Error:"):
		progress.Fail()
		printer.PrintError(line)
		return false, fmt.Errorf("container setup error: %s", line)

	case strings.HasPrefix(line, "Warning:"):
		printer.PrintWarning(line)

	case strings.HasPrefix(line, "distrobox:"):
		parts := strings.SplitN(line, " ", 2)
		if len(parts) > 1 {
			progress.Done()
			progress.Next("%s", parts[1])
		}

	case strings.HasPrefix(line, "container_setup_done"):
		return true, nil
	}
	return false, nil
}

// readLogChunk returns the log content after offset and the new offset. A
// truncated (rotated) log restarts from the beginning.
func readLogChunk(path string, offset int64) (string, int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", offset, fmt.Errorf("cannot read init log: %w", err)
	}
	if offset > int64(len(raw)) {
		offset = 0
	}
	return string(raw[offset:]), int64(len(raw)), nil
}

// tailOf returns the last few lines of a log for error messages.
func tailOf(logs string) string {
	lines := strings.Split(strings.TrimSpace(logs), "\n")
	//nolint:mnd // keep error output short
	if len(lines) > 15 {
		lines = lines[len(lines)-15:]
	}
	return strings.Join(lines, "\n")
}
