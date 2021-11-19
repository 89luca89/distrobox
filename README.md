# simpler-toolbox

A simplified version of Fedora Toolbox written in Posix Sh

![Screenshot from 2021-11-18 17-47-14](https://user-images.githubusercontent.com/598882/142459870-6447300f-3bdd-4518-ad2b-e13d29552ace.png)
![Screenshot from 2021-11-18 18-57-11](https://user-images.githubusercontent.com/598882/142470693-eabf33a4-6309-425a-bb2f-eb43770f1618.png)


## What it does

It implements what https://github.com/containers/toolbox does but in a simplified and less-featured way.

All the props goes to them as they had the great idea to implement this stuff.

## What it does not

It **doesn NOT want** to be a replacement for the full toolbox tool, it's not battle tested (yet)
And it probably doesn't cover some of the cornercases I've not encountered yet :-)

# Aims

This project aims to bring `toolbox` to any distro supporting podman.
It has been written in posix sh to be as portable as possible and not have problems
with glibc compatibility or versions.

It also is a bit faster to enter the toolbox, which adds up if you use the toolbox
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

# Compatibility

Differently from the original project, this one does **not need** a dedicated image
but can use normal images in example from docker hub.

Granted, they may not be as featureful as expected (some of them do not even have `which` )
but that's all doable in the toolbox itself after bootstrapping it.

Main concern is having basic user management utilities (`usermod, passwd`) and `sudo` correctly
set.

Host compatibility tested on:

- Fedora 34
- Fedora 35
- Ubuntu 20.04
- Ubuntu 21.10
- Debian 11
- Centos 8 Stream

Toolbox tested with the following containers:

- Alpine Linux	(docker.io/library/alpine:latest)
- Archlinux		(docker.io/library/archlinux:latest)
- Centos 7		(quay.io/centos/centos:7)
- Centos 8		(quay.io/centos/centos:8)
- Debian 11		(docker.io/library/debian:latest)
- Fedora 34		(registry.fedoraproject.org/fedora-toolbox:34, docker.io/library/fedora:34)
- Fedora 35		(registry.fedoraproject.org/fedora-toolbox:35, docker.io/library/fedora:35)
- Opensuse Leap	(registry.opensuse.org/opensuse/leap:latest)
- Opensuse Tumbleweed	(registry.opensuse.org/opensuse/thumbleweed:latest)
- Ubuntu 20.04	(docker.io/library/ubuntu:20.04)
- Ubuntu 21.10	(docker.io/library/ubuntu:21.10)

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

# Installation

place the three files somewhere in your $PATH.

# Dependencies

It depends on `podman` configured in `rootless mode`

Check out your distro's documentation to check how to.

## Authors

- Luca Di Maio      <luca.dimaio1@gmail.com>

## License

- GNU GPLv3, See LICENSE file.
