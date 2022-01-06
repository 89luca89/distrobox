# distrobox-init

## DESCRIPTION

distrobox-init is the entrypoint of a created distrobox. Note that this
HAS to run from inside a distrobox, will not work if you run it from
your host.

distrobox-init will take care of installing missing dependencies (eg.
sudo), set up the user and groups, mount directories from the host to
ensure the tight integration.

Usage: distrobox-init **\--name** mb **\--user** 1000 **\--group** 1000
**\--home** */home/mb*

## OPTIONS

**\--name**/-n:

:   user name

**\--user**/-u:

:   uid of the user

**\--group**/-g:

:   gid of the user

**\--home**/-d:

:   path/to/home of the user

**\--help**/-h:

:   show this message

**\--verbose**/-v:

:   show more verbosity

**\--version**/-V:

:   show version

## SEE ALSO

GitHub repository is located at https://github.com/89luca89/distrobox
