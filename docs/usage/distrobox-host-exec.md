<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox-host-exec

# DESCRIPTION

distrobox-host-exec lets one execute command on the host, while inside of a container.

Under the hood, distrobox-host-exec uses `host-spawn` a project that let's us
execute commands back on the host.
If the tool is not found the user will be prompted to install it.

# SYNOPSIS

Just pass to "distrobox-host-exec" any command and all its arguments, if any.

	distrobox-host-exec [command [arguments]]

	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

If no command is provided, it will execute "$SHELL".

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
