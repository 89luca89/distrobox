### Init the distrobox (not to be launched manually)

distrobox-init is the entrypoint of a created distrobox.
Note that this HAS to run from inside a distrobox, will not work if you run it
from your host.

This is not intended to be used manually, but instead used by distrobox-enter
to set up the container's entrypoint.

distrobox-init will take care of installing missing dependencies (eg. sudo), set
up the user and groups, mount directories from the host to ensure the tight
integration.

Usage:

	distrobox-init --name test-user --user 1000 --group 1000 --home /home/test-user

Options:

	--name/-n:		user name
	--user/-u:		uid of the user
	--group/-g:		gid of the user
	--home/-d:		path/to/home of the user
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

This is used as entrypoint for the created container, it will take care of creating the users,
setting up sudo, mountpoints, and exports.

**You should not have to launch this manually**, this is used by `distrobox create` to set up
container's entrypoint.
