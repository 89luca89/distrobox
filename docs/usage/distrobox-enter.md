<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox enter
	distrobox-enter

# DESCRIPTION

distrobox-enter takes care of entering the container with the name specified.
Default command executed is your SHELL, but you can specify different shells or
entire commands to execute.
If using it inside a script, an application, or a service, you can specify the
--headless mode to disable tty and interactivity.

# SYNOPSIS

**distrobox enter**

	--name/-n:		name for the distrobox						default: my-distrobox
	--/-e:			end arguments execute the rest as command to execute at login	default: bash -l
	--no-tty/-T:		do not instantiate a tty
	--no-workdir/-nw:		always start the container from container's home directory
	--additional-flags/-a:	additional flags to pass to the container manager command
	--help/-h:		show this message
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

	distrobox-enter --name fedora-toolbox-35 -- bash -l
	distrobox-enter my-alpine-container -- sh -l
	distrobox-enter --additional-flags "--preserve-fds" --name test -- bash -l
	distrobox-enter --additional-flags "--env MY_VAR=value" --name test -- bash -l
	MY_VAR=value distrobox-enter --additional-flags "--preserve-fds" --name test -- bash -l

You can also use environment variables to specify container manager and container name:

	DBX_CONTAINER_MANAGER="docker" DBX_CONTAINER_NAME=test-alpine distrobox-enter

Supported environment variables:

	DBX_CONTAINER_NAME
	DBX_CONTAINER_MANAGER
	DBX_SKIP_WORKDIR
	DBX_SUDO_PROGRAM

This is used to enter the distrobox itself. Personally, I just create multiple profiles in
my `gnome-terminal` to have multiple distros accessible.

The `--additional-flags` or `-a` is useful to modify default command when executing in the container.
For example:

	distrobox enter -n dev-arch --additional-flags "--env my_var=test" -- printenv &| grep my_var
	my_var=test

This is possible also using normal env variables:

	my_var=test distrobox enter -n dev-arch --additional-flags -- printenv &| grep my_var
	my_var=test

If you'd like to enter a rootful container having distrobox use a program other than 'sudo' to
run podman/docker as root, such as 'pkexec' or 'doas', you may specify it with the
`DBX_SUDO_PROGRAM` environment variable. For example, to use 'doas' to enter a rootful container:

	DBX_SUDO_PROGRAM="doas" distrobox enter -n container --root

Additionally, in one of the config file paths that distrobox supports, such as `~/.distroboxrc`,
you can also append the line `distrobox_sudo_program="doas"` (for example) to always run
distrobox commands involving rootful containers using 'doas'.
