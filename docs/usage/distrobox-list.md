<!-- markdownlint-disable MD010 -->
# List containers

distrobox-list lists available distroboxes. It detects them and lists them separately
from the rest of normal podman or docker containers.

Usage:

	distrobox-list

You can also use environment variables to specify container manager

	DBX_CONTAINER_MANAGER="docker" distrobox-list

Supported environment variables:

	DBX_CONTAINER_MANAGER

Options:

	--help/-h:		show this message
	--root/-r:		launch podman/docker with root privileges. Note that if you need root this is the preferred
				way over "sudo distrobox"
	--size/-s:		show also container size
	--verbose/-v:		show more verbosity
	--version/-V:		show version

![image](https://user-images.githubusercontent.com/598882/147831082-24b5bc2e-b47e-49ac-9b1a-a209478c9705.png)
