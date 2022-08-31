<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox-init

# DESCRIPTION

**Init the distrobox (not to be launched manually)**

distrobox-init is the entrypoint of a created distrobox.
Note that this HAS to run from inside a distrobox, will not work if you run it
from your host.

**This is not intended to be used manually, but instead used by distrobox-create
to set up the container's entrypoint.**

distrobox-init will take care of installing missing dependencies (eg. sudo), set
up the user and groups, mount directories from the host to ensure the tight
integration.

# SYNOPSIS

**distrobox-init**

	--name/-n:		user name
	--user/-u:		uid of the user
	--group/-g:		gid of the user
	--home/-d:		path/to/home of the user
	--help/-h:		show this message
	--init/-I:		whether to use or not init
	--pre-init-hooks:	commands to execute prior to init
	--upgrade/-U:		run init in upgrade mode
	--verbose/-v:		show more verbosity
	--version/-V:		show version
	--:			end arguments execute the rest as command to execute during init

# EXAMPLES

	distrobox-init --name test-user --user 1000 --group 1000 --home /home/test-user
	distrobox-init --upgrade
