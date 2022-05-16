<!-- markdownlint-disable MD010 -->
# Stop containers

distrobox-stop stop a running distrobox.

Distroboxes are left running, even after exiting out of them, so that
subsequent enters are really quick. This is how they can be stopped.

Usage:

	distrobox-stop --name container-name
	distrobox-stop container-name

Options:

	--name/-n:		name for the distrobox
	--yes/-Y:		non-interactive, stop without asking
	--help/-h:		show this message
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox"
	--verbose/-v:		show more verbosity
	--version/-V:		show version
