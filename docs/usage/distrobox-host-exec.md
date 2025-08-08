<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox-host-exec

# DESCRIPTION

distrobox-host-exec lets one execute command on the host, while inside of a container.

Under the hood, distrobox-host-exec uses `host-spawn` a project that lets us
execute commands back on the host.
If the tool is not found the user will be prompted to install it.

# SYNOPSIS

Just pass to "distrobox-host-exec" any command and all its arguments, if any.

	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version
	--yes/-Y:		Automatically answer yes to prompt:
                                host-spawn will be installed on the guest system
                                if host-spawn is not detected.
                                This behaviour is default when running in a non-interactive shell.
	--current-work-dir=PATH, -cwd PATH:
				Set working directory on the host before executing the command.
				Useful when the container's current directory does not exist on the host.

If no command is provided, it will execute "$SHELL".

Note about working directory mismatches:
If the current working directory inside the container does not exist on the host
(for example you are in ~/test in the container while ~/test is missing on the host),
invoking distrobox-host-exec from that directory may fail on the host.
In such cases the tool will print a warning but still attempt to execute using host-spawn.
Consider specifying --current-work-dir to an existing path on the host.

Alternatively, use symlinks to make `distrobox-host-exec` execute as that command:

	~$: ln -s /usr/bin/distrobox-host-exec /usr/local/bin/podman
	~$: ls -l /usr/local/bin/podman
	lrwxrwxrwx. 1 root root 51 Jul 11 19:26 /usr/local/bin/podman -> /usr/bin/distrobox-host-exec
	~$: podman version
	...this is executed on host...

# EXAMPLES

	distrobox-host-exec ls
	distrobox-host-exec bash -l
	distrobox-host-exec flatpak run org.mozilla.firefox
	distrobox-host-exec podman ps -a
