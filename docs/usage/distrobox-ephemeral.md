<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox ephemeral
	distrobox-ephemeral

# DESCRIPTION

distrobox-ephemeral creates a temporary distrobox that is automatically destroyed
when the command is terminated.

# SYNOPSIS

**distrobox ephemeral**

	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--verbose/-v:		show more verbosity
	--help/-h:		show this message
	--/-e:			end arguments execute the rest as command to execute at login	default: bash -l
	--version/-V:		show version

# EXAMPLES

	distrobox-ephemeral --image alpine:latest -- cat /etc/os-release
	distrobox-ephemeral --root --verbose --image alpine:latest --volume /opt:/opt

You can also use [flags from **distrobox-create**](distrobox-create.md) to customize the ephemeral container to run.

Refer to

	man distrobox-create

or

	distrobox-create --help

Supported environment variables:

	distrobox-ephemeral calls distrobox-create, SEE ALSO distrobox-create(1) for
	a list of supported environment variables to use.
