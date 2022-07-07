<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox stop
	distrobox-stop

# DESCRIPTION

distrobox-stop stop a running distrobox.

Distroboxes are left running, even after exiting out of them, so that
subsequent enters are really quick. This is how they can be stopped.

# SYNOPSIS

**distrobox stop**

	--name/-n:		name for the distrobox
	--yes/-Y:		non-interactive, stop without asking
	--help/-h:		show this message
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

	distrobox-stop --name container-name
	distrobox-stop container-name

You can also use environment variables to specify container manager and name:

	DBX_CONTAINER_MANAGER="docker" DBX_CONTAINER_NAME=test-alpine distrobox-stop

Supported environment variables:

	DBX_CONTAINER_MANAGER
	DBX_CONTAINER_NAME
	DBX_NON_INTERACTIVE
	DBX_SUDO_PROGRAM
