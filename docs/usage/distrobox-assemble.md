<!-- markdownlint-disable MD010 MD036 -->
# NAME

	distrobox assemble
	distrobox-assemble

# DESCRIPTION

distrobox-assemble takes care of creating or destroying containers in batches,
based on a manifest file.
The manifest file by default is `./distrobox.ini`, but can be specified using the
`--file` flag.

# SYNOPSIS

**distrobox assemble**

	--file/-f:		path to the distrobox manifest/ini file
	--replace/-R:		replace already existing distroboxes with matching names
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

This is an example manifest file to create two containers:

	[ubuntu]
	additional_packages=git vim tmux nodejs
	image=ubuntu:latest
	init=false
	nvidia=false
	pull=true
	root=false
	replace=true

	# You can add comments using this #
	[arch] # also inline comments are supported
	additional_packages=git vim tmux nodejs
	home=/tmp/home
	image=archlinux:latest
	init=false
	init_hooks="touch /init-normal"
	nvidia=true
	pre_init_hooks="touch /pre-init"
	pull=true
	root=false
	replace=false
	volume=/tmp/test:/run/a /tmp/test:/run/b

**Create**

We can bring them up simply using

	distrobox assemble create

If the file is called `distrobox.ini` and is in the same directory you're launching
the command, no further arguments are needed.
You can specify a custom path for the file using

	distrobox assemble create -f /my/custom/path.ini

**Replace**

By default, `distrobox assemble` will replace a container only if `replace=true`
is specified in the manifest file.

In the example of the manifest above, the ubuntu container will always be replaced
when running `distrobox assemble create`, while the arch container will not.

To force a replace for all containers in a manifest use the `--replace` flag

	distrobox assemble create --replace [-f my/custom/path.ini]

**Remove**

We can bring down all the containers in a manifest file by simply doing

	distrobox assemble rm

Or using a custom path for the ini file

	distrobox assemble rm --file my/custom/path.ini

**Test**

You can always test what distrobox **would do** by using the `--dry-run` flag.
This command will only print what commands distrobox would do without actually
running them.
