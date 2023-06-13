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
	--delete/-d:		delete exported application or binary
	--export-label/-el:	label to add to exported application name.
				Defaults to (on \$container_name)
	--export-path/-ep:	path where to export the binary
	--extra-flags/-ef:	extra flags to add to the command
	--login/-l		run the exported item in a login shell
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

**Run inside a login shell**

The option "--login" will launch the exported item inside a login shell, allowing access to
variables sourced at login in the container (e.g: /etc/profile).

**Exporting apps from rootful containers**

It is worth noting that, when exporting any item - which includes graphical apps - from rootful
containers (created with `distrobox create --root`), root privileges will be needed every time
the item is launched (in order to enter the rootful container), which, by default, is done
using `sudo` (see docs for `distrobox-enter` on how to customize that). However, for
graphical apps in specific, since they launch without a terminal, the usage of `sudo`
might, at first, make it impossible to launch them.

To fix this without needing to customize the sudo program, one can define a global
`SUDO_ASKPASS` environment variable on their machine, which is a PATH to an executable
that is run by `sudo` when no terminal is available (or when it is given the `--askpass`
or `-A` option), and the output of that executable to stdout is used as the password input.
The executable is called as many times is needed for authentication as root to succeed
(unless a limit of amount of attempts is reached).

To do this, pick a program to ask the user for graphical password input. In this example,
we will use `zenity --password`, which should be present for GNOME users (and can
also be installed in other DEs) - there are other options, such as
`kdialog --password "Message"` for KDE users.

Write the call to the desired program to a script file, for example to
`/usr/bin/my-password-prompt` (sample contents below):

	#!/bin/sh
	zenity --password "Authentication as root is required"

(You may save the script under, for example, `~/.local/bin` if you want to keep it
fully local to your user.)

Afterwards, make it executable (e.g. run `sudo chmod +x /usr/bin/my-password-prompt`). Then,
make sure to set `SUDO_ASKPASS` to `"/usr/bin/my-password-prompt"` (replace with your script's path)
in a global profile file, so that it is picked up by sudo when running graphical apps (and, therefore,
sudo will run the script you created to ask for a password).
This is done with the shell line `export SUDO_ASKPASS="/path/to/script/goes/here"`.
You can do this for your user only by running the command below (replace the script path as needed):

	echo 'export SUDO_ASKPASS="/usr/bin/my-password-prompt"' >> ~/.profile

Which appends the appropriate line to the end of your `~/.profile` file, thus making the change
local to your user. Alternatively, to set it system-wide (for all users), you may create a file
in `/etc/profile.d/` (or equivalent for your system) with that line.

Now just log out and log back in, and graphical apps exported from rootful containers should
now be properly asking for root's password before launching (instead of not opening, if that
was the case before).

**Notes**

Note you can use --app OR --bin but not together.

![app-export](https://user-images.githubusercontent.com/598882/144294795-c7785620-bf68-4d1b-b251-1e1f0a32a08d.png)

NOTE: some electron apps such as vscode and atom need additional flags to work from inside the
container, use the `--extra-flags` option to provide a series of flags, for example:

`distrobox-export --app atom --extra-flags "--foreground"`
