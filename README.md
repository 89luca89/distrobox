![distrobox-logo](https://user-images.githubusercontent.com/598882/144294113-ab3c62b0-4ff0-488f-8e85-dfecc308e561.png)

# Distrobox

Use any Linux distribution inside your terminal.
Distrobox uses `podman` or `docker` to create containers using the Linux distribution of your choice.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external USB devices and
graphical apps (X11/Wayland), and audio.

---

![overview](https://user-images.githubusercontent.com/598882/144294862-f6684334-ccf4-4e5e-85f8-1d66210a0fff.png)

---

- [Distrobox](#distrobox)
  * [What it does](#what-it-does)
  * [Why?](#why-)
    + [Aims](#aims)
- [Compatibility](#compatibility)
    + [Supported container managers](#supported-container-managers)
    + [Host Distros](#host-distros)
      - [New Host Distro support](#new-host-distro-support)
    + [Containers Distros](#containers-distros)
      - [New Distro support](#new-distro-support)
- [Usage](#usage)
  * [Outside the distrobox](#outside-the-distrobox)
    + [Create the distrobox](#create-the-distrobox)
    + [Enter the distrobox](#enter-the-distrobox)
  * [Inside the distrobox](#inside-the-distrobox)
    + [Application and service exporting](#application-and-service-exporting)
      - [Init the distrobox](#init-the-distrobox)
- [Installation](#installation)
- [Useful tips](#useful-tips)
  * [Improve distrobox-enter performance](#improve-distrobox-enter-performance)
  * [Container save and restore](#container-save-and-restore)
  * [Check used resources](#check-used-resources)
  * [Using podman inside a distrobox](#using-podman-inside-a-distrobox)
  * [Using docker inside a distrobox](#using-docker-inside-a-distrobox)
  * [Using distrobox as main cli](#using-distrobox-as-main-cli)
- [Authors](#authors)
- [License](#license)

## What it does

Simply put it's a fancy wrapper around `podman` or `docker` to create and start containers highly integrated with the hosts.

The distrobox environment is based on an OCI image.
This image is used to create a container that seamlessly integrates with the rest of the operating system by providing access to the user's home directory,
the Wayland and X11 sockets, networking, removable devices (like USB sticks), systemd journal, SSH agent, D-Bus,
ulimits, /dev and the udev database, etc...

It implements the same concepts introduced by https://github.com/containers/toolbox but in a simplified way using POSIX sh and aiming at broader compatibility.

All the props go to them as they had the great idea to implement this stuff.

It is divided into 4 parts:

- `distrobox-create` - creates the container
- `distrobox-enter`  - to enter the container
- `distrobox-init`   - it's the entrypoint of the container (not meant to be used manually)
- `distrobox-export` - it is meant to be used inside the container, useful to export apps and services from the container to the host

## Why?

- Provide a mutable environment on an immutable OS, like Endless OS, Fedora Silverblue, OpenSUSE MicroOS or SteamOS3
- Provide a locally privileged environment for sudoless setups (eg. company-provided laptops, security reasons, etc...)
- To mix and match a stable base system (eg. Debian Stable, Ubuntu LTS, RedHat) with a bleeding-edge environment for development or gaming (eg. Arch or OpenSUSE Tumbleweed or Fedora with latest Mesa)
- Leverage high abundance of curated distro images for docker/podman to manage multiple environments

### Aims

This project aims to bring **any distro userland to any other distro** supporting podman or docker.
It has been written in POSIX sh to be as portable as possible and not have problems with glibc version's compatibility.

It also aims to enter the container **as fast as possible**, every millisecond adds up if you use the container as your default environment for your terminal:

These are some sample results of `distrobox-enter` on the same container on my weak laptop from 2015 with 2 core cpu:

```
Total time for 100 container enters:

  luca-linux@x250:~$ time (for i in {1..100}; do distrobox-enter --name fedora-toolbox-35 -- whoami; done)
  real	0m36.209s
  user	0m6.520s
  sys	0m4.803s

Mean:

36.209s/100 = ~0.362ms mean time to enter the container
```

I would like to keep it always below the [Doherty Treshold](https://lawsofux.com/doherty-threshold/) of 400ms.

# Compatibility

This project **does not need a dedicated image**. It can use any OCI images from docker-hub, quay.io, or any registry of your choice.

Granted, they may not be as featureful as expected (some of them do not even have `which`, `mount`, `less` or `vi`)
but that's all doable in the container itself after bootstrapping it.

The main concern is having basic Linux utilities (`mount`), basic user management utilities (`usermod, passwd`), and `sudo` correctly set.

### Supported container managers

`distrobox` can run on either `podman` or `docker`

It depends either on `podman` configured in `rootless mode`
or on `docker` configured without sudo (you're in the `docker` group)

- Minimum podman version: **2.1.0**
- Minimum docker version: **18.06.1**

Follow the official installation guide here:

  - https://podman.io/getting-started/installation
  - https://docs.docker.com/engine/install

### Host Distros

Distrobox has been successfully tested on:

|    Distro  |    Version    | Notes |
| --- | --- | --- |
| Alpine Linux | 3.14.3 | To setup rootless podman, look [HERE](https://wiki.alpinelinux.org/wiki/Podman) |
| Arch Linux | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| Manjaro | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| Centos | 8</br>8 Stream | Works with corresponding RedHat releases. |
| Debian | 11</br>Testing</br>Unstable | |
| Fedora | 34</br>35 | |
| Fedora Silverblue | 34</br>35 | |
| Gentoo | | To setup rootless podman, look [HERE](https://wiki.gentoo.org/wiki/Podman) |
| Ubuntu | 20.04</br>21.10 | Older versions based on 20.04 needs external repos to install newer Podman and Docker releases. </br> Derivatives like Pop_OS!, Mint and Elementary OS should work the same. |
| EndlessOS | 4.0.0 | |
| OpenSUSE | Leap 15</br>Tumbleweed | |
| OpenSUSE MicroOS | 20211209 | |
| NixOS | 21.11 | To setup Docker, look [HERE](https://nixos.wiki/wiki/Docker) </br>To setup Podman, look [HERE](https://nixos.wiki/wiki/Podman) and [HERE](https://gist.github.com/adisbladis/187204cb772800489ee3dac4acdd9947) |

#### New Host Distro support

If your distro of choice is not on the list, open an issue requesting support for it,
we can work together to check if it is possible to add support for it.

Or just try using it anyway, if it works, open an issue
and it will be added to the list!

---

### Containers Distros

Distrobox guests tested successfully with the following container images:

|    Distro  |    Version | Images    |
| --- | --- | --- |
| AlmaLinux | 8     | docker.io/library/almalinux:8    |
| Alpine Linux    | 3.14</br>3.15 | docker.io/library/alpine:latest    |
| AmazonLinux | 2  | docker.io/library/amazonlinux:2.0.20211005.0    |
| Archlinux     | | docker.io/library/archlinux:latest    |
| Centos | 7</br>8 | quay.io/centos/centos:7</br>quay.io/centos/centos:8  |
| Debian | 8</br>9</br>10</br>11 | docker.io/library/debian:8</br>docker.io/library/debian:9</br>docker.io/library/debian:10</br>docker.io/library/debian:stable</br>docker.io/library/debian:stable-backports    |
| Debian | Testing    | docker.io/library/debian:testing </br> docker.io/library/debian:testing-backports    |
| Debian | Unstable | docker.io/library/debian:unstable    |
| Neurodebian | nd100 | docker.io/library/neurodebian:nd100 |
| Fedora | 34</br>35 | registry.fedoraproject.org/fedora-toolbox:34</br> docker.io/library/fedora:34</br>registry.fedoraproject.org/fedora-toolbox:35</br>docker.io/library/fedora:35    |
| Mageia | 8 | docker.io/library/mageia |
| Opensuse | Leap | registry.opensuse.org/opensuse/leap:latest    |
| Opensuse | Tumbleweed | registry.opensuse.org/opensuse/tumbleweed:latest </br> registry.opensuse.org/opensuse/toolbox:latest    |
| Oracle Linux | 7</br>8 | container-registry.oracle.com/os/oraclelinux:7</br>container-registry.oracle.com/os/oraclelinux:8    |
| Rocky Linux | 8 | docker.io/rockylinux/rockylinux:8    |
| Scientific Linux | 7 | docker.io/library/sl:7    |
| Slackware | 14.2 | docker.io/vbatts/slackware:14.2    |
| Slackware | current | docker.io/vbatts/slackware:current    |
| Ubuntu | 14.04</br>16.04</br>18.04</br>20.04</br>21.10</br>22.04 | docker.io/library/ubuntu:14.04</br>docker.io/library/ubuntu:16.04</br>docker.io/library/ubuntu:18.04</br>docker.io/library/ubuntu:20.04</br>docker.io/library/ubuntu:21.10</br>docker.io/library/ubuntu:22.04    |
| Kali Linux | rolling | docker.io/kalilinux/kali-rolling:latest |
| Void Linux | | ghcr.io/void-linux/void-linux:latest-thin-bb-x86_64 </br> ghcr.io/void-linux/void-linux:latest-thin-bb-x86_64-musl </br> ghcr.io/void-linux/void-linux:latest-full-x86_64 </br> ghcr.io/void-linux/void-linux:latest-full-x86_64-musl |


Note however that if you use a non-toolbox preconfigured image (e.g. images pre-baked to work with https://github.com/containers/toolbox), the **first** `distrobox-enter` you'll perform
can take a while as it will download and install the missing dependencies.

A small time tax to pay for the ability to use any type of image.
This will **not** occur after the first time, **subsequent enters will be much faster.**

#### New Distro support

If your distro of choice is not on the list, open an issue requesting support for it,
we can work together to check if it is possible to add support for it.

Or just try using it anyway, if it works, open an issue
and it will be added to the list!

# Usage

As stated above, there are 4 tools at disposal, 2 have to be used **outside the distrobox (from the host)** and 2 have to be used **inside the distrobox (from the container)**.

## Outside the distrobox

### Create the distrobox

distrobox-create takes care of creating the container with input name and image.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external usb devices and
graphical apps (X11/Wayland), and audio.

Usage:

	distrobox-create --image registry.fedoraproject.org/fedora-toolbox:35 --name fedora-toolbox-35
	distrobox-create --clone fedora-toolbox-35 --name fedora-toolbox-35-copy

Options:

	--image/-i:		image to use for the container	default: registry.fedoraproject.org/fedora-toolbox:35
	--name/-n:		name for the distrobox		default: fedora-toolbox-35
	--clone/-c:		name of the distrobox container to use as base for a new container
				this will be useful to either rename an existing distrobox or have multiple copies
				of the same environment.
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

Options:

	--name/-n:		name for the distrobox						default: fedora-toolbox-35
	--/-e:			end arguments execute the rest as command to execute at login	default: bash -l
	--headless/-H:		do not instantiate a tty
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

This is used to enter the distrobox itself. Personally, I just create multiple profiles in my `gnome-terminal` to have multiple distros accessible.

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

#### Init the distrobox

distrobox-init is the entrypoint of a created distrobox.
Note that this HAS to run from inside a distrobox, will not work if you run it
from your host.

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

**You should not have to launch this manually**

# Installation

If you like to live your life dangerously, you can trust me and simply run this in your terminal:

`curl https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh`

or if you want to select a custom directory to install without sudo:

`curl https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- -p ~/.local/bin`

Else you can clone the project using `git clone` or using the `download zip` voice after clicking the green button above.

Enter the directory and run `./install`, by default it will attempt to install in `/usr/local/bin`, you can specify another directory if needed with `./install -p ~/.local/bin`

# Useful tips

## Improve distrobox-enter performance

If you are experiencing a bit slow performance using `podman` you should enable
the podman socket using

`systemctl --user enable --now podman.socket`

this will improve a lot `podman`'s command performances.

## Container save and restore

To save, export and reuse an already configured container, you can leverage `podman save` or `docker save` and `podman import` or `docker import`
to create snapshots of your environment.

---

To save a container to an image:

with podman:

```
podman container commit -p distrobox_name image_name_you_choose
podman save image_name_you_choose:latest | gzip > image_name_you_choose.tar.gz
```

with docker:

```
docker container commit -p distrobox_name image_name_you_choose
docker save image_name_you_choose:latest | gzip > image_name_you_choose.tar.gz
```

This will create a tar.gz of the container of your choice at that exact moment.

---

Now you can backup that archive or transfer it to another host, and to restore it
just run

```
podman import image_name_you_choose.tar.gz
```

or

```
docker import image_name_you_choose.tar.gz
```

And create a new container based on that image:

```
distrobox-create --image image_name_you_choose:latest --name distrobox_name
distrobox-enter --name distrobox_name
```

And you're good to go, now you can reproduce your personal environment everywhere
in simple (and scriptable) steps.

## Check used resources

- You can always check how much space a `distrobox` is taking by using `podman` command:

`podman system df -v` or `docker system df -v`

- You can check running `distrobox` using:

`podman ps -a` or `docker ps -a`

- You can remove a `distrobox` using

`podman rm distrobox_name` or `docker rm distrobox_name`

## Using podman inside a distrobox

If `distrobox` is using `podman` as the container engine, you can use `podman socket` to
control host's podman from inside a `distrobox`, just use:

`podman --remote`

inside the `distrobox` to use it.

It may be necessary to enable the socket on your host system by using:

`systemctl --user enable --now podman.socket`

## Using docker inside a distrobox

You can use `docker` to control host's podman from inside a `distrobox`,
by default if `distrobox` is using docker as a container engine, it will mount the
docker.sock into the container.

So in the container just install `docker`, add yourself to the `docker` group, and
you should be good to go.

## Using distrobox as main cli

In case you want (like me) to use your container as the main CLI environment, it comes
handy to use `gnome-terminal` profiles to create a dedicated setup for it:

![Screenshot from 2021-12-19 22-29-08](https://user-images.githubusercontent.com/598882/146691460-b8a5bb0a-a83d-4e32-abd0-4a0ff9f50eb7.png)

Personally, I just bind `Ctrl-Alt-T` to the Distrobox profile and `Super+Enter` to the Host profile.

For other terminals, there are similar features (profiles) or  you can set up a dedicated shortcut to
launch a terminal directly in the distrobox

# Authors

- Luca Di Maio      <luca.dimaio1@gmail.com>

# License

- GNU GPLv3, See LICENSE file.
