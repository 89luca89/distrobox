![distrobox-logo](https://user-images.githubusercontent.com/598882/144294113-ab3c62b0-4ff0-488f-8e85-dfecc308e561.png)

# Distrobox

![Lint](https://github.com/89luca89/distrobox/actions/workflows/main.yml/badge.svg)
[![GitHub](https://img.shields.io/github/license/89luca89/distrobox?color=blue)](../COPYING.md)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/89luca89/distrobox)](https://github.com/89luca89/distrobox/releases/latest)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/89luca89/distrobox)
[![Github issue needs help](https://img.shields.io/github/issues-raw/89luca89/distrobox/help%20wanted?color=blue&label=Help%20Wanted%20Issues)](https://github.com/89luca89/distrobox/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
[![GitHub issues by-label](https://img.shields.io/github/issues-raw/89luca89/distrobox/bug?color=red&label=Open%20Bug%20Reports)](https://github.com/89luca89/distrobox/issues?q=is%3Aissue+is%3Aopen+label%3A%22bug%22)


Use any Linux distribution inside your terminal. Enable both backward and forward compatibility with software and freedom to use whatever distribution you’re more comfortable with.
Distrobox uses `podman` or `docker` to create containers using the Linux distribution of your choice.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external USB devices and
graphical apps (X11/Wayland), and audio.

---

![overview](https://user-images.githubusercontent.com/598882/144294862-f6684334-ccf4-4e5e-85f8-1d66210a0fff.png)

---

- [Distrobox](#distrobox)
  * [What it does](#what-it-does)
    + [See it in action](#see-it-in-action)
  * [Why?](#why-)
    + [Aims](#aims)
- [Installation](#installation)
    + [Uninstallation](#uninstallation)
- [Compatibility](compatibility.md)
    + [Supported container managers](compatibility.md#supported-container-managers)
    + [Host Distros](compatibility.md#host-distros)
    + [Containers Distros](compatibility.md#containers-distros)
- [Usage](usage/usage.md)
  * [Outside the distrobox](#outside-the-distrobox)
    + [distrobox-create](usage/distrobox-create.md)
    + [distrobox-enter](usage/distrobox-enter.md)
    + [distrobox-list](usage/distrobox-list.md)
    + [distrobox-rm](usage/distrobox-rm.md)
  * [Inside the distrobox](#inside-the-distrobox)
    + [distrobox-export](usage/distrobox-export.md)
    + [distrobox-init](usage/distrobox-init.md)
- [Useful tips](useful_tips.md)
- [Posts](posts/posts.md)
    + [Run latest GNOME and KDE using distrobox](posts/run_latest_gnome_kde_on_distrobox.md)
    + [Integrate VSCode and Distrobox](posts/integrate_vscode_distrobox.md)
    + [Execute a command on the Host](posts/execute_commands_on_host.md)
- [Featured Articles](featured_articles.md)
    + [Run Distrobox on Fedora Linux - Fedora Magazine](https://fedoramagazine.org/run-distrobox-on-fedora-linux/)
    + [DistroBox – Run Any Linux Distribution Inside Linux Terminal - TecMint](https://www.tecmint.com/distrobox-run-any-linux-distribution/)
    + [Using Distrobox To Augment The Package Selection On Clear Linux - Phoronix](https://www.phoronix.com/scan.php?page=news_item&px=Distrobox-Clear-Linux)
    + [Benchmark: benefits of Clear Linux containers (distrobox) - Phoronix](https://www.phoronix.com/forums/forum/phoronix/latest-phoronix-articles/1305326-clear-linux-container-performance-continues-showing-sizable-gains)
    + [Distrobox - A great item in the Linux toolbelt - phmurphy's blog](https://phmurphy.com/posts/distrobox-toolbelt/)
    + [Running Other Linux Distros with Distrobox on Fedora Linux - bandithijo's blog](featured_articles.md)
    + [Day-to-day differences between Fedora Silverblue and Ubuntu - castrojo's blog](https://www.ypsidanger.com/day-to-day-advantages-of-fedora-silverblue/)
    + [Podcasts](featured_articles.md#podcasts)

---

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

### See it in action

Thanks to [castrojo](https://github.com/castrojo), you can see Distrobox in action in this explanatory video on his setup with Distrobox, Toolbx, Fedora Silverblue on his project [ublue](https://github.com/castrojo/ublue) (check it out!)

[![Video](https://user-images.githubusercontent.com/598882/153680522-f5903607-2854-4cfb-a186-cba7403745bd.png)](https://www.youtube.com/watch?v=Q2PrISAOtbY)

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

  ~$ time (for i in {1..100}; do distrobox-enter --name fedora-toolbox-35 -- whoami; done)
  real	0m36.209s
  user	0m6.520s
  sys	0m4.803s

Mean:

36.209s/100 = ~0.362ms mean time to enter the container
```

I would like to keep it always below the [Doherty Treshold](https://lawsofux.com/doherty-threshold/) of 400ms.

---

# Installation

If you like to live your life dangerously, you can trust me and simply run this in your terminal:

`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh`

or if you want to select a custom directory to install without sudo:

`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- -p ~/.local/bin`

Else you can clone the project using `git clone` or using the latest release [HERE](https://github.com/89luca89/distrobox/releases/latest).

Enter the directory and run `./install`, by default it will attempt to install in `~/.local/bin` but if you run the script as root, it will default to `/usr/local/bin`. You can specify a custom directory with the `-p` flag such as `./install -p ~/.bin`. 

Or check the [Host Distros](compatibility.md#host-distros) compatibility list for distro-specific instructions.

## Dependencies

Distrobox depends on a container manager to work, you can choose to install either podman or docker.
Please look in the [Compatibility Table](compatibility.md#host-distros) for your distribution notes.

---

## Uninstallation

If you installed distrobox using the `install` script in the default install directory use this:

`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/uninstall | sudo sh`

or if you specified a custom path:

`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/uninstall | sh -s -- -p ~/.local/bin`

Else if cloned the project using `git clone` or using the latest archive release from [HERE](https://github.com/89luca89/distrobox/releases/latest),

enter the directory and run `./uninstall`, by default it will assume the install directory was `/usr/local/bin`, you can specify another directory if needed with `./uninstall -p ~/.local/bin`

---

You can take a look at some usage examples [HERE](usage/usage.md) <br>
with a list of useful tips [HERE](useful_tips.md)

---
