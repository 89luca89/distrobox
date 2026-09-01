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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ocistore"
)

// deviceBindCandidates returns the host device paths shared with the
// machine when devices are not unshared. nspawn owns the machine's /dev, so
// unlike podman's whole-/dev bind, devices are forwarded individually (each
// only if present on the host).
func deviceBindCandidates() []string {
	return []string{
		"/dev/dri",
		"/dev/snd",
		"/dev/input",
		"/dev/bus/usb",
		"/dev/fuse",
		"/dev/kvm",
		"/dev/net/tun",
		"/dev/vfio",
	}
}

// nvidiaBindCandidates returns the devices forwarded additionally when
// --nvidia is set: distrobox-init integrates the userspace driver from
// /run/host, but it cannot mknod the device nodes in an unprivileged
// machine.
func nvidiaBindCandidates() []string {
	return []string{
		"/dev/nvidiactl",
		"/dev/nvidia-modeset",
		"/dev/nvidia-uvm",
		"/dev/nvidia-uvm-tools",
		"/dev/nvidia-caps",
		"/dev/nvidia0",
		"/dev/nvidia1",
		"/dev/nvidia2",
		"/dev/nvidia3",
	}
}

func (n *Nspawn) Create(ctx context.Context, opts containermanager.CreateOptions) error {
	userEnv := userenv.LoadUserEnvironment(ctx)

	scriptsDir, err := insidedistrobox.ProvisionScripts(opts.ScriptsDir)
	if err != nil {
		return fmt.Errorf("failed to provision scripts: %w", err)
	}

	// ensure custom home dir exists, if needed
	if opts.ContainerUserCustomHome != "" && !containermanager.PathExists(opts.ContainerUserCustomHome) {
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.MkdirAll(opts.ContainerUserCustomHome, 0755); err != nil {
			return fmt.Errorf("failed to create custom home directory: %w", err)
		}
	}

	d := n.dirs()
	m, err := n.buildMachine(opts, userEnv, d, scriptsDir)
	if err != nil {
		return err
	}

	if opts.DryRun {
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(strings.Join(m.StartCommand, " "))
		return nil
	}

	if !n.root {
		if err := n.probeRootless(ctx); err != nil {
			return err
		}
	}

	if n.root {
		return n.createRootful(ctx, m)
	}
	return createMachine(ctx, d, m)
}

// scriptTargets maps the in-container /usr/bin entry names onto the host
// script files copied into the rootfs.
func scriptTargets() map[string]string {
	return map[string]string{
		"entrypoint":          "distrobox-init",
		"distrobox-export":    "distrobox-export",
		"distrobox-host-exec": "distrobox-host-exec",
	}
}

// plantScripts copies the distrobox helper scripts into the rootfs at the
// same /usr/bin paths podman/docker bind-mount them to. Binds are not an
// option here: unprivileged nspawn resolves bind sources with the machine's
// mapped uid, which cannot traverse the user's 0700 home — where the
// scripts (and any user path) usually live. The rootfs is ours, so copies
// planted at create time work in both privilege modes.
func plantScripts(rootfs, scriptsDir string) error {
	binDir := filepath.Join(rootfs, "usr", "bin")
	//nolint:gosec // standard directory permissions
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", binDir, err)
	}

	// Many images ship /usr/bin read-only (e.g. Fedora, mode r-x); we own
	// the extracted copy, so lift the mode while planting and restore it.
	info, err := os.Stat(binDir)
	if err != nil {
		return fmt.Errorf("cannot inspect %s: %w", binDir, err)
	}
	if info.Mode().Perm()&0o700 != 0o700 {
		if err := os.Chmod(binDir, info.Mode().Perm()|0o700); err != nil {
			return fmt.Errorf("cannot make %s writable: %w", binDir, err)
		}
		defer func() { _ = os.Chmod(binDir, info.Mode().Perm()) }()
	}

	for name, script := range scriptTargets() {
		content, err := os.ReadFile(filepath.Join(scriptsDir, script))
		if err != nil {
			return fmt.Errorf("cannot read helper script %s: %w", script, err)
		}
		target := filepath.Join(binDir, name)
		if _, err := os.Lstat(target); err == nil {
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("cannot replace %s: %w", target, err)
			}
		}
		//nolint:gosec // helper scripts must be world-executable inside the container
		if err := os.WriteFile(target, content, 0o755); err != nil {
			return fmt.Errorf("cannot install helper script %s: %w", name, err)
		}
	}
	return nil
}

// createMachine materializes a machine on disk: rootfs flattened from the
// image store plus metadata. Runs in-process rootless; in root mode the
// same code runs inside the nspawn-helper subprocess as real root.
func createMachine(ctx context.Context, d dirs, m *machine) error {
	preserve := os.Geteuid() == 0
	return withLock(filepath.Join(d.machinesDir(), ".lock"), func() error {
		if containermanager.PathExists(d.metadataPath(m.Name)) {
			return fmt.Errorf("container %s already exists", m.Name)
		}
		if err := mkdirMachine(d, m.Name); err != nil {
			return err
		}

		store := ocistore.New(d.Cache)
		if digest, err := store.Resolve(m.Image); err == nil {
			m.ImageDigest = digest
		}
		if err := store.Flatten(ctx, m.Image, d.rootfsDir(m.Name), ocistore.FlattenOptions{
			PreserveOwnership: preserve,
			AllowDevices:      preserve,
		}); err != nil {
			_ = removeAllForce(d.machineDir(m.Name))
			return fmt.Errorf("failed to prepare container filesystem: %w", err)
		}

		if err := plantScripts(d.rootfsDir(m.Name), m.ScriptsDir); err != nil {
			_ = removeAllForce(d.machineDir(m.Name))
			return fmt.Errorf("failed to prepare container filesystem: %w", err)
		}

		if err := saveMachine(d, m); err != nil {
			_ = removeAllForce(d.machineDir(m.Name))
			return err
		}
		return nil
	})
}

// createRootful ships the machine definition to a root re-exec of
// ourselves: in-process extraction cannot be sudo-prefixed the way podman
// commands can, so the helper subcommand runs createMachine as real root.
func (n *Nspawn) createRootful(ctx context.Context, m *machine) error {
	return n.runHelper(ctx, m, "create-machine")
}

// buildMachine assembles the full machine metadata, including the start
// command, from the create options.
func (n *Nspawn) buildMachine(
	opts containermanager.CreateOptions,
	userEnv *userenv.UserEnvironment,
	d dirs,
	scriptsDir string,
) (*machine, error) {
	machineName := sanitizeMachineName(opts.ContainerName)

	homeToUse := userEnv.Home
	if opts.ContainerUserCustomHome != "" {
		homeToUse = opts.ContainerUserCustomHome
	}

	env := []string{
		"SHELL=" + filepath.Base(userEnv.Shell),
		"HOME=" + userEnv.Home,
		"container=systemd-nspawn",
		"TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo",
		"CONTAINER_ID=" + opts.ContainerName,
	}
	if opts.ContainerUserCustomHome != "" {
		env = append(env,
			"HOME="+opts.ContainerUserCustomHome,
			"DISTROBOX_HOST_HOME="+userEnv.Home,
		)
	}

	labels := map[string]string{
		"manager":                        "distrobox",
		"distrobox.unshare_groups":       strconv.Itoa(containermanager.Btoi(opts.UnshareGroups)),
		containermanager.VersionLabelKey: strconv.Itoa(containermanager.SchemaVersion),
	}
	for key, value := range opts.Labels {
		if _, managed := labels[key]; !managed {
			labels[key] = value
		}
	}

	mounts, err := n.buildMounts(opts, userEnv, scriptsDir)
	if err != nil {
		return nil, err
	}

	initArgs := []string{
		"--verbose",
		"--name", userEnv.User,
		"--user", userEnv.UserID,
		"--group", userEnv.GroupID,
		"--home", homeToUse,
		"--init", strconv.Itoa(containermanager.Btoi(opts.Init)),
		"--nvidia", strconv.Itoa(containermanager.Btoi(opts.Nvidia)),
		"--pre-init-hooks", opts.ContainerPreInitHook,
		"--additional-packages", strings.Join(opts.AdditionalPackages, " "),
		"--", opts.ContainerInitHook,
	}

	m := &machine{
		SchemaVersion: metadataSchemaVersion,
		Name:          opts.ContainerName,
		MachineName:   machineName,
		UnitName:      unitName(machineName),
		Image:         opts.ContainerImage,
		CreatedAt:     time.Now().UTC(),
		Labels:        labels,
		Home:          userEnv.Home,
		CustomHome:    opts.ContainerUserCustomHome,
		Hostname:      opts.ContainerHostname,
		Init:          opts.Init,
		Nvidia:        opts.Nvidia,
		ScriptsDir:    scriptsDir,
		Unshare: machineUnshare{
			IPC:     opts.UnshareIPC,
			NetNS:   opts.UnshareNetNS,
			Process: opts.UnshareProcess,
			Devsys:  opts.UnshareDevsys,
		},
		Env:      env,
		InitArgs: initArgs,
		Mounts:   mounts,
	}

	m.StartCommand = n.makeStartCommand(d, m, opts.AdditionalFlags)
	return m, nil
}

// buildMounts computes the recorded bind mounts for the machine, mirroring
// the podman/docker mount set where nspawn allows it.
//
//nolint:gocognit // mostly imperative conditional mount appending, mirroring makeCreateCommand
func (n *Nspawn) buildMounts(
	opts containermanager.CreateOptions,
	userEnv *userenv.UserEnvironment,
	scriptsDir string,
) ([]machineMount, error) {
	bind := func(source, destination, options string) machineMount {
		return machineMount{Source: source, Destination: destination, Options: options}
	}

	// idmapped binds require privileged mount operations that even the
	// managed user namespace mode does not delegate (systemd-mountfsd only
	// idmaps the rootfs tree). User-owned bind sources therefore appear
	// owned by "nobody" inside unprivileged machines until systemd grows
	// unprivileged idmapped binds.
	userOwned := ""

	mounts := []machineMount{
		bind("/tmp", "/tmp", ""),
		bind(userEnv.Home, userEnv.Home, userOwned),
	}

	// /run/host is nspawn's own integration directory (notify socket,
	// os-release exports), so unlike podman the host root cannot be bound
	// there wholesale; bind each top-level directory instead, mirroring
	// the podman+runc workaround (hostRootMountsForRunc).
	mounts = append(mounts, hostRootMounts()...)

	if opts.ContainerUserCustomHome != "" {
		mounts = append(mounts, bind(opts.ContainerUserCustomHome, opts.ContainerUserCustomHome, userOwned))
	}

	// Mount also the /var/home dir on ostree based systems, if $HOME was
	// not already set to /var/home/username.
	ostreeHome := "/var/home/" + userEnv.User
	if userEnv.Home != ostreeHome && containermanager.PathExists(ostreeHome) {
		mounts = append(mounts, bind(ostreeHome, ostreeHome, userOwned))
	}

	// XDG_RUNTIME_DIR keeps host app integration working; skipped for
	// initful machines so a dedicated systemd user session can be used.
	xdgRuntimeDir := "/run/user/" + userEnv.UserID
	if containermanager.PathExists(xdgRuntimeDir) && !opts.Init {
		mounts = append(mounts, bind(xdgRuntimeDir, xdgRuntimeDir, userOwned))
	}

	if !opts.UnshareDevsys {
		for _, device := range deviceBindCandidates() {
			if containermanager.PathExists(device) {
				mounts = append(mounts, bind(device, device, ""))
			}
		}
	}
	if opts.Nvidia {
		for _, device := range nvidiaBindCandidates() {
			if containermanager.PathExists(device) {
				mounts = append(mounts, bind(device, device, ""))
			}
		}
	}

	// In some systems /dev/shm is a symlink to /run/shm; mount the real
	// source so the link resolves inside the machine too.
	if containermanager.IsSymlink("/dev/shm") && !opts.UnshareIPC {
		if realPath, err := filepath.EvalSymlinks("/dev/shm"); err == nil {
			mounts = append(mounts, bind(realPath, realPath, ""))
		}
	}

	// Forward RedHat subscription-manager configuration, as the podman and
	// docker providers do.
	rhelSubscriptionMounts := []machineMount{
		bind("/etc/pki/entitlement/", "/run/secrets/etc-pki-entitlement", "ro"),
		bind("/etc/rhsm/", "/run/secrets/rhsm", "ro"),
		bind("/etc/yum.repos.d/redhat.repo", "/run/secrets/redhat.repo", "ro"),
	}
	for _, mount := range rhelSubscriptionMounts {
		if containermanager.PathExists(mount.Source) {
			mounts = append(mounts, mount)
		}
	}

	// Keep host connectivity files in sync inside the machine. /etc/hostname
	// is only shared when the machine hostname matches the host's.
	if !opts.UnshareNetNS {
		netFiles := []string{"/etc/hosts"}
		hostname, _ := os.Hostname()
		if opts.ContainerHostname == hostname {
			netFiles = append(netFiles, "/etc/hostname")
		}
		for _, netFile := range netFiles {
			if containermanager.PathExists(netFile) {
				mounts = append(mounts, bind(netFile, netFile, "ro"))
			}
		}
	}

	if opts.Nopasswd {
		mounts = append(mounts, bind("/dev/null", "/run/.nopasswd", "ro"))
	}

	// Signal rootless mode explicitly so distrobox-init does not rely
	// solely on heuristics.
	if !n.root {
		mounts = append(mounts, bind("/dev/null", "/run/.distrobox.rootless", "ro"))
	}

	userMounts, err := translateVolumes(opts.AdditionalVolumes, n.verbose)
	if err != nil {
		return nil, err
	}
	return append(mounts, userMounts...), nil
}

// hostRootMounts returns per-directory binds of the host root under
// /run/host. nspawn reserves /run/host itself for its container interface
// (notify socket, os-release exports), so distrobox's whole-root bind used
// with podman/docker would break machine startup.
func hostRootMounts() []machineMount {
	entries, err := os.ReadDir("/")
	if err != nil {
		return nil
	}

	mounts := make([]machineMount, 0, len(entries))
	for _, entry := range entries {
		// Skip hidden entries (shell glob /* doesn't match dotfiles)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		rootdir := "/" + entry.Name()
		info, err := os.Lstat(rootdir)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		mounts = append(mounts, machineMount{Source: rootdir, Destination: "/run/host" + rootdir})
	}
	return mounts
}

// translateVolumes converts docker/podman-style volume specs
// (src:dest[:options]) into nspawn bind mounts. Options that only make
// sense on container engines (propagation, selinux relabeling) are dropped;
// anything else is a hard error rather than a silent misconfiguration.
func translateVolumes(volumes []string, verbose bool) ([]machineMount, error) {
	mounts := make([]machineMount, 0, len(volumes))
	for _, volume := range volumes {
		parts := strings.Split(volume, ":")
		//nolint:mnd // src:dest[:options]
		if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid volume %q: expected src:dest[:options]", volume)
		}

		mount := machineMount{Source: parts[0], Destination: parts[1]}
		//nolint:mnd // src:dest:options
		if len(parts) == 3 {
			for _, option := range strings.Split(parts[2], ",") {
				switch option {
				case "ro":
					mount.Options = "ro"
				case "rw", "":
					// nspawn binds are read-write by default.
				case "rslave", "rshared", "rprivate", "slave", "shared", "private", "z", "Z":
					if verbose {
						fmt.Fprintf(os.Stderr, "Warning: volume option %q has no nspawn equivalent, ignoring\n", option)
					}
				default:
					return nil, fmt.Errorf("volume option %q of %q is not supported with the nspawn container manager", option, volume)
				}
			}
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
}

// makeStartCommand builds the full argv that starts the machine: a
// systemd-run transient service wrapping systemd-nspawn with distrobox-init
// as the machine's PID 1 payload. distrobox-init exec's systemd itself for
// initful containers, so --boot is never needed.
func (n *Nspawn) makeStartCommand(d dirs, m *machine, additionalFlags []string) []string {
	cmd := []string{"systemd-run"}
	if !n.root {
		cmd = append(cmd, "--user")
	}
	initLog := d.initLogPath(m.Name)
	cmd = append(cmd,
		"--unit="+m.UnitName,
		"--collect",
		"--quiet",
		"--property=Delegate=yes",
		"--property=KillMode=mixed",
		"--property=TimeoutStopSec=10",
		"--property=StandardOutput=append:"+initLog,
		"--property=StandardError=append:"+initLog,
	)
	if m.Init {
		// SIGRTMIN+3 is systemd's orderly-shutdown signal; distrobox-init
		// exec's systemd as PID 1 for initful machines.
		cmd = append(cmd, "--property=KillSignal=SIGRTMIN+3")
	}

	cmd = append(cmd, "--",
		"systemd-nspawn",
		"--quiet",
		"--keep-unit",
		"--register=yes",
		"--console=pipe",
		"--notify-ready=no",
		"--directory="+d.rootfsDir(m.Name),
		"--machine="+m.MachineName,
	)

	if n.root {
		cmd = append(cmd, "--private-users=no")
	} else {
		// Unprivileged operation requires the managed user namespace mode:
		// systemd-nsresourced allocates the UID range and systemd-mountfsd
		// idmaps the caller-owned rootfs (caller <-> container root).
		cmd = append(cmd, "--private-users=managed")
	}

	if m.Hostname != "" {
		cmd = append(cmd, "--hostname="+m.Hostname)
	}

	if m.Unshare.NetNS {
		cmd = append(cmd, "--private-network")
	} else {
		// Host networking is nspawn's default (rootful); unprivileged
		// machines always get a private namespace and are networked via
		// pasta at start. replace-host copies the host's resolv.conf in —
		// a bind would be a single-file bind, which unprivileged nspawn
		// cannot do.
		cmd = append(cmd, "--resolv-conf=replace-host")
	}

	if m.Init {
		cmd = append(cmd, "--kill-signal=SIGRTMIN+3")
	}

	for _, mount := range m.Mounts {
		cmd = append(cmd, bindFlag(mount))
	}

	// systemd (or journald) tries to set ACLs on this path; give the
	// machine a private one like the podman/docker unnamed volume.
	cmd = append(cmd, "--tmpfs=/var/log/journal")

	for _, env := range m.Env {
		cmd = append(cmd, "--setenv="+env)
	}

	// Additional flags are appended raw to the nspawn argv; with this
	// backend they use systemd-nspawn syntax.
	cmd = append(cmd, additionalFlags...)

	cmd = append(cmd, "/usr/bin/entrypoint")
	cmd = append(cmd, m.InitArgs...)
	return cmd
}

// bindFlag renders one recorded mount as an nspawn --bind/--bind-ro flag.
func bindFlag(mount machineMount) string {
	flag := "--bind="
	options := mount.Options
	if options == "ro" || strings.HasPrefix(options, "ro,") {
		flag = "--bind-ro="
		options = strings.TrimPrefix(strings.TrimPrefix(options, "ro"), ",")
	}
	spec := mount.Source + ":" + mount.Destination
	if options != "" {
		spec += ":" + options
	}
	return flag + spec
}
