<!-- markdownlint-disable MD010 MD036 -->

# NAME

	distrobox-export

# DESCRIPTION

**Application and binary exporting**

distrobox-export takes care of exporting an app or a binary from the container
to the host.

The exported app will be easily available in your normal launcher and it will
automatically be launched from the container it is exported from.

# SYNOPSIS

**distrobox-export**

	--app/-a:		name of the application to export
	--bin/-b:		absolute path of the binary to export
	--list-apps:		list applications exported from this container
	--list-binaries		list binaries exported from this container, use -ep to specify custom paths to search
	--delete/-d:		delete exported application or binary
	--export-label/-el:	label to add to exported application name.
				Use "none" to disable.
				Defaults to (on \$container_name)
	--export-path/-ep:	path where to export the binary
	--extra-flags/-ef:	extra flags to add to the command
	--enter-flags/-nf:	flags to add to distrobox-enter
	--sudo/-S:		specify if the exported item should be run as sudo
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

You may want to install graphical applications or CLI tools in your distrobox.
Using `distrobox-export` from **inside** the container will let you use them from the host itself.

# EXAMPLES

	distrobox-export --app mpv [--extra-flags "flags"] [--delete] [--sudo]
	distrobox-export --bin /path/to/bin [--export-path ~/.local/bin] [--extra-flags "flags"] [--delete] [--sudo]

**App export example**

	distrobox-export --app abiword

This tool will simply copy the original `.desktop` files along with needed icons,
add the prefix `/usr/local/bin/distrobox-enter -n distrobox_name -e ...` to the commands to run, and
save them in your home to be used directly from the host as a normal app.

**Binary export example**

	distrobox-export --bin /usr/bin/code --extra-flags "--foreground" --export-path $HOME/.local/bin

In the case of exporting binaries, you will have to specify **where** to export it
(`--export-path`) and the tool will create a little wrapper script that will
`distrobox-enter -e` from the host, the desired binary.
This can be handy with the use of `direnv` to have different versions of the same binary based on
your `env` or project.

The exported binaries will be exported in the "--export-path" of choice as a wrapper
script that acts naturally both on the host and in the container.

**Additional flags**

You can specify additional flags to add to the command, for example if you want
to export an electron app, you could add the "--foreground" flag to the command:

	distrobox-export --app atom --extra-flags "--foreground"
	distrobox-export --bin /usr/bin/vim --export-path ~/.local/bin --extra-flags "-p"

This works for binaries and apps.
Extra flags are only used then the exported app or binary is used from
the host, using them inside the container will not include them.

**Unexport**

The option "--delete" will un-export an app or binary

	distrobox-export --app atom --delete
	distrobox-export --bin /usr/bin/vim --export-path ~/.local/bin --delete

**Run as root in the container**

The option "--sudo" will launch the exported item as root inside the distrobox.

**Notes**

Note you can use --app OR --bin but not together.

![app-export](https://user-images.githubusercontent.com/598882/144294795-c7785620-bf68-4d1b-b251-1e1f0a32a08d.png)

NOTE: some electron apps such as vscode and atom need additional flags to work from inside the
container, use the `--extra-flags` option to provide a series of flags, for example:

`distrobox-export --app atom --extra-flags "--foreground"`
