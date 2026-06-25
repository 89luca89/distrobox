<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox ephemeral

# DESCRIPTION

`distrobox ephemeral` creates a temporary distrobox that is automatically destroyed
when the command is terminated.

It accepts the same flags as `distrobox create` (except `--no-entry` and
`--compatibility`, which are not relevant for a throwaway container), plus
`-e`/`--exec` to mark the start of the command to run inside the container
(the bare `--` separator works too).

# SYNOPSIS

**distrobox ephemeral**

	--/-e/--exec:		end arguments execute the rest as command to execute at login	default: default ${USER}'s shell
	--root/-r:		launch podman/docker/lilipod with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--verbose/-v:		show more verbosity
	--help/-h:		show this message
	--version/-V:		show version

All other flags are inherited from [distrobox-create](distrobox-create.md).

# EXAMPLES

	distrobox ephemeral --image alpine:latest -- cat /etc/os-release
	distrobox ephemeral --root --verbose --image alpine:latest --volume /opt:/opt

# SEE ALSO

	distrobox create --help
	man distrobox-create

# ENVIRONMENT VARIABLES

	distrobox ephemeral reuses distrobox create's options; see
	distrobox-create(1) for the list of supported environment variables.
