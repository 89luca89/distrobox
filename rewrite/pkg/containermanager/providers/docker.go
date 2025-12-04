package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

type Docker struct {
	root        bool
	sudoCommand string
	verbose     bool
}

var _ containermanager.ContainerManager = &Docker{}

func NewDocker(root bool, sudoCommand string, verbose bool) *Docker {
	return &Docker{
		sudoCommand: sudoCommand,
		root:        root,
		verbose:     verbose,
	}
}

func (d *Docker) Name() string {
	return "docker"
}

// dockerContainer represents the JSON output from `docker ps --format json`.
type dockerContainer struct {
	ID     string `json:"ID"`
	Image  string `json:"Image"`
	Names  string `json:"Names"`
	Status string `json:"Status"`
	Labels string `json:"Labels"`
}

func (d *Docker) ListContainers(ctx context.Context) ([]containermanager.Container, error) {
	args := []string{"ps", "-a", "--no-trunc", "--format", "json"}
	out, err := d.run(ctx, args, false)
	if err != nil {
		return nil, err
	}
	return parseContainerList(out)
}

func (d *Docker) Create(
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

	cmd := d.makeCreateCommand(
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

	_, err = d.run(ctx, cmd, opts.DryRun)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	return nil
}

// makeCreateCommand builds the docker create command with all necessary options.
//
//nolint:gocognit,funlen // ignore cognitive complexity here, the function is mostly imperative option appending
func (d *Docker) makeCreateCommand(
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
	containerManager := d.Name()

	containerUserHome := userEnv.Home
	containerUserName := userEnv.User
	containerUserUID := userEnv.UserID
	containerUserGID := userEnv.GroupID
	shellFilepath := filepath.Base(userEnv.Shell)

	var options []string

	if containerPlatform != "" {
		options = append(options, "--platform", containerPlatform)
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
	options = append(options, "--volume", "/:/run/host/:rslave")

	if !unshareDevsys {
		options = append(options, "--volume", "/dev:/dev:rslave")
		options = append(options, "--volume", "/sys:/sys:rslave")
	}

	// In case of initful containers, we implement a series of mountpoint in order
	// for systemd to work properly inside a container.
	// The following are a flag-based implementation of what podman's --systemd flag
	// does under the hood, as explained in their docs here:
	//   https://docs.podman.io/en/latest/markdown/options/systemd.html
	//
	// set the default stop signal to SIGRTMIN+3.
	// mount tmpfs file systems on the following directories
	//	/run
	//	/run/lock
	//	/tmp
	//	/var/lib/journal
	//	/sys/fs/cgroup/systemd <- this one is done by cgroupns=host
	if init && strings.Contains(containerManager, "docker") {
		// In case of docker we're actually rootful, so we need to use hosts cgroups
		options = append(options, "--cgroupns", "host")
		// In case of all other non-podman container managers, we can do this
		options = append(options, "--stop-signal", "SIGRTMIN+3")
		options = append(options, "--mount", "type=tmpfs,destination=/run")
		options = append(options, "--mount", "type=tmpfs,destination=/run/lock")
		options = append(options, "--mount", "type=tmpfs,destination=/var/lib/journal")
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
		"--name", containerName,
		"--user", containerUserUID,
		"--group", containerUserGID,
		"--home", homeToUse,
		"--init", fmt.Sprintf("\"%d\"", btoi(init)),
		"--nvidia", fmt.Sprintf("\"%d\"", btoi(nvidia)),
		"--pre-init-hooks", containerPreInitHook,
		"--additional-packages", strings.Join(containerAdditionalPackages, " "),
		"--", fmt.Sprintf("'%s'", containerInitHook),
	}

	// Final assembly of the command
	// docker create [options] image [args...]
	//nolint:mnd // 2 is fine here, it's "create" and image
	cmd := make([]string, 0, len(options)+len(args)+2)
	cmd = append(cmd, "create")
	cmd = append(cmd, options...)
	cmd = append(cmd, containerImage)
	cmd = append(cmd, args...)

	return cmd
}

func (d *Docker) run(ctx context.Context, args []string, dryRun bool) (string, error) {
	command := "docker"

	// Empty elements are considered as positional argument by exec.Command
	cleanArgs := stripEmpty(args)
	if d.root {
		command = d.sudoCommand
		cleanArgs = append([]string{"docker"}, cleanArgs...)
	}

	if dryRun {
		fullCmd := fmt.Sprintf("%s %s", command, strings.Join(cleanArgs, " "))
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(fullCmd)
		return "", nil
	}

	cmd := exec.CommandContext(ctx, command, cleanArgs...)
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

func parseContainerList(output string) ([]containermanager.Container, error) {
	var containers []containermanager.Container

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}

		var dc dockerContainer
		if err := json.Unmarshal([]byte(line), &dc); err != nil {
			return nil, fmt.Errorf("failed to parse container JSON: %w", err)
		}

		const containerIDMaxLength = 12

		id := dc.ID
		if len(id) > containerIDMaxLength {
			id = id[:containerIDMaxLength]
		}

		containers = append(containers, containermanager.Container{
			ID:     id,
			Image:  dc.Image,
			Name:   dc.Names,
			Status: dc.Status,
			Labels: parseLabels(dc.Labels),
		})
	}

	return containers, nil
}

func parseLabels(labels string) map[string]string {
	result := make(map[string]string)
	if labels == "" {
		return result
	}

	for label := range strings.SplitSeq(labels, ",") {
		key, value, found := strings.Cut(label, "=")
		if found {
			result[key] = value
		}
	}
	return result
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func stripEmpty(a []string) []string {
	newArr := make([]string, 0, len(a))
	for _, str := range a {
		if str != "" {
			newArr = append(newArr, str)
		}
	}
	return newArr
}
