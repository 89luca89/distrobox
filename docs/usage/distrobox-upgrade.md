<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox-upgrade

# DESCRIPTION

distrobox-upgrade will enter the specified list of containers and will perform
an upgrade using the container's package manager.

# SYNOPSIS

**distrobox upgrade**

	--help/-h:		show this message
	--all/-a:		perform for all distroboxes
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

Upgrade all distroboxes

	distrobox-upgrade --all

Upgrade a specific distrobox

	distrobox-upgrade alpine-linux 

Upgrade a list of distroboxes

	distrobox-upgrade alpine-linux ubuntu22 my-distrobox123
