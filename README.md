# simpler-toolbox

A simplified version of Fedora Toolbox written in Posix Sh

![Screenshot from 2021-11-18 17-47-14](https://user-images.githubusercontent.com/598882/142459870-6447300f-3bdd-4518-ad2b-e13d29552ace.png)
![Screenshot from 2021-11-18 18-57-11](https://user-images.githubusercontent.com/598882/142470693-eabf33a4-6309-425a-bb2f-eb43770f1618.png)


## What it does

It implements what https://github.com/containers/toolbox does but in a simplified and less-featured way.

All the props goes to them as they had the great idea to implement this stuff.

# Aims

This project aims to bring `toolbox` to any distro supporting podman.
It has been written in posix sh to be as portable as possible and not have problems
with glibc compatibility or versions.

It also is a bit faster to enter the container, which adds up if you use the it
as your default environment for your terminal:

These are some simple results of `toolbox enter` on the same container on my weak laptop:

```
luca-linux@x250:~$ time bin/toolbox_enter -n fedora-toolbox-35 -- whoami
luca-linux

real	0m0.494s
user	0m0.135s
sys	0m0.070s
luca-linux@x250:~$ time toolbox run -c fedora-toolbox-35 whoami
luca-linux

real	0m1.165s
user	0m0.298s
sys	0m0.200s
luca-linux@x250:~$
```

It also includes a `toolbox_export` functionality to export applications and services from
the container onto the host.

# Compatibility

Differently from the original project, this one does **not need** a dedicated image
but can use normal images in example from docker hub.

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

tested with the following container images:

- Alpine Linux	(docker.io/library/alpine:latest)
- Archlinux		(docker.io/library/archlinux:latest)
- Centos 7		(quay.io/centos/centos:7)
- Centos 8		(quay.io/centos/centos:8)
- Debian 11		(docker.io/library/debian:latest)
- Fedora 34		(registry.fedoraproject.org/fedora-toolbox:34, docker.io/library/fedora:34)
- Fedora 35		(registry.fedoraproject.org/fedora-toolbox:35, docker.io/library/fedora:35)
- Opensuse Leap	(registry.opensuse.org/opensuse/leap:latest)
- Opensuse Tumbleweed	(registry.opensuse.org/opensuse/thumbleweed:latest, registry.opensuse.org/opensuse/toolbox:latest)
- Ubuntu 20.04	(docker.io/library/ubuntu:20.04)
- Ubuntu 21.10	(docker.io/library/ubuntu:21.10)

Note however that if you use a non-toolbox pre configured image, the **first** `toolbox_enter` (or to be more precise the `podman start`) you perform
will take a while as it will install with the pkg manager the missing dependencies.
A small time-tax to pay for the ability to use any type of image.
This will **not** occur after the first time, and will enter directly.

# Usage

### Create the toolbox

	toolbox_create --image registry.fedoraproject.org/fedora-toolbox:35 --name fedora-toolbox-35

	Arguments:
		--image/-i: image to use for the container	default: registry.fedoraproject.org/fedora-toolbox:35
		--name/-n:  name for the toolbox			default: fedora-toolbox-35
		--help/-h:	show this message
		-v:			show more verbosity

If the image is not present you'll be prompted to `podman pull` it.

### Init the toolbox


	toolbox_init --name test-user --user 1000 --group 1000 --home /home/test-user

	Arguments:
		--name/-n:		user name
		--user/-u:		uid of the user
		--group/-g:		gid of the user
		--home/-d:		path/to/home of the user
		--help/-h:		show this message
		-v:			show more verbosity

This is used as entrypoint for the created container, it will take care of creating the users,
setting up sudo, mountpoints and exports.

### Enter the toolbox

	toolbox_enter --name fedora-toolbox-35 -- bash -l

	Arguments:
		--name/-n:		name for the toolbox			default: fedora-toolbox-35
		--:			end arguments execute the rest as command to execute at login		default: bash -l
		--help/-h:		show this message
		-v:			show more verbosity

This is used to enter the toolbox itself, personally I just create multiple profiles in my `gnome-terminal` to have multiple distros accessible.

# Application and service exporting

You may want to install graphical applications or user services in your toolbox.
Using `toolbox_eport` from **inside** the container, will let you use them from the host itself.

Examples:

`toolbox_export --app mpv`
`toolbox_export --service syncthing`

This tool will simply copy the original `.desktop` files (with needed icons) or `.service` files,
add the prefix `/usr/local/bin/toolbox_enter -n fedora-toolbox -e ... ` to the commands to run, and
save them in your home to be used directly from the host as a normal app or `systemctl --user` service.

# Installation

place the three files somewhere in your $PATH.

# Dependencies

It depends on `podman` configured in `rootless mode`

Check out your distro's documentation to check how to.

## Authors

- Luca Di Maio      <luca.dimaio1@gmail.com>

## License

- GNU GPLv3, See LICENSE file.
