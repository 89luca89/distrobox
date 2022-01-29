### Create the distrobox

distrobox-create takes care of creating the container with input name and image.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external usb devices and
graphical apps (X11/Wayland), and audio.

Usage:

	distrobox-create --image registry.fedoraproject.org/fedora-toolbox:35 --name fedora-toolbox-35
	distrobox-create --clone fedora-toolbox-35 --name fedora-toolbox-35-copy
	distrobox-create --image alpine my-alpine-container
	distrobox create --image fedora:35 --name test --volume /opt/my-dir:/usr/local/my-dir:rw --additional-flags "--pids-limit -1"
	distrobox create --image fedora:35 --name test --additional-flags "--env MY_VAR-value"
	distrobox create --image alpine:latest --name test --init-hooks "touch /var/tmp/test1 && touch /var/tmp/test2"
	distrobox create -i docker.io/almalinux/8-init --init --name test

You can also use environment variables to specify container name and image

	DBX_NON_INTERACTIVE=1 DBX_CONTAINER_NAME=test-alpine DBX_CONTAINER_IMAGE=alpine distrobox-create

Supported environment variables:

	DBX_NON_INTERACTIVE
	DBX_CONTAINER_NAME
	DBX_CONTAINER_IMAGE

Options:

	--image/-i:		image to use for the container	default: registry.fedoraproject.org/fedora-toolbox:35
	--name/-n:		name for the distrobox		default: fedora-toolbox-35
	--yes/-Y:	non-interactive, pull images without asking
	--clone/-c:		name of the distrobox container to use as base for a new container
				this will be useful to either rename an existing distrobox or have multiple copies
				of the same environment.
	--home/-H		select a custom HOME directory for the container. Useful to avoid host's home littering with temp files.
	--volume		additional volumes to add to the container
	--additional-flags/-a:	additional flags to pass to the container manager command
	--init-hooks		additional commands to execute during container initialization
	--init/-I		use init system (like systemd) inside the container.
				this will make host's processes not visible from within the container.
	--help/-h:		show this message
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version

The `--additional-flags` or `-a` is useful to modify defaults in the container creations.
For example:

	distrobox create -i docker.io/library/archlinux -n dev-arch

	podman container inspect dev-arch | jq '.[0].HostConfig.PidsLimit'
	2048

	distrobox rm -f dev-arch
	distrobox create -i docker.io/library/archlinux -n dev-arch --volume $CBL_TC:/tc --additional-flags "--pids-limit -1"

	podman container inspect dev-arch | jq '.[0].HostConfig,.PidsLimit'
	0

Additional volumes can be specified using the `--volume` flag. This flag follows the same standard as `docker` and `podman`
to specify the mount point so `--volume SOURCE_PATH:DEST_PATH:MODE`.

	distrobox create --image docker.io/library/archlinux --name dev-arch --volume /usr/share/:/var/test:ro

During container creation, it is possible to specify (using the additional-flags) some environment variables that will
persist in the container and be independent from your environment:

	distrobox create --image fedora:35 --name test --additional-flags "--env MY_VAR-value"

The `--init-hooks` is useful to add commands to the entrypoint (init) of the container. This could be useful
to create containers with a set of programs already installed, add users, groups.

	distrobox create  --image fedora:35 --name test --init-hooks "dnf groupinstall -y \"C Development Tools and Libraries\""

The `--init` is useful to create a container that will use its own separate init system within. For example using:

	distrobox create -i docker.io/almalinux/8-init --init-hooks "dnf install -y openssh-server" --init --name test

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

Note that enabling `--init` **will disable host's process integration**. From within the container you will not be able
to see and manage host's processes. This is needed because `/sbin/init` must be pid 1.
