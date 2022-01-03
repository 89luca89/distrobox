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
- [Installation](#installation)
- [Compatibility](compatibility.md)
- [Usage](usage.md)
- [Useful tips](useful_tips.md)
- [Featured Articles](featured_articles.md)

## What it does

Simply put it's a fancy wrapper around `podman` or `docker` to create and start containers highly integrated with the hosts.

The distrobox environment is based on an OCI image.
This image is used to create a container that seamlessly integrates with the rest of the operating system by providing access to the user's home directory,
the Wayland and X11 sockets, networking, removable devices (like USB sticks), systemd journal, SSH agent, D-Bus,
ulimits, /dev and the udev database, etc...

It implements the same concepts introduced by https://github.com/containers/toolbox but in a simplified way using POSIX sh and aiming at broader compatibility.

All the props go to them as they had the great idea to implement this stuff.

It is divided into 6 commands:

- `distrobox-create` - creates the container
- `distrobox-enter`  - to enter the container
- `distrobox-list` - to list containers created with distrobox
- `distrobox-rm` - to delete a container created with distrobox
- `distrobox-init`   - it's the entrypoint of the container (not meant to be used manually)
- `distrobox-export` - it is meant to be used inside the container, useful to export apps and services from the container to the host

It also includes a little wrapper to launch commands with `distrobox COMMAND` instead of calling the single files.

## Why?

- Provide a mutable environment on an immutable OS, like Endless OS, Fedora Silverblue, OpenSUSE MicroOS or SteamOS3
- Provide a locally privileged environment for sudoless setups (eg. company-provided laptops, security reasons, etc...)
- To mix and match a stable base system (eg. Debian Stable, Ubuntu LTS, RedHat) with a bleeding-edge environment for development or gaming (eg. Arch, OpenSUSE Tumbleweed or Fedora with latest Mesa)
- Leverage high abundance of curated distro images for docker/podman to manage multiple environments

Refer to the compatiblity list for an overview of supported host's distro [HERE](compatibility.md#host-distros) and container's distro [HERE](compatibility.md#containers-distros).

### Aims

This project aims to bring **any distro userland to any other distro** supporting podman or docker.
It has been written in POSIX sh to be as portable as possible and not have problems with dependencies and glibc version's compatibility.

Refer [HERE](compatibility.md#supported-container-managers) for a list of supported container managers and minimum supported versions.

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

# Installation

If you like to live your life dangerously, you can trust me and simply run this in your terminal:

`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh`

or if you want to select a custom directory to install without sudo:

`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- -p ~/.local/bin`

Else you can clone the project using `git clone` or using the latest release [HERE](https://github.com/89luca89/distrobox/releases/latest).

Enter the directory and run `./install`, by default it will attempt to install in `/usr/local/bin`, you can specify another directory if needed with `./install -p ~/.local/bin`

---

You can take a look at some usage examples [HERE](usage.md) <br>
with a list of useful tips [HERE](useful_tips.md).
