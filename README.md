![distrobox-logo](https://user-images.githubusercontent.com/598882/144294113-ab3c62b0-4ff0-488f-8e85-dfecc308e561.png)

# Distrobox

Use any linux distribution inside your terminal.

![overview](https://user-images.githubusercontent.com/598882/144294862-f6684334-ccf4-4e5e-85f8-1d66210a0fff.png)


## What it does

It implements what https://github.com/containers/toolbox does but in a simplified way using POSIX sh and with broader compatibility.

All the props goes to them as they had the great idea to implement this stuff.

## But what is a distrobox?

Distrobox is a tool for Linux operating systems, which allows the use of containerized command line environments.
It is built on top of **podman** and other standard container technologies from OCI.

The intention is to provide a mutable environment on a host where the file-system is immutable (like Suse's MicroOS, Fedora Silverblue, Endless OS or SteamOS3)
or where the user doesn't have privileges to modify the host (non-sudo users for example)

So even if you're not a sudoer or your distro doesn't have access to a traditional package manager, you
will still be able to perform your `apt/dnf/pacman/pkg/zypper` shenanigans.

The distrobox environment is based on an OCI image.
This image is used to create a container that seamlessly integrates with the rest of the operating system by providing access to the user's home directory,
the Wayland and X11 sockets, networking, removable devices (like USB sticks), systemd journal, SSH agent, D-Bus,
ulimits, /dev and the udev database, etc..

# Aims

This project aims to bring any distro userland to any other distro supporting podman.
It has been written in posix sh to be as portable as possible and not have problems with glibc compatibility or versions.

It also aims to enter the container as fast as possible, every millisecond adds up if you use the it
as your default environment for your terminal:

These are some simple results of `distrobox-enter` on the same container on my weak laptop:

```
luca-linux@x250:~$ time distrobox-enter -n fedora-distrobox-35 -- whoami
luca-linux

real	0m0.494s
user	0m0.135s
sys	0m0.070s

luca-linux@x250:~$ time distrobox-enter -n fedora-distrobox-35 -- whoami
luca-linux

real	0m0,302s
user	0m0,118s
sys	0m0,095s

luca-linux@x250:~$ time distrobox-enter -n fedora-distrobox-35 -- whoami
luca-linux

real	0m0,281s
user	0m0,116s
sys	0m0,063s
```

It is included also a `distrobox-export` functionality to export applications and services from the container onto the host.

# Compatibility

This project does **not need** a dedicated image but can use normal images in example from docker hub.

Granted, they may not be as featureful as expected (some of them do not even have `which` )
but that's all doable in the container itself after bootstrapping it.

Main concern is having basic user management utilities (`usermod, passwd`) and `sudo` correctly
set.

Host compatibility tested on:

- Fedora 34
- Fedora 35
- Ubuntu 20.04
- Ubuntu 21.10
- Debian 11
- Centos 8 Stream

distrobox guests tested with the following container images:

|	Distro  |	Images	|
| --- | --- |
| AlmaLinux 8	 	| docker.io/library/almalinux:8	|
| Alpine Linux		| docker.io/library/alpine:latest	|
| AmazonLinux 2  	| docker.io/library/amazonlinux:2.0.20211005.0	|
| Archlinux		 	| docker.io/library/archlinux:latest	|
| Centos 7		 	| quay.io/centos/centos:7	|
| Centos 8		 	| quay.io/centos/centos:8	|
| Debian 11			| docker.io/library/debian:stable, docker.io/library/debian:stable-backports	|
| Debian Testing	| docker.io/library/debian:testing, docker.io/library/debian:testing-backports	|
| Debian Unstable	| docker.io/library/debian:unstable	|
| Fedora 34			| registry.fedoraproject.org/fedora-toolbox:34, docker.io/library/fedora:34	|
| Fedora 35			| registry.fedoraproject.org/fedora-toolbox:35, docker.io/library/fedora:35	|
| Opensuse Leap		| registry.opensuse.org/opensuse/leap:latest	|
| Opensuse Tumbleweed	| registry.opensuse.org/opensuse/tumbleweed:latest, registry.opensuse.org/opensuse/toolbox:latest	|
| Oracle Linux 7 	| container-registry.oracle.com/os/oraclelinux:7	|
| Oracle Linux 8 	| container-registry.oracle.com/os/oraclelinux:8	|
| Rocky Linux 8		| docker.io/rockylinux/rockylinux:8	|
| Scientific Linux 7| docker.io/library/sl:7	|
| Ubuntu 20.04		| docker.io/library/ubuntu:20.04	|
| Ubuntu 21.10		| docker.io/library/ubuntu:21.10	|


Note however that if you use a non-toolbox pre configured image (e.g. images pre-baked to work with https://github.com/containers/toolbox),
the **first** `distrobox-enter` (or to be more precise the `podman start`) you perform
will take a while as it will install with the pkg manager the missing dependencies.

A small time-tax to pay for the ability to use any type of image.
This will **not** occur after the first time, and will enter directly.

# Usage

### Create the distrobox

	distrobox-create --image registry.fedoraproject.org/fedora:35 --name fedora-35

	Arguments:
		--image/-i: image to use for the container	default: registry.fedoraproject.org/fedora-toolbox:35
		--name/-n:  name for the distrobox			default: fedora-toolbox-35
		--help/-h:	show this message
		-v:			show more verbosity

If the image is not present you'll be prompted to `podman pull` it.

### Init the distrobox


	distrobox-init --name test-user --user 1000 --group 1000 --home /home/test-user

	Arguments:
		--name/-n:		user name
		--user/-u:		uid of the user
		--group/-g:		gid of the user
		--home/-d:		path/to/home of the user
		--help/-h:		show this message
		-v:			show more verbosity

This is used as entrypoint for the created container, it will take care of creating the users,
setting up sudo, mountpoints and exports.

### Enter the distrobox

	distrobox-enter --name fedora-35 -- bash -l

	Arguments:
		--name/-n:		name for the distrobox			default: fedora-35
		--:			end arguments execute the rest as command to execute at login		default: bash -l
		--help/-h:		show this message
		-v:			show more verbosity

This is used to enter the distrobox itself, personally I just create multiple profiles in my `gnome-terminal` to have multiple distros accessible.

# Application and service exporting

	distrobox-export --app mpv
	distrobox-export --service syncthing

	Note you can use --app OR --service but not together.

	Arguments:
			--app/-a:               name of the application to export
			--service/-s:           name of the service to export
			--delete/-d:		delete exported application or service
			--help/-h:              show this message
			-v:                     show more verbosity

You may want to install graphical applications or user services in your distrobox.
Using `distrobox-eport` from **inside** the container, will let you use them from the host itself.

Examples:

`distrobox-export --app abiword`

`distrobox-export --service syncthing`

This tool will simply copy the original `.desktop` files (with needed icons) or `.service` files,
add the prefix `/usr/local/bin/distrobox-enter -n fedora-35 -e ... ` to the commands to run, and
save them in your home to be used directly from the host as a normal app or `systemctl --user` service.

![app-export](https://user-images.githubusercontent.com/598882/144294795-c7785620-bf68-4d1b-b251-1e1f0a32a08d.png)


![service-export](https://user-images.githubusercontent.com/598882/144294314-29a8921f-4511-453d-bf8e-d0d1e336db91.png)


# Installation

If you like to live your life dangerously, you can trust me and simply run this in your terminal:

`curl https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh`

or if you want to select a custom folder to install without sudo:

`curl https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- -p ~/.local/bin`

Else you can clone the project using `git clone` or using the `download zip` voice after clicking the green button above.

Enter the folder and run `./install`, by default it will attemp to install in `/usr/local/bin`, you can specify another folder if needed with `./install -p ~/.local/bin` 
 
# Dependencies

It depends on `podman` configured in `rootless mode`

Check out your distro's documentation to check how to.

## Authors

- Luca Di Maio      <luca.dimaio1@gmail.com>

## License

- GNU GPLv3, See LICENSE file.
