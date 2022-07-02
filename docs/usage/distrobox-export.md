<!-- markdownlint-disable MD010 MD036 -->

# NAME

	distrobox-export

# DESCRIPTION

**Application and service exporting**

distrobox-export takes care of exporting an app a binary or a service from the container
to the host.

The exported app will be easily available in your normal launcher and it will
automatically be launched from the container it is exported from.

# SYNOPSIS

**distrobox-export**

	--app/-a:		name of the application to export
	--bin/-b:		absolute path of the binary to export
	--service/-s:		name of the service to export
	--delete/-d:		delete exported application or service
	--export-label/-el:	label to add to exported application name.
				Defaults to (on \$container_name)
	--export-path/-ep:	path where to export the binary
	--extra-flags/-ef:	extra flags to add to the command
	--sudo/-S:		specify if the exported item should be run as sudo (refer to --sudo-program if sudo is
				not available in the container or not desired)
	--sudo-program:		when used with --sudo, specifies a program other than the default 'sudo' with which to launch the exported app
				with root privileges inside this container (common options include 'pkexec' for a graphical root authentication prompt,
				'doas', and so on)
	--host-sudo-program:		if this container is rootful, then this parameter specifies the program that should be used
				in the host to enter this container with root privileges when launching the exported app/service/binary,
				other than the default 'sudo' (such as 'pkexec' for a graphical authentication prompt)
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

You may want to install graphical applications or user services in your distrobox.
Using `distrobox-export` from **inside** the container will let you use them from the host itself.

# EXAMPLES

	distrobox-export --app mpv [--extra-flags "flags"] [--delete] [--sudo]
	distrobox-export --service syncthing [--extra-flags "flags"] [--delete] [--sudo]
	distrobox-export --bin /path/to/bin --export-path ~/.local/bin [--extra-flags "flags"] [--delete] [--sudo]

**App export example**

	distrobox-export --app abiword

This tool will simply copy the original `.desktop` files along with needed icons,
add the prefix `/usr/local/bin/distrobox-enter -n distrobox_name -e ...` to the commands to run, and
save them in your home to be used directly from the host as a normal app.

**Service export example**

	distrobox-export --service syncthing --extra-flags "--allow-newer-config"
	distrobox-export --service nginx --sudo

For services, it will similarly export the systemd unit inside the container to a
`systemctl --user` service, prefixing the various
`ExecStart ExecStartPre ExecStartPost ExecReload ExecStop ExecStopPost` with the
`distrobox-enter` command prefix.

The exported services will be available in the host's user's systemd session, so

	systemctl --user status exported_service_name

will show the status of the service exported.

**Binary export example**

	distrobox-export --bin /usr/bin/code --extra-flags "--foreground" --export-path $HOME/.local/bin

In the case of exporting binaries, you will have to specify **where** to export it
(`--export-path`) and the tool will create a little wrapper script that will
`distrobox-enter -e` from the host, the desired binary.
This can be handy with the use of `direnv` to have different versions of the same binary based on
your `env` or project.

The exported binaries will be exported in the "--export-path" of choice as a wrapper
script that acts naturally both on the host and in the container.
Note that "--export-path" is NOT OPTIONAL, you have to explicitly set it.

**Additional flags**

You can specify additional flags to add to the command, for example if you want
to export an electron app, you could add the "--foreground" flag to the command:

	distrobox-export --app atom --extra-flags "--foreground"
	distrobox-export --bin /usr/bin/vim --export-path ~/.local/bin --extra-flags "-p"
	distrobox-export --service syncthing --extra-flags "-allow-newer-config"

This works for services, binaries, and apps.
Extra flags are only used then the exported app, binary, or service is used from
the host, using them inside the container will not include them.

**Unexport**

The option "--delete" will un-export an app, binary, or service.

	distrobox-export --app atom --delete
	distrobox-export --bin /usr/bin/vim --export-path ~/.local/bin --delete
	distrobox-export --service syncthing --delete
	distrobox-export --service nginx --delete

**Run as root in the container**

The option "--sudo" will launch the exported item as root inside the distrobox, by prepending
its launch command inside the container with `sudo`. If, however, you'd like to use a different
command than `sudo` - such as `pkexec` (for a graphical authentication prompt) or `doas` -,
then specify `--sudo-program command_name` to ensure that `command_name` will be used
inside the container every time the exported item is launched. For example:

	distrobox-export --app simplescreenrecorder --sudo --sudo-program doas

This will ensure that `doas` will be used to launch `simplescreenrecorder` in the container
every time the exported app is opened through its desktop shortcut (as specified in the
`.desktop` file). Note that the specified command must already exist inside the container.

**Exporting items from rootful containers**

The usage of rootful containers requires using the `--root` option alongside distrobox commands.
This means that when an app, service or binary is exported from inside a rootful container,
distrobox will automatically use `--root` whenever the exported item is launched from the host.
However, this means that root privileges are required to run the exported item from the host,
which is normally handled using `sudo` in the host system. If, however, you'd prefer to use
a different command to invoke the rootful container with root privileges, such as
`pkexec` - which can be useful for exported apps, as it provides a graphical prompt
for authentication - or `doas`, make sure to use the `--host-sudo-program` option
to specify the desired sudo program.
For example, to always request root authentication with `pkexec` (instead of `sudo`) in the host
to open the kitty app exported from a rootful container, you may use the following command
while exporting the app:

	distrobox-export --app kitty --host-sudo-program pkexec

This will ensure a graphical authentication prompt for root permissions will be shown before
attempting to launch the rootful container's `kitty` app from the host, which can be useful
if your particular desktop environment doesn't work well with launching desktop shortcuts
that use `sudo`, for example. (This example assumes that `pkexec` is installed and properly
configured in the host system.)

**Notes**

Note you can use --app OR --bin OR --service but not together.

	distrobox-export --service nginx --sudo

![app-export](https://user-images.githubusercontent.com/598882/144294795-c7785620-bf68-4d1b-b251-1e1f0a32a08d.png)

![service-export](https://user-images.githubusercontent.com/598882/144294314-29a8921f-4511-453d-bf8e-d0d1e336db91.png)

NOTE: some electron apps such as vscode and atom need additional flags to work from inside the
container, use the `--extra-flags` option to provide a series of flags, for example:

`distrobox-export --app atom --extra-flags "--foreground"`
