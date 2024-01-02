<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox create
	distrobox-create

# DESCRIPTION

distrobox-create takes care of creating the container with input name and image.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external usb devices and
graphical apps (X11/Wayland), and audio.

# SYNOPSIS

**distrobox create**

	--image/-i:		image to use for the container	default: ${container_image_default}
	--name/-n:		name for the distrobox		default: ${container_name_default}
	--pull/-p:		pull the image even if it exists locally (implies --yes)
	--yes/-Y:		non-interactive, pull images without asking
	--root/-r:		launch podman/docker/lilipod with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--clone/-c:		name of the distrobox container to use as base for a new container
				this will be useful to either rename an existing distrobox or have multiple copies
				of the same environment.
	--home/-H:		select a custom HOME directory for the container. Useful to avoid host's home littering with temp files.
	--volume:		additional volumes to add to the container
	--additional-flags/-a:	additional flags to pass to the container manager command
	--additional-packages/-ap:	additional packages to install during initial container setup
	--init-hooks:		additional commands to execute during container initialization
	--pre-init-hooks:	additional commands to execute prior to container initialization
	--init/-I:		use init system (like systemd) inside the container.
				this will make host's processes not visible from within the container. (assumes --unshare-process)
	--nvidia:		try to integrate host's nVidia drivers in the guest
	--unshare-devsys:          do not share host devices and sysfs dirs from host
	--unshare-ipc:          do not share ipc namespace with host
	--unshare-netns:        do not share the net namespace with host
	--unshare-process:          do not share process namespace with host
	--unshare-all:          activate all the unshare flags below
	--compatibility/-C:	show list of compatible images
	--help/-h:		show this message
	--no-entry:		do not generate a container entry in the application list
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version

	--absolutely-disable-root-password-i-am-really-positively-sure: ⚠️ ⚠️  when setting up a rootful distrobox, this will skip user password setup, leaving it blank. ⚠️ ⚠️

# COMPATIBILITY

	for a list of compatible images and container managers, please consult the man page:
		man distrobox
		man distrobox-compatibility
	or consult the documentation page on: https://github.com/89luca89/distrobox/blob/main/docs/compatibility.md#containers-distros

# EXAMPLES

Create a distrobox with image alpine, called my-alpine container

	distrobox create --image alpine my-alpine-container

Create a distrobox from fedora-toolbox:35 image

	distrobox create --image registry.fedoraproject.org/fedora-toolbox:35 --name fedora-toolbox-35

Clone an existing distrobox container

	distrobox create --clone fedora-35 --name fedora-35-copy

Always pull for the new image when creating a distrobox

	distrobox create --pull --image centos:stream9 --home ~/distrobox/centos9

Add additional environment variables to the container

	distrobox create --image fedora:35 --name test --additional-flags "--env MY_VAR=value"

Add additional volumes to the container

	distrobox create --image fedora:35 --name test --volume /opt/my-dir:/usr/local/my-dir:rw --additional-flags "--pids-limit -1"

Add additional packages to the container

	distrobox create --image alpine:latest --name test2 --additional-packages "git tmux vim"

Use init-hooks to perform an action during container startup

	distrobox create --image alpine:latest --name test --init-hooks "touch /var/tmp/test1 && touch /var/tmp/test2"

Use pre-init-hooks to perform an action at the beginning of the container startup (before any package manager starts)

	distrobox create -i docker.io/almalinux/8-init --init --name test --pre-init-hooks "dnf config-manager --enable powertools && dnf -y install epel-release"

Use init to create a Systemd container (acts similar to an LXC):

	distrobox create -i ubuntu:latest --name test --additional-packages "systemd libpam-systemd" --init

Use init to create a OpenRC container (acts similar to an LXC):

	distrobox create -i alpine:latest --name test --additional-packages "openrc" --init

Use host's NVidia drivers integration

	distrobox create --image ubuntu:22.04 --name ubuntu-nvidia --nvidia

Do not use host's IP inside the container:

	distrobox create --image ubuntu:latest --name test --unshare-netns

Create a more isolated container, where only the $HOME, basic sockets and host's FS (in /run/host) is shared:

	distrobox create --name unshared-test --unshare-all

Create a more isolated container, with it's own init system, this will act very similar to a full LXC container:

	distrobox create --name unshared-init-test --unshare-all --init --image fedora:latest

Use environment variables to specify container name, image and container manager:

	DBX_CONTAINER_MANAGER="docker" DBX_NON_INTERACTIVE=1 DBX_CONTAINER_NAME=test-alpine DBX_CONTAINER_IMAGE=alpine distrobox-create

# ENVIRONMENT VARIABLES

	DBX_CONTAINER_ALWAYS_PULL
	DBX_CONTAINER_CUSTOM_HOME
	DBX_CONTAINER_HOME_PREFIX
	DBX_CONTAINER_IMAGE
	DBX_CONTAINER_MANAGER
	DBX_CONTAINER_NAME
	DBX_NON_INTERACTIVE
	DBX_SUDO_PROGRAM

DBX_CONTAINER_HOME_PREFIX defines where containers' home directories will be located.
If you define it as ~/dbx then all future containers' home directories will be ~/dbx/$container_name

# EXTRA

The `--additional-flags` or `-a` is useful to modify defaults in the container creations.
For example:

	distrobox create -i docker.io/library/archlinux -n dev-arch

	podman container inspect dev-arch | jq '.[0].HostConfig.PidsLimit'
	2048

	distrobox rm -f dev-arch
	distrobox create -i docker.io/library/archlinux -n dev-arch --volume $CBL_TC:/tc --additional-flags "--pids-limit -1"

	podman container inspect dev-arch | jq '.[0].HostConfig,.PidsLimit'
	0

Additional volumes can be specified using the `--volume` flag. This flag follows the
same standard as `docker` and `podman` to specify the mount point so `--volume SOURCE_PATH:DEST_PATH:MODE`.

	distrobox create --image docker.io/library/archlinux --name dev-arch --volume /usr/share/:/var/test:ro

During container creation, it is possible to specify (using the additional-flags) some
environment variables that will persist in the container and be independent from your environment:

	distrobox create --image fedora:35 --name test --additional-flags "--env MY_VAR=value"

The `--init-hooks` is useful to add commands to the entrypoint (init) of the container.
This could be useful to create containers with a set of programs already installed, add users, groups.

	distrobox create  --image fedora:35 --name test --init-hooks "dnf groupinstall -y \"C Development Tools and Libraries\""

The `--init` is useful to create a container that will use its own separate init system within.
For example using:

	distrobox create -i docker.io/almalinux/8-init --init --name test
	distrobox create -i docker.io/library/debian --additional-packages "systemd" --init --name test-debian

Inside the container we will be able to use normal systemd units:

	~$ distrobox enter test
	user@test:~$ sudo systemctl enable --now sshd
	user@test:~$ sudo systemctl status sshd
		● sshd.service - OpenSSH server daemon
		   Loaded: loaded (/usr/lib/systemd/system/sshd.service; enabled; vendor preset: enabled)
		   Active: active (running) since Fri 2022-01-28 22:54:50 CET; 17s ago
			 Docs: man:sshd(8)
				   man:sshd_config(5)
		 Main PID: 291 (sshd)

Note that enabling `--init` **will disable host's process integration**.
From within the container you will not be able to see and manage host's processes.
This is needed because `/sbin/init` must be pid 1.

If you want to use a non-pre-create image, you'll need to add the additional package:

	distrobox create -i alpine:latest --init --additional-packages "openrc" -n test
	distrobox create -i debian:stable --init --additional-packages "systemd libpam-systemd" -n test
	distrobox create -i ubuntu:22.04 --init --additional-packages "systemd libpam-systemd" -n test
	distrobox create -i archlinux:latest --init --additional-packages "systemd" -n test
	distrobox create -i registry.opensuse.org/opensuse/tumbleweed:latest --init --additional-packages "systemd" -n test
	distrobox create -i registry.fedoraproject.org/fedora:39 --init --additional-packages "systemd" -n test

The `--init` flag is useful to create system containers, where the container acts
more similar to a full VM than an application-container.
Inside you'll have a separate init, user-session, daemons and so on.

The `--home` flag let's you specify a custom HOME for the container.
Note that this will NOT prevent the mount of the host's home directory,
but will ensure that configs and dotfiles will not litter it.

The `--root` flag will let you create a container with real root privileges. At
first `enter` the user will be required to setup a password. This is done in order
to not enable passwordless sudo/su, in a **rootful** container, this is needed
because **in this mode, root inside the container is also root outside the container!**

The `--absolutely-disable-root-password-i-am-really-positively-sure`  will skip user password setup,
leaving it blank.
**This is genuinely dangerous and you really, positively should NOT enable this**.

From version 1.4.0 of distrobox, when you create a new container, it will also generate
an entry in the applications list.

## NVidia integration

If your host has an NVidia gpu, with installed proprietary drivers, you can integrate
them with the guests by using the `--nvidia` flag:

`distrobox create --nvidia --image ubuntu:latest --name ubuntu-nvidia`

Be aware that **this is not compatible with non-glibc systems** and **needs somewhat newer
distributions to work**.

This feature was tested working on:

- Almalinux
- Archlinux
- Centos 7 and newer
- Clearlinux
- Debian 10 and newer
- OpenSUSE Leap
- OpenSUSE Tumbleweed
- Rockylinux
- Ubuntu 18.04 and newer
- Void Linux (glibc)
