<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox list
	distrobox-list

# DESCRIPTION

distrobox-list lists available distroboxes. It detects them and lists them separately
from the rest of normal podman or docker containers.

# SYNOPSIS

**distrobox list**

	--help/-h:		show this message
	--no-color:		disable color formatting
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox" (note: if using a program other than 'sudo' for root privileges is necessary,
				specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)
	--size/-s:		show also container size
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

	distrobox-list

You can also use environment variables to specify container manager

	DBX_CONTAINER_MANAGER="docker" distrobox-list

Supported environment variables:

	DBX_CONTAINER_MANAGER
	DBX_SUDO_PROGRAM

![image](https://user-images.githubusercontent.com/598882/147831082-24b5bc2e-b47e-49ac-9b1a-a209478c9705.png)
