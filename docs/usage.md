- [Distrobox](README.md)
  * [Outside the distrobox](#outside-the-distrobox)
    + [Create the distrobox](#create-the-distrobox)
    + [Enter the distrobox](#enter-the-distrobox)
    + [List containers](#list-containers)
    + [Remove containers](#remove-containers)
  * [Inside the distrobox](#inside-the-distrobox)
    + [Application and service exporting](#application-and-service-exporting)
    + [Init the distrobox](#init-the-distrobox)

---

# Usage

As stated above, there are 4 tools at disposal, 2 have to be used **outside the distrobox (from the host)** and 2 have to be used **inside the distrobox (from the container)**.

---

## Outside the distrobox

### Create the distrobox

distrobox-create takes care of creating the container with input name and image.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external usb devices and
graphical apps (X11/Wayland), and audio.

Usage:

	distrobox-create --image registry.fedoraproject.org/fedora-toolbox:35 --name fedora-toolbox-35
	distrobox-create --clone fedora-toolbox-35 --name fedora-toolbox-35-copy
	distrobox-create --image alpine my-alpine-container

You can also use environment variables to specify container name and image

	DBX_NON_INTERACTIVE=1 DBX_CONTAINER_NAME=test-alpine DBX_CONTAINER_IMAGE=alpine distrobox-create

Options:

	--image/-i:		image to use for the container	default: registry.fedoraproject.org/fedora-toolbox:35
	--name/-n:		name for the distrobox		default: fedora-toolbox-35
	--non-interactive/-N:	non-interactive, pull images without asking
	--clone/-c:		name of the distrobox container to use as base for a new container
				this will be useful to either rename an existing distrobox or have multiple copies
				of the same environment.
	--home/-H		select a custom HOME directory for the container. Useful to avoid host's home littering with temp files.
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

### Enter the distrobox

distrobox-enter takes care of entering the container with the name specified.
Default command executed is your SHELL, but you can specify different shells or
entire commands to execute.
If using it inside a script, an application, or a service, you can specify the
--headless mode to disable tty and interactivity.

Usage:

	distrobox-enter --name fedora-toolbox-35 -- bash -l
	distrobox-enter my-alpine-container -- sh -l

Options:

	--name/-n:		name for the distrobox						default: fedora-toolbox-35
	--/-e:			end arguments execute the rest as command to execute at login	default: bash -l
	--headless/-H:		do not instantiate a tty
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

This is used to enter the distrobox itself. Personally, I just create multiple profiles in my `gnome-terminal` to have multiple distros accessible.

### List containers

distrobox-list lists available distroboxes. It detects them and lists them separately
from the rest of normal podman or docker containers.

Usage:

	distrobox-list

Options:

	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

![image](https://user-images.githubusercontent.com/598882/147831082-24b5bc2e-b47e-49ac-9b1a-a209478c9705.png)

### Remove containers

distrobox-rm delete one of the available distroboxes.

Usage:

	distrobox-rm --name container-name [--force]
	distrobox-rm container-name [-f]

Options:

	--name/-n:		name for the distrobox
	--force/-f:		force deletion
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

---

## Inside the distrobox

### Application and service exporting

distrobox-export takes care of exporting an app a binary or a service from the container
to the host.

The exported app will be easily available in your normal launcher and it will
automatically be launched from the container it is exported from.

The exported services will be available in the host's user's systemd session, so

	systemctl --user status exported_service_name

will show the status of the service exported.

The exported binaries will be exported in the "--export-path" of choice as a wrapper
script that acts naturally both on the host and in the container.
Note that "--export-path" is NOT OPTIONAL, you have to explicitly set it.

You can specify additional flags to add to the command, for example if you want
to export an electron app, you could add the "--foreground" flag to the command:

	distrobox-export --app atom --extra-flags "--foreground"
	distrobox-export --bin /usr/bin/vim --export-path ~/.local/bin --extra-flags "-p"
	distrobox-export --service syncthing --extra-flags "-allow-newer-config"

This works for services, binaries, and apps.
Extra flags are only used then the exported app, binary, or service is used from
the host, using them inside the container will not include them.

The option "--delete" will un-export an app, binary, or service.

	distrobox-export --app atom --delete
	distrobox-export --bin /usr/bin/vim --export-path ~/.local/bin --delete
	distrobox-export --service syncthing --delete
	distrobox-export --service nginx --delete

The option "--sudo" will launch the exported item as root inside the distrobox.

Note you can use --app OR --bin OR --service but not together.

	distrobox-export --service nginx --sudo

Usage:

	distrobox-export --app mpv [--extra-flags "flags"] [--delete] [--sudo]
	distrobox-export --service syncthing [--extra-flags "flags"] [--delete] [--sudo]
	distrobox-export --bin /path/to/bin --export-path ~/.local/bin [--extra-flags "flags"] [--delete] [--sudo]


Options:

	--app/-a:		name of the application to export
	--bin/-b:		absolute path of the binary to export
	--service/-s:		name of the service to export
	--delete/-d:		delete exported application or service
	--export-label/-el:	label to add to exported application name.
				Defaults to (on \$container_name)
	--export-path/-ep:	path where to export the binary
	--extra-flags/-ef:	extra flags to add to the command
	--sudo/-S:		specify if the exported item should be ran as sudo
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

You may want to install graphical applications or user services in your distrobox.
Using `distrobox-export` from **inside** the container will let you use them from the host itself.

App export example:

	distrobox-export --app abiword

This tool will simply copy the original `.desktop` files along with needed icons,
add the prefix `/usr/local/bin/distrobox-enter -n distrobox_name -e ... ` to the commands to run, and
save them in your home to be used directly from the host as a normal app.

Service export example:

	distrobox-export --service syncthing --extra-flags "--allow-newer-config"
	distrobox-export --service nginx --sudo

For services, it will similarly export the systemd unit inside the container to a `systemctl --user` service,
prefixing the various `ExecStart ExecStartPre ExecStartPost ExecReload ExecStop ExecStopPost` with the `distrobox-enter` command prefix.

Binary export example:


	distrobox-export --bin /usr/bin/code --extra-flags "--foreground" --export-path $HOME/.local/bin

In the case of exporting binaries, you will have to specify **where** to export it (`--export-path`) and the tool will create
a little wrapper script that will `distrobox-enter -e` from the host, the desired binary.
This can be handy with the use of `direnv` to have different versions of the same binary based on
your `env` or project.

![app-export](https://user-images.githubusercontent.com/598882/144294795-c7785620-bf68-4d1b-b251-1e1f0a32a08d.png)

![service-export](https://user-images.githubusercontent.com/598882/144294314-29a8921f-4511-453d-bf8e-d0d1e336db91.png)


NOTE: some electron apps such as vscode and atom need additional flags to work from inside the
container, use the `--extra-flags` option to provide a series of flags, for example:

`distrobox-export --app atom --extra-flags "--foreground"`

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
