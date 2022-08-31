<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox generate-entry

# DESCRIPTION

distrobox-generate-entry will create a desktop icon for one of the available distroboxes.
This will be then deleted when you remove the matching distrobox.

# SYNOPSIS

**distrobox generate-entry**

		--help/-h:			  show this message
		--all/-a:			   perform for all distroboxes
		--delete/-d:			delete the entry
		--icon/-i:			  specify a custom icon [/path/to/icon] (default auto)
		--verbose/-v:		   show more verbosity
		--version/-V:		   show version

# EXAMPLES

	distrobox-generate-entry container-name [--delete] [--icon [auto,/path/to/icon]]
