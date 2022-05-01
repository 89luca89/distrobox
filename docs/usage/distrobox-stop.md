<!-- markdownlint-disable MD010 -->
# Stop containers

distrobox-rm delete one of the available distroboxes.

Usage:

	distrobox-rm --name container-name
	distrobox-rm container-name

Options:

	--name/-n:		name for the distrobox
	--yes/-Y:		non-interactive, stop without asking
	--help/-h:		show this message
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox"
	--verbose/-v:		show more verbosity
	--version/-V:		show version
