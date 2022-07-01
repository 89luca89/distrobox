<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox rm
	distrobox-rm

# DESCRIPTION

distrobox-rm delete one of the available distroboxes.

# SYNOPSIS

**distrobox rm**

	--name/-n:		name for the distrobox
	--force/-f:		force deletion
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				refer to --sudo-program)
	--sudo-program		when used with --root, specifies a program other than the default 'sudo' with which to launch podman/docker
				with root privileges (common options include 'pkexec' for a graphical root authentication prompt, 'doas', and so on)
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

	distrobox-rm --name container-name [--force]
	distrobox-rm container-name [-f]

You can also use environment variables to specify container manager and name:

	DBX_CONTAINER_MANAGER="docker" DBX_CONTAINER_NAME=test-alpine distrobox-rm

Supported environment variables:

	DBX_CONTAINER_MANAGER
	DBX_CONTAINER_NAME
	DBX_NON_INTERACTIVE
	DBX_SUDO_PROGRAM
