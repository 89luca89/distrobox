<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox-host-exec

# DESCRIPTION

distrobox-host-exec lets one execute command on the host, while inside of a container.

If "flatpak-spawn" is installed in the container, this is what is used, and it is the
most powerful and recommended method. If, instead, "flatpak-spawn" can't be found, it
still try to get the job done with "chroot" (but beware that not all commands/programs
will work well in this mode).

# SYNOPSIS

Just pass to "distrobox-host-exec" any command and all its arguments, if any.

	distrobox-host-exec [command [arguments]]

	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

If no command is provided, it will execute "/bin/sh".

# EXAMPLES

	distrobox-host-exec ls
	distrobox-host-exec bash -l
	distrobox-host-exec flatpak run org.mozilla.firefox
	distrobox-host-exec podman ps -a
