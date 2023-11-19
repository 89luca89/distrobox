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

	--all/-a:		stop all distroboxes
	--yes/-Y:		non-interactive, stop without asking
	--help/-h:		show this message
	--root/-r:		launch podman/docker/lilipod with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

	distrobox-stop container-name1 container-name2
	distrobox-stop container-name
	distrobox-stop --all

You can also use environment variables to specify container manager and name:

	DBX_CONTAINER_MANAGER="docker" DBX_CONTAINER_NAME=test-alpine distrobox-stop

# ENVIRONMENT VARIABLES

	DBX_CONTAINER_MANAGER
	DBX_CONTAINER_NAME
	DBX_NON_INTERACTIVE
	DBX_SUDO_PROGRAM
