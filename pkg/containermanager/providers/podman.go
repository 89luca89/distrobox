package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type Podman struct {
	root        bool
	sudoCommand string
	verbose     bool
}

var _ containermanager.ContainerManager = &Podman{}

func NewPodman(root bool, sudoCommand string, verbose bool) *Podman {
	return &Podman{
		sudoCommand: sudoCommand,
		root:        root,
		verbose:     verbose,
	}
}

func (p *Podman) Name() string {
	return "podman"
}

// podmanContainer represents the JSON output from `podman ps --format json`.
type podmanContainer struct {
	ID     string            `json:"ID"`
	Image  string            `json:"Image"`
	Names  []string          `json:"Names"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels"`
}

func (p *Podman) ListContainers(ctx context.Context) ([]containermanager.Container, error) {
	args := []string{"ps", "-a", "--no-trunc", "--format", "json"}
	out, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return nil, err
	}
	return parsePodmanContainerList(out)
}

func (p *Podman) Create(
	ctx context.Context,
	opts containermanager.CreateOptions,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)

	scriptsDir, err := insidedistrobox.ProvisionScripts()
	if err != nil {
		return fmt.Errorf("failed to provision scripts: %w", err)
	}

	// ensure custom home dir exists, if needed
	if opts.ContainerUserCustomHome != "" && !pathExists(opts.ContainerUserCustomHome) {
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.MkdirAll(opts.ContainerUserCustomHome, 0755); err != nil {
			return fmt.Errorf("failed to create custom home directory: %w", err)
		}
	}

	cmd := p.makeCreateCommand(
		ctx,
		opts.ContainerName,
		opts.ContainerImage,
		opts.AdditionalFlags,
		opts.ContainerHostname,
		opts.AdditionalPackages,
		opts.AdditionalVolumes,
		opts.ContainerUserCustomHome,
		opts.ContainerPlatform,
		opts.Nopasswd,
		opts.Init,
		opts.ContainerPreInitHook,
		opts.ContainerInitHook,
		opts.Nvidia,
		opts.UnshareDevsys,
		opts.UnshareGroups,
		opts.UnshareIPC,
		opts.UnshareNetNS,
		opts.UnshareProcess,
		userEnv,
		filepath.Join(scriptsDir, "distrobox-init"),
		filepath.Join(scriptsDir, "distrobox-export"),
		filepath.Join(scriptsDir, "distrobox-host-exec"),
	)

	_, err = p.run(ctx, cmd, runOptions{DryRun: opts.DryRun})
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	return nil
}

// makeCreateCommand builds the podman create command with all necessary options.
//
//nolint:gocognit,funlen // ignore cognitive complexity here, the function is mostly imperative option appending
func (p *Podman) makeCreateCommand(
	ctx context.Context,
	containerName string,
	containerImage string,
	containerAdditionalFlags []string,
	containerHostname string,
	containerAdditionalPackages []string,
	containerAdditionalVolumes []string,
	containerUserCustomHome string,
	containerPlatform string,
	nopasswd bool,
	init bool,
	containerPreInitHook string,
	containerInitHook string,
	nvidia bool,
	unshareDevsys bool,
	unshareGroups bool,
	unshareIPC bool,
	unshareNetNS bool,
	unshareProcess bool,
	userEnv *userenv.UserEnvironment,
	distroboxInitPath string,
	distroboxExportPath string,
	distroboxHostexecPath string,
) []string {
	containerManager := p.Name()

	containerUserHome := userEnv.Home
	containerUserName := userEnv.User
	containerUserUID := userEnv.UserID
	containerUserGID := userEnv.GroupID
	shellFilepath := filepath.Base(userEnv.Shell)

	var options []string

	if containerPlatform != "" {
		options = append(options, "--platform="+containerPlatform)
	}
	options = append(options, "--hostname", containerHostname)
	options = append(options, "--name", containerName)
	options = append(options, "--privileged")
	options = append(options, "--security-opt", "label=disable")
	options = append(options, "--security-opt", "apparmor=unconfined")
	options = append(options, "--pids-limit=-1")
	options = append(options, "--user", "root:root")

	if !unshareIPC {
		options = append(options, "--ipc", "host")
	}

	if !unshareNetNS {
		options = append(options, "--network", "host")
	}

	if !unshareProcess {
		options = append(options, "--pid", "host")
	}

	// Mount useful stuff inside the container.
	// We also mount host's root filesystem to /run/host, to be able to syphon
	// dynamic configurations from the host.
	//
	// Mount user home, dev and host's root inside container.
	// This grants access to external devices like usb webcams, disks and so on.
	//
	// Mount also the distrobox-init utility as the container entrypoint.
	// Also mount in the container the distrobox-export and distrobox-host-exec
	// utilities.

	options = append(options, "--label", "manager=distrobox")
	options = append(
		options,
		"--label",
		fmt.Sprintf("distrobox.unshare_groups=%d", btoi(unshareGroups)),
	)
	options = append(options, "--env", fmt.Sprintf("SHELL=%s", shellFilepath))
	options = append(options, "--env", fmt.Sprintf("HOME=%s", containerUserHome))
	options = append(options, "--env", fmt.Sprintf("container=%s", containerManager))
	options = append(
		options,
		"--env",
		"TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo",
	)
	options = append(options, "--env", fmt.Sprintf("CONTAINER_ID=%s", containerName))
	options = append(options, "--volume", "/tmp:/tmp:rslave")
	options = append(options, "--volume", fmt.Sprintf("%s:%s", distroboxExportPath, "/usr/bin/distrobox-export:ro"))
	options = append(
		options,
		"--volume",
		fmt.Sprintf("%s:%s", distroboxHostexecPath, "/usr/bin/distrobox-host-exec:ro"),
	)
	options = append(options, "--volume", fmt.Sprintf("%s:%s:rslave", containerUserHome, containerUserHome))

	// Due to breaking change in https://github.com/opencontainers/runc/commit/d4b670fca6d0ac606777376440ffe49686ce15f4
	// now we cannot mount /:/run/host as before, as it will try to mount RO partitions as RW thus breaking things.
	// This will ensure we will mount directories one-by-one thus avoiding this problem.
	//
	// This happens ONLY with podman+runc, docker and lilipod are unaffected,
	// so let's do this only if we have podman AND runc.
	if p.usesRunc(ctx) {
		options = append(options, hostRootMountsForRunc(ctx)...)
	} else {
		options = append(options, "--volume", "/:/run/host/:rslave")
	}

	if !unshareDevsys {
		options = append(options, "--volume", "/dev:/dev:rslave")
		options = append(options, "--volume", "/sys:/sys:rslave")
	}

	// This fix is needed so that the container can have a separate devpts instance
	// inside
	// This will mount an empty /dev/pts, and the init will take care of mounting
	// a new devpts with the proper flags set
	// Mounting an empty volume there, is needed in order to ensure that no package
	// manager tries to fiddle with /dev/pts/X that would not be writable by them
	//
	// This implementation is done this way in order to be compatible with both
	// docker and podman
	if !unshareDevsys {
		options = append(options, "--volume", "/dev/pts")
		options = append(options, "--volume", "/dev/null:/dev/ptmx")
	}

	// This fix is needed as on Selinux systems, the host's selinux sysfs directory
	// will be mounted inside the rootless container.
	//
	// This works around this and allows the rootless container to work when selinux
	// policies are installed inside it.
	//
	// Ref. Podman issue 4452:
	//    https://github.com/containers/podman/issues/4452
	if pathExists("/sys/fs/selinux") {
		options = append(options, "--volume", "/sys/fs/selinux")
	}

	// This fix is needed as systemd (or journald) will try to set ACLs on this
	// path. For now overlayfs and fuse.overlayfs are not compatible with ACLs
	//
	// This works around this using an unnamed volume so that this path will be
	// mounted with a normal non-overlay FS, allowing ACLs and preventing errors.
	//
	// This work around works in conjunction with distrobox-init's package manager
	// setups.
	// So that we can use pre/post hooks for package managers to present to the
	// systemd install script a blank path to work with, and mount the host's
	// journal path afterwards.
	options = append(options, "--volume", "/var/log/journal")

	// In some systems, for example using sysvinit, /dev/shm is a symlink
	// to /run/shm, instead of the other way around.
	// Resolve this detecting if /dev/shm is a symlink and mount original
	// source also in the container.
	if isSymlink("/dev/shm") && !unshareIPC {
		realPath, _ := filepath.EvalSymlinks("/dev/shm")
		options = append(options, "--volume", fmt.Sprintf("%s:%s", realPath, realPath))
	}

	// Ensure support forwarding of RedHat subscription-manager
	// This is needed in order to have a working subscription forwarded into the container,
	// this will ensure that rhel-9-for-x86_64-appstream-rpms and rhel-9-for-x86_64-baseos-rpms repos
	// will be available in the container, so that distrobox-init will be able to
	// install properly all the dependencies like mesa drivers.
	//
	// /run/secrets is a standard location for RHEL containers, that is being pointed by
	// /etc/rhsm-host by default.
	rhelSubscriptionFiles := []string{
		"/etc/pki/entitlement/:/run/secrets/etc-pki-entitlement:ro",
		"/etc/rhsm/:/run/secrets/rhsm:ro",
		"/etc/yum.repos.d/redhat.repo:/run/secrets/redhat.repo:ro",
	}
	for _, rhelFile := range rhelSubscriptionFiles {
		parts := strings.Split(rhelFile, ":")
		if pathExists(parts[0]) {
			options = append(options, "--volume", rhelFile)
		}
	}

	// If we have a custom home to use,
	//	1- override the HOME env variable
	//	2- export the DISTROBOX_HOST_HOME env variable pointing to original HOME
	// 	3- mount the custom home inside the container.
	if containerUserCustomHome != "" {
		options = append(options, "--env", fmt.Sprintf("HOME=%s", containerUserCustomHome))
		options = append(options, "--env", fmt.Sprintf("DISTROBOX_HOST_HOME=%s", containerUserHome))
		options = append(
			options,
			"--volume",
			fmt.Sprintf("%s:%s:rslave", containerUserCustomHome, containerUserCustomHome),
		)
	}

	// Mount also the /var/home dir on ostree based systems
	// do this only if $HOME was not already set to /var/home/username
	homePath := fmt.Sprintf("/var/home/%s", containerUserName)
	if containerUserHome != homePath && pathExists(homePath) {
		options = append(options, "--volume", fmt.Sprintf("%s:%s:rslave", homePath, homePath))
	}

	// Mount also the XDG_RUNTIME_DIR to ensure functionality of the apps.
	// This is skipped in case of initful containers, so that a dedicated
	// systemd user session can be used.
	xdgRuntimeDir := fmt.Sprintf("/run/user/%s", containerUserUID)
	if pathExists(xdgRuntimeDir) && !init {
		options = append(options, "--volume", fmt.Sprintf("%s:%s:rslave", xdgRuntimeDir, xdgRuntimeDir))
	}

	// These are dynamic configs needed by the container to function properly
	// and integrate with the host
	//
	// We're doing this now instead of inside the init because some distros will
	// have symlinks places for these files that use absolute paths instead of
	// relative paths.
	// This is the bare minimum to ensure connectivity inside the container.
	// These files, will then be kept updated by the main loop every 15 seconds.
	if !unshareNetNS {
		netFiles := []string{
			"/etc/hosts",
			"/etc/resolv.conf",
		}

		// If container_hostname is custom, we skip mounting /etc/hostname, else
		// we want to keep it in sync
		hostname, _ := os.Hostname()
		if containerHostname == hostname {
			netFiles = append(netFiles, "/etc/hostname")
		}

		for _, netFile := range netFiles {
			if pathExists(netFile) {
				options = append(options, "--volume", fmt.Sprintf("%s:%s:ro", netFile, netFile))
			}
		}
	}

	// if nopasswd, then let the init know via a mountpoint
	if nopasswd {
		options = append(options, "--volume", "/dev/null:/run/.nopasswd:ro")
	}

	// Add additional flags
	options = append(options, containerAdditionalFlags...)

	// Add additional volumes
	for _, vol := range containerAdditionalVolumes {
		options = append(options, "--volume", vol)
	}

	// Podman-specific flags
	// If possible, always prefer crun, as it allows keeping original groups.
	// useful for rootless containers.
	if commandExists("crun") {
		options = append(options, "--runtime=crun")
	}
	options = append(options, "--annotation", "run.oci.keep_original_groups=1")
	options = append(options, "--ulimit", "host")

	// For init containers, use podman's systemd support
	if init {
		options = append(options, "--systemd=always")
	}

	// Use keep-id only if going rootless.
	if !p.root {
		if p.supportsKeepIDSize(ctx, containerImage) {
			options = append(options, "--userns", "keep-id:size=65536")
		} else {
			options = append(options, "--userns", "keep-id")
		}
	}

	// Now execute the entrypoint, refer to `distrobox-init -h` for instructions
	// containerManager
	// Be aware that entrypoint corresponds to distrobox-init, the copying of it
	// inside the container is moved to distrobox-enter, in the start phase.
	// This is done to make init, export and host-exec location independent from
	// the host, and easier to upgrade.
	//
	// We set the entrypoint _before_ running the container image so that
	// we can override any user provided entrypoint if need be
	options = append(options, "--volume", fmt.Sprintf("%s:%s", distroboxInitPath, "/usr/bin/entrypoint:ro"))
	options = append(options, "--entrypoint", "/usr/bin/entrypoint")

	// Build the rest of the arguments for distrobox-init
	//
	// The arguments will be passed to distrobox-init as the entrypoint
	homeToUse := containerUserHome
	if containerUserCustomHome != "" {
		homeToUse = containerUserCustomHome
	}
	args := []string{
		"--verbose",
		"--name", containerUserName,
		"--user", containerUserUID,
		"--group", containerUserGID,
		"--home", homeToUse,
		"--init", strconv.Itoa(btoi(init)),
		"--nvidia", strconv.Itoa(btoi(nvidia)),
		"--pre-init-hooks", containerPreInitHook,
		"--additional-packages", strings.Join(containerAdditionalPackages, " "),
		"--", containerInitHook,
	}

	// Final assembly of the command
	// podman create [options] image [args...]
	//nolint:mnd // 2 is fine here, it's "create" and image
	cmd := make([]string, 0, len(options)+len(args)+2)
	cmd = append(cmd, "create")
	cmd = append(cmd, options...)
	cmd = append(cmd, containerImage)
	cmd = append(cmd, args...)

	return cmd
}

func (p *Podman) Exists(ctx context.Context, containerName string) bool {
	args := []string{"inspect", "--type", "container", containerName}
	_, err := p.run(ctx, args, runOptions{})
	return err == nil
}

func (p *Podman) run(ctx context.Context, args []string, opts runOptions) (string, error) {
	command := p.Name()
	if p.root {
		args = append([]string{command}, args...)
		command = p.sudoCommand
	}

	if opts.DryRun {
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(command, strings.Join(args, " "))
		return "", nil
	}

	cmd := exec.CommandContext(ctx, command, args...)

	if opts.Interactive {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("error running the interactive command :%w", err)
		}
		return "", nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		captured := strings.TrimSpace(stderr.String())
		if captured != "" {
			return "", fmt.Errorf("command execution failed: %s", captured)
		}
		return "", fmt.Errorf("command execution failed: %w", err)
	}
	return stdout.String(), nil
}

func (p *Podman) Enter(
	ctx context.Context,
	options containermanager.EnterOptions,
	progress *ui.Progress,
	printer *ui.Printer,
) error {
	userEnv := userenv.LoadUserEnvironment(ctx)
	user := userEnv.User

	command, config, err := p.generateEnterCommand(
		ctx,
		options.ContainerName,
		options.AdditionalFlags,
		options.NoTTY,
		options.NoWorkDir,
		options.CleanPath,
		options.Verbose,
	)
	if err != nil {
		return fmt.Errorf("err: %w", err)
	}

	commandArgs := buildCommandArgs(options.CustomCommand, user, options.NoTTY, config.UnshareGroups)

	if options.DryRun {
		command = append(command, commandArgs...)
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(p.Name() + " " + strings.Join(command, "\n"))

		return nil
	}

	inspectResult, err := p.InspectContainer(ctx, options.ContainerName)
	if err != nil || inspectResult.ContainerStatus != RunningStatus {
		logTimestamp := timestampNow()

		_ = p.startContainer(ctx, options.ContainerName, progress)

		// Monitor logs for setup completion
		if err := p.waitForSetup(ctx, options.ContainerName, logTimestamp, progress, printer); err != nil {
			return err
		}

		progress.Finalize("Container Setup Complete!")
	}

	_, _ = p.run(ctx, append(command, commandArgs...), runOptions{Interactive: !options.NoTTY})

	return nil
}

func (p *Podman) ImageExists(ctx context.Context, imageName string) bool {
	args := []string{"inspect", "--type", "image", "--format", "json", imageName}
	output, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return false
	}

	var inspects []inspectOutput
	if err := json.Unmarshal([]byte(output), &inspects); err != nil {
		return false
	}

	if len(inspects) == 0 {
		return false
	}

	return true
}

func (p *Podman) PullImage(ctx context.Context, imageName string, platform string) error {
	var args []string
	if platform != "" {
		args = []string{"pull", "--platform", platform, imageName}
	} else {
		args = []string{"pull", imageName}
	}
	_, err := p.run(ctx, args, runOptions{TailLogs: true})
	return err
}

func (p *Podman) Remove(
	ctx context.Context,
	containerName string,
	options containermanager.RmOptions,
) error {
	args := []string{"rm"}
	if options.Force {
		args = append(args, "--force")
	}

	args = append(args, []string{"--volumes", containerName}...)

	_, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return fmt.Errorf("error removing the container: %w", err)
	}

	if options.RemoveHome {
		err = os.RemoveAll(options.ContainerHome)
		if err != nil {
			return fmt.Errorf("error removing home directory %s: %w", options.ContainerHome, err)
		}
	}

	return nil
}

func (p *Podman) Stop(ctx context.Context, containerNames []string) error {
	args := []string{"stop"}
	args = append(args, containerNames...)

	_, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return fmt.Errorf("error stopping containers: %w", err)
	}
	return nil
}

func parsePodmanContainerList(output string) ([]containermanager.Container, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var containers []containermanager.Container

	var pc []podmanContainer
	if err := json.Unmarshal([]byte(output), &pc); err != nil {
		return nil, fmt.Errorf("failed to parse container JSON: %w", err)
	}

	const containerIDMaxLength = 12

	for _, c := range pc {
		id := c.ID
		if len(id) > containerIDMaxLength {
			id = id[:containerIDMaxLength]
		}

		containers = append(containers, containermanager.Container{
			ID:     id,
			Image:  c.Image,
			Name:   c.Names[0],
			Status: c.Status,
			Labels: c.Labels,
		})
	}
	return containers, nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// supportsKeepIDSize tests whether podman supports the keep-id:size= userns option
// by attempting a quick container run. Older podman versions do not support the size suboption.
func (p *Podman) supportsKeepIDSize(ctx context.Context, image string) bool {
	_, err := p.run(ctx, []string{"run", "--rm", "--userns=keep-id:size=65536", image, "/bin/true"}, runOptions{})
	if err == nil {
		return true
	}
	// If the error mentions "size" as unknown option, the feature is not supported.
	// Other errors (e.g., /bin/true not found exit 127) mean the option itself was accepted.
	return !strings.Contains(err.Error(), "unknown option specified: \"size\"")
}

// usesRunc detects whether podman is configured to use runc as the OCI runtime.
// Equivalent to shell: podman info 2>/dev/null | grep -q runc.
func (p *Podman) usesRunc(ctx context.Context) bool {
	out, err := p.run(ctx, []string{"info"}, runOptions{})
	if err != nil {
		return false
	}

	return strings.Contains(out, "runc")
}

// hostRootMountsForRunc returns per-directory volume mounts for /run/host,
// working around a runc breaking change that prevents mounting / as a whole.
func hostRootMountsForRunc(ctx context.Context) []string {
	var mounts []string

	entries, err := os.ReadDir("/")
	if err != nil {
		return mounts
	}

	for _, entry := range entries {
		// Skip hidden directories (shell glob /* doesn't match dotfiles)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		rootdir := "/" + entry.Name()

		// Skip symlinks
		info, err := os.Lstat(rootdir)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		target := fmt.Sprintf("%s:/run/host%s", rootdir, rootdir)
		if isMountReadOnly(ctx, rootdir) {
			mounts = append(mounts, "--volume", target+":ro,rslave")
		} else {
			mounts = append(mounts, "--volume", target+":rslave")
		}
	}

	return mounts
}

// isMountReadOnly checks if the given path resides on a read-only mount
// by parsing findmnt output. Equivalent to shell:
// findmnt --notruncate --noheadings --list --output OPTIONS --target "$path" | tr ',' '\n' | grep -q "^ro$".
func isMountReadOnly(ctx context.Context, path string) bool {
	out, err := exec.CommandContext(
		ctx,
		"findmnt", "--notruncate", "--noheadings", "--list",
		"--output", "OPTIONS", "--target", path,
	).Output()
	if err != nil {
		return false
	}

	for _, opt := range strings.Split(string(out), ",") {
		if strings.TrimSpace(opt) == "ro" {
			return true
		}
	}

	return false
}

func (p *Podman) Commit(ctx context.Context, containerID string, tag string) error {
	_, err := p.run(ctx, []string{"container", "commit", containerID, tag}, runOptions{})
	return err
}

func (p *Podman) InspectContainer(ctx context.Context, containerName string) (*containermanager.InspectResult, error) {
	config := containermanager.InspectResult{}
	args := []string{"inspect", "--type", "container", "--format", "json", containerName}
	output, err := p.run(ctx, args, runOptions{})
	if err != nil {
		return nil, err
	}

	var inspects []inspectOutput
	if err := json.Unmarshal([]byte(output), &inspects); err != nil {
		return nil, errors.New("error marshaling json into containerInspect")
	}

	if len(inspects) == 0 {
		return nil, errors.New("container not found")
	}

	inspect := inspects[0]
	config.ContainerStatus = inspect.State.Status

	// Check for unshare_groups label
	if v, ok := inspect.Config.Labels["distrobox.unshare_groups"]; ok && v == "1" {
		config.UnshareGroups = true
	}

	// Extract HOME and PATH from container env
	for _, env := range inspect.Config.Env {
		if strings.HasPrefix(env, "HOME=") {
			config.ContainerHome = strings.TrimPrefix(env, "HOME=")
		} else if strings.HasPrefix(env, "PATH=") {
			config.ContainerPath = strings.TrimPrefix(env, "PATH=")
		}
	}

	return &config, nil
}

func (p *Podman) generateEnterCommand(
	ctx context.Context,
	containerName string,
	additionalFlags string,
	noTTY bool,
	noWorkDir bool,
	cleanPath bool,
	verbose bool,
) ([]string, *containermanager.InspectResult, error) {
	cmd := []string{}

	if verbose {
		cmd = append(cmd, "--log-level", "debug")
	}

	cmd = append(cmd, "exec")
	cmd = append(cmd, "--interactive")
	cmd = append(cmd, "--detach-keys=")

	containerConfig, err := p.InspectContainer(ctx, containerName)
	if err != nil {
		// TODO handle missing container
		return nil, nil, err
	}
	// User selection
	if containerConfig.UnshareGroups {
		cmd = append(cmd, "--user=root")
	} else {
		userEnv := userenv.LoadUserEnvironment(ctx)
		username := userEnv.User
		cmd = append(cmd, fmt.Sprintf("--user=%s", username))
	}

	// TTY allocation — auto-detect headless mode like the shell version:
	// if stdin or stdout is not a terminal, skip --tty.
	if !noTTY && isTTY() {
		cmd = append(cmd, "--tty")
	}

	// Working directory
	workdir, err := getWorkDir(containerConfig.ContainerHome, noWorkDir)
	if err != nil {
		return nil, nil, err
	}

	cmd = append(cmd, fmt.Sprintf("--workdir=%s", workdir))

	executablePath, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting the executable path: %w", err)
	}

	// Environment variables
	cmd = append(cmd, fmt.Sprintf("--env=CONTAINER_ID=%s", containerName))
	cmd = append(cmd, fmt.Sprintf("--env=DISTROBOX_ENTER_PATH=%s", executablePath))

	for _, env := range filterEnvVars() {
		cmd = append(cmd, fmt.Sprintf("--env=%s", env))
	}
	// PATH handling
	containerPaths := buildContainerPath(cleanPath, os.Getenv("PATH"), containerConfig.ContainerPath)
	cmd = append(cmd, fmt.Sprintf("--env=PATH=%s", containerPaths))

	// XDG_DATA_DIRS
	xdgDataDirs := buildXDGPaths("XDG_DATA_DIRS", []string{"/usr/local/share", "/usr/share"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_DIRS=%s", xdgDataDirs))

	// XDG_CONFIG_DIRS
	xdgConfigDirs := buildXDGPaths("XDG_CONFIG_DIRS", []string{"/etc/xdg"})
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_DIRS=%s", xdgConfigDirs))

	// XDG home directories
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CACHE_HOME=%s/.cache", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_CONFIG_HOME=%s/.config", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_DATA_HOME=%s/.local/share", containerConfig.ContainerHome))
	cmd = append(cmd, fmt.Sprintf("--env=XDG_STATE_HOME=%s/.local/state", containerConfig.ContainerHome))

	// Additional flags
	if len(additionalFlags) > 0 {
		cmd = append(cmd, strings.Fields(additionalFlags)...)
	}

	// Container name
	cmd = append(cmd, containerName)

	return cmd, containerConfig, nil
}

func (p *Podman) startContainer(ctx context.Context, containerName string, progress *ui.Progress) error {
	// Start the container
	_, err := p.run(ctx, []string{"start", containerName}, runOptions{Interactive: true})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Check if container is running after start
	inspectResult, err := p.InspectContainer(ctx, containerName)
	if err != nil || inspectResult.ContainerStatus != RunningStatus {
		logs, err := p.run(ctx, []string{"logs", containerName}, runOptions{})
		if err != nil {
			return fmt.Errorf("could not inspect container logs: %w", err)
		}
		return fmt.Errorf("could not start entrypoint.\n%s", logs)
	}

	progress.Next("Starting container...")

	userEnv := userenv.LoadUserEnvironment(ctx)

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(userEnv.Home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "distrobox")

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil { //nolint:gosec // we need this writable by everybody
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return nil
}

func (p *Podman) waitForSetup(
	ctx context.Context,
	containerName string,
	since string,
	progress *ui.Progress,
	printer *ui.Printer,
) error {
	for {
		// Check container is still running
		inspectResult, err := p.InspectContainer(ctx, containerName)
		if err != nil || inspectResult.ContainerStatus != RunningStatus {
			printer.PrintError("\nContainer Setup Failure!")
			return fmt.Errorf("container stopped during setup: %w", err)
		}

		// Get logs
		nextSince := timestampNow()
		output, err := p.run(ctx, []string{"logs", "--since", since, containerName}, runOptions{})
		if err != nil {
			time.Sleep(100 * time.Millisecond) //nolint:mnd // TODO refactor sleeps
			continue
		}
		since = nextSince

		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			switch {
			case strings.HasPrefix(line, "+"):
				// Ignore logging commands
				continue

			case strings.HasPrefix(line, "Error:"):
				progress.Fail()
				printer.PrintError(line)
				return fmt.Errorf("container setup error: %s", line)

			case strings.HasPrefix(line, "Warning:"):
				printer.PrintWarning(line)

			case strings.HasPrefix(line, "distrobox:"):
				parts := strings.SplitN(line, " ", Two)
				if len(parts) > 1 {
					progress.Done()
					progress.Next("%s", parts[1])
				}

			case strings.HasPrefix(line, "container_setup_done"):
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond) //nolint:mnd // TODO refactor sleeps
	}
}
