<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox generate-entry

# DESCRIPTION

distrobox-generate-entry will create a desktop icon for one of the available distroboxes.
This will be then deleted when you remove the matching distrobox.

# SYNOPSIS

**distrobox generate-entry**

	--help/-h:		show this message
	--all/-a:		perform for all distroboxes
	--delete/-d:		delete the entry
	--icon/-i:		specify a custom icon [/path/to/icon] (default auto)
	--root/-r:		perform on rootful distroboxes
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

Generate an entry for a container

	distrobox generate-entry my-container-name

Specify a custom icon for the entry

	distrobox generate-entry my-container-name --icon /path/to/icon.png

Generate an entry for all distroboxes

	distrobox generate-entry --all

Delete an entry

	distrobox generate-entry container-name --delete
