<!-- markdownlint-disable MD010 -->
# Remove containers

distrobox-rm delete one of the available distroboxes.

Usage:

	distrobox-rm --name container-name [--force]
	distrobox-rm container-name [-f]

Options:

	--name/-n:		name for the distrobox
	--force/-f:		force deletion
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox"
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version
