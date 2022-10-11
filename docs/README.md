![distrobox-logo](https://user-images.githubusercontent.com/598882/157771834-7423cf9b-8311-4e90-8a79-cd0eff6bd632.png)
<sub>logo credits [j4ckr3d](https://github.com/j4ckr3d)<sub>

# Distrobox

![Lint](https://github.com/89luca89/distrobox/actions/workflows/main.yml/badge.svg)
[![CI](https://github.com/89luca89/distrobox/actions/workflows/compatibility.yml/badge.svg)](https://github.com/89luca89/distrobox/actions/workflows/compatibility.yml)
[![GitHub](https://img.shields.io/github/license/89luca89/distrobox?color=blue)](../COPYING.md)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/89luca89/distrobox)](https://github.com/89luca89/distrobox/releases/latest)
[![Packaging status](https://repology.org/badge/tiny-repos/distrobox.svg)](https://repology.org/project/distrobox/versions)
[![Github issue needs help](https://img.shields.io/github/issues-raw/89luca89/distrobox/help%20wanted?color=blue&label=Help%20Wanted%20Issues)](https://github.com/89luca89/distrobox/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
[![GitHub issues by-label](https://img.shields.io/github/issues-raw/89luca89/distrobox/bug?color=red&label=Open%20Bug%20Reports)](https://github.com/89luca89/distrobox/issues?q=is%3Aissue+is%3Aopen+label%3A%22bug%22)

Use any Linux distribution inside your terminal. Enable both backward and forward
compatibility with software and freedom to use whatever distribution you’re more
comfortable with.
Distrobox uses `podman` or `docker` to create containers using the Linux distribution
of your choice.
The created container will be tightly integrated with the host, allowing sharing
of the HOME directory of the user, external storage, external USB devices and
graphical apps (X11/Wayland), and audio.

---

[Documentation](https://distrobox.privatedns.org/#distrobox) -
[Matrix Room](https://matrix.to/#/%23distrobox:matrix.org) -
[Telegram Group](https://t.me/distrobox)

---

![overview](https://user-images.githubusercontent.com/598882/144294862-f6684334-ccf4-4e5e-85f8-1d66210a0fff.png)

---

- [Distrobox](#distrobox)
  - [What it does](#what-it-does)
    - [See it in action](#see-it-in-action)
  - [Why?](#why)
    - [Aims](#aims)
- [Installation](#installation)
  - [Alternative methods](#alternative-methods)
    - [Curl](#curl)
    - [Git](#git)
    - [Flatpak](#flatpak)
  - [Dependencies](#dependencies)
    - [Install Podman without root](compatibility.md#install-podman-in-a-static-manner)
  - [Uninstallation](#uninstallation)
- [Compatibility](compatibility.md)
  - [Supported container managers](compatibility.md#supported-container-managers)
  - [Host Distros](compatibility.md#host-distros)
  - [Containers Distros](compatibility.md#containers-distros)
- [Usage](usage/usage.md)
  - [Outside the distrobox](usage/usage.md#outside-the-distrobox)
    - [distrobox-create](usage/distrobox-create.md)
    - [distrobox-enter](usage/distrobox-enter.md)
    - [distrobox-ephemeral](usage/distrobox-ephemeral.md)
    - [distrobox-generate-entry](usage/distrobox-generate-entry.md)
    - [distrobox-list](usage/distrobox-list.md)
    - [distrobox-rm](usage/distrobox-rm.md)
    - [distrobox-stop](usage/distrobox-stop.md)
    - [distrobox-upgrade](usage/distrobox-upgrade.md)
  - [Inside the distrobox](usage/usage.md#inside-the-distrobox)
    - [distrobox-export](usage/distrobox-export.md)
    - [distrobox-host-exec](usage/distrobox-host-exec.md)
    - [distrobox-init](usage/distrobox-init.md)
  - [Configure distrobox](#configure-distrobox)
- [Useful tips](useful_tips.md)
  - [Execute complex commands directly from distrobox-enter](useful_tips.md#execute-complex-commands-directly-from-distrobox-enter)
  - [Create a distrobox with a custom HOME directory](useful_tips.md#create-a-distrobox-with-a-custom-home-directory)
  - [Mount additional volumes in a distrobox](useful_tips.md#mount-additional-volumes-in-a-distrobox)
  - [Use a different shell than the host](useful_tips.md#use-a-different-shell-than-the-host)
  - [Duplicate an existing distrobox](useful_tips.md#duplicate-an-existing-distrobox)
  - [Export to the host](useful_tips.md#export-to-the-host)
  - [Execute commands on the host](useful_tips.md#execute-commands-on-the-host)
  - [Enable SSH X-Forwarding when SSH-ing in a distrobox](useful_tips.md#enable-ssh-x-forwarding-when-ssh-ing-in-a-distrobox)
  - [Use distrobox to install different flatpaks from the host](useful_tips.md#use-distrobox-to-install-different-flatpaks-from-the-host)
  - [Using podman inside a distrobox](useful_tips.md#using-podman-inside-a-distrobox)
  - [Using docker inside a distrobox](useful_tips.md#using-docker-inside-a-distrobox)
  - [Using init system inside a distrobox](useful_tips.md#using-init-system-inside-a-distrobox)
  - [Using distrobox as main cli](useful_tips.md#using-distrobox-as-main-cli)
  - [Using a different architecture](useful_tips.md#using-a-different-architecture)
  - [Slow creation on podman and image size getting bigger with distrobox-create](useful_tips.md#slow-creation-on-podman-and-image-size-getting-bigger-with-distrobox-create)
  - [Container save and restore](useful_tips.md#container-save-and-restore)
  - [Check used resources](useful_tips.md#check-used-resources)
  - [Build a Gentoo distrobox container](distrobox_gentoo.md)
  - [Build a Dedicated distrobox container](distrobox_custom.md)
- [Posts](posts/posts.md)
  - [Run latest GNOME and KDE Plasma using distrobox](posts/run_latest_gnome_kde_on_distrobox.md)
  - [Integrate VSCode and Distrobox](posts/integrate_vscode_distrobox.md)
  - [Execute a command on the Host](posts/execute_commands_on_host.md)
- [Featured Articles](featured_articles.md)
  - [Articles](featured_articles.md#articles)
    - [Run Distrobox on Fedora Linux - Fedora Magazine](https://fedoramagazine.org/run-distrobox-on-fedora-linux/)
    - [DistroBox – Run Any Linux Distribution Inside Linux Terminal - TecMint](https://www.tecmint.com/distrobox-run-any-linux-distribution/)
    - [Distrobox: Try Multiple Linux Distributions via the Terminal - It's FOSS](https://itsfoss.com/distrobox/)
    - [Distrobox - How to quickly deploy a Linux distribution with GUI applications via a container](https://www.techrepublic.com/article/how-to-quickly-deploy-a-linux-distribution-with-gui-applications-via-a-container/)
    - [Using Distrobox To Augment The Package Selection On Clear Linux - Phoronix](https://www.phoronix.com/scan.php?page=news_item&px=Distrobox-Clear-Linux)
    - [Benchmark: benefits of Clear Linux containers (distrobox) - Phoronix](https://www.phoronix.com/forums/forum/phoronix/latest-phoronix-articles/1305326-clear-linux-container-performance-continues-showing-sizable-gains)
    - [Distrobox - A great item in the Linux toolbelt - phmurphy's blog](https://phmurphy.com/posts/distrobox-toolbelt/)
    - [Distrobox: Run (pretty much) any Linux distro under almost any other - TheRegister](https://www.theregister.com/2022/05/31/distrobox_130_released/)
    - [Day-to-day differences between Fedora Silverblue and Ubuntu - castrojo's blog](https://www.ypsidanger.com/day-to-day-advantages-of-fedora-silverblue/)
    - [Distrobox is Awesome - Running Window Manager and Desktop environments using Distrobox](https://cloudyday.tech.blog/2022/05/14/distrobox-is-awesome/)
    - [Japanese input on Clear Linux with Mozc via Ubuntu container with Distrobox](https://impsbl.hatenablog.jp/entry/JapaneseInputOnClearLinuxWithMozc_en)
    - [MID (MaXX Interactive Desktop) on Clear Linux via Ubuntu container with Distrobox](https://impsbl.hatenablog.jp/entry/MIDonClearLinuxWithDistrobox_en)
    - [Running Other Linux Distros with Distrobox on Fedora Linux - bandithijo's blog](featured_articles.md)
  - [Talks](featured_articles.md#talks)
    - [Linux App Summit 2022 - Distrobox: Run Any App On Any Distro - BoF](https://github.com/89luca89/distrobox/files/8598433/distrobox-las-talk.pdf)
    - [A "Box" Full of Tools and Distros - Dario Faggioli @ OpenSUSE Conference 2022](https://www.youtube.com/watch?v=_RzARte80SQ)
  - [Podcasts](featured_articles.md#podcasts)

---

## What it does

Simply put it's a fancy wrapper around `podman` or `docker` to create and start
containers highly integrated with the hosts.

The distrobox environment is based on an OCI image.
This image is used to create a container that seamlessly integrates with the
rest of the operating system by providing access to the user's home directory,
the Wayland and X11 sockets, networking, removable devices (like USB sticks),
systemd journal, SSH agent, D-Bus,
ulimits, /dev and the udev database, etc...

It implements the same concepts introduced by <https://github.com/containers/toolbox>
but in a simplified way using POSIX sh and aiming at broader compatibility.

All the props go to them as they had the great idea to implement this stuff.

It is divided into 10 commands:

- `distrobox-create` - creates the container
- `distrobox-enter`  - to enter the container
- `distrobox-list` - to list containers created with distrobox
- `distrobox-rm` - to delete a container created with distrobox
- `distrobox-stop` - to stop a running container created with distrobox
- `distrobox-upgrade` - to upgrade one or more running containers created with distrobox at once
- `distrobox-generate-entry` - to create an entry of a created container in the applications list
- `distrobox-init`   - the entrypoint of the container (not meant to be used manually)
- `distrobox-export` - it is meant to be used inside the container,
  useful to export apps and services from the container to the host
- `distrobox-host-exec` - to run commands/programs from the host, while inside
 of the container

It also includes a little wrapper to launch commands with `distrobox COMMAND`
instead of calling the single files.

Please check [the usage docs here](usage/usage.md) and [see some handy tips on how to use it](useful_tips.md)

### See it in action

Thanks to [castrojo](https://github.com/castrojo), you can see Distrobox in
action in this explanatory video on his setup with Distrobox, Toolbx,
Fedora Silverblue on his project [ublue](https://github.com/castrojo/ublue)
(check it out!)

[![Video](https://user-images.githubusercontent.com/598882/153680522-f5903607-2854-4cfb-a186-cba7403745bd.png)](https://www.youtube.com/watch?v=Q2PrISAOtbY)

## Why

- Provide a mutable environment on an immutable OS, like [Endless OS,
  Fedora Silverblue, OpenSUSE MicroOS](compatibility.md#host-distros)  or [SteamOS3](posts/install_rootless.md)
- Provide a locally privileged environment for sudoless setups
  (eg. company-provided laptops, security reasons, etc...)
- To mix and match a stable base system (eg. Debian Stable, Ubuntu LTS, RedHat)
  with a bleeding-edge environment for development or gaming
  (eg. Arch, OpenSUSE Tumbleweed or Fedora with latest Mesa)
- Leverage high abundance of curated distro images for docker/podman to
  manage multiple environments

Refer to the compatiblity list for an overview of supported host's distro
[HERE](compatibility.md#host-distros) and container's distro [HERE](compatibility.md#containers-distros).

### Aims

This project aims to bring **any distro userland to any other distro**
supporting podman or docker.
It has been written in POSIX sh to be as portable as possible and not have
problems with dependencies and glibc version's compatibility.

Refer [HERE](compatibility.md#supported-container-managers) for a list of
supported container managers and minimum supported versions.

It also aims to enter the container **as fast as possible**, every millisecond
adds up if you use the container as your default environment for your terminal:

These are some sample results of `distrobox-enter` on the same container on my
weak laptop:

```console
~$ hyperfine --warmup 3 --runs 100 "distrobox enter bench -- whoami"
Benchmark 1: distrobox enter bench -- whoami
  Time (mean ± σ):     395.6 ms ±  10.5 ms    [User: 167.4 ms, System: 62.4 ms]
  Range (min … max):   297.3 ms … 408.9 ms    100 runs
```

#### Security implications

Isolation and sandboxing is **not** the main aim of the project, on the contrary
it aims to tightly integrate the container with the host.
The container will have complete access to your home, pen drives and so on,
so do not expect it to be highly sandboxed like a plain
docker/podman container or a flatpak.

⚠️ **BE CAREFUL**:⚠️  if you use docker, or you use podman with the `--root/-r` flag,
the containers will run as root, so **root inside the rootful container can modify
system stuff outside the container**,
if you have security concern for this, **use podman that runs in rootless mode**.
Rootless docker is still not working as intended and will be included in the future
when it will be complete.

That said, it is in the works to implement some sort of decoupling with the host,
as discussed here: [#28 Sandboxed mode](https://github.com/89luca89/distrobox/issues/28)

---

# Basic usage

Create a new distrobox:

`distrobox create -n test`

Enter created distrobox:

`distrobox enter test`
  
Add [various](https://github.com/89luca89/distrobox/blob/main/docs/compatibility.md#host-distros)
distroboxes, eg Ubuntu 20.04:

`distrobox create -i ubuntu:20.04`

Execute a command in a distrobox:

`distrobox enter test -- command-to-execute`

Upgrade all distroboxes at once:

`distrobox upgrade --all`

List running distroboxes:

`distrobox list`

Stop a running distrobox:

`distrobox stop test`

Remove a distrobox

`distrobox rm test`

You can check [HERE for more advanced usage](usage/usage.md)
and check a [comprehensive list of useful tips HERE](useful_tips.md)

# Configure Distrobox

Configuration files can be placed in the following paths, from the least important
to the most important:

- /usr/share/distrobox/distrobox.conf
- /usr/etc/distrobox/distrobox.conf
- /etc/distrobox/distrobox.conf
- ${HOME}/.config/distrobox/distrobox.conf
- ${HOME}/.distroboxrc

Example configuration file:

```conf
container_always_pull="1"
container_user_custom_home="/home/.local/share/container-home-test"
container_image="registry.opensuse.org/opensuse/toolbox:latest"
container_manager="docker"
container_name="test-name-1"
container_entry=0
non_interactive="1"
skip_workdir="0"
```

Alternatively it is possible to specify preferences using ENV variables:

- DBX_CONTAINER_ALWAYS_PULL
- DBX_CONTAINER_CUSTOM_HOME
- DBX_CONTAINER_IMAGE
- DBX_CONTAINER_MANAGER
- DBX_CONTAINER_NAME
- DBX_CONTAINER_ENTRY
- DBX_NON_INTERACTIVE
- DBX_SKIP_WORKDIR

---

# Installation

Distrobox is packaged in the following distributions, if your distribution is
on this list, you can refer to your repos for installation:

[![Packaging status](https://repology.org/badge/vertical-allrepos/distrobox.svg)](https://repology.org/project/distrobox/versions)

Thanks to the maintainers for their work: [M0Rf30](https://github.com/M0Rf30),
[alcir](https://github.com/alcir), [dfaggioli](https://github.com/dfaggioli),
[AtilaSaraiva](https://github.com/AtilaSaraiva), [michel-slm](https://github.com/michel-slm)

You can also [follow the guide to install in a rootless manner](posts/install_rootless.md)

## Alternative methods

Here is a list of alternative ways to install distrobox

### Curl

If you like to live your life dangerously, or you want the latest release,
you can trust me and simply run this in your terminal:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh
```

or if you want to select a custom directory to install without sudo:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --prefix ~/.local
```

If you want to install the last development version, directly from last commit on git, you can use:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh -s -- --next
```

or:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --next --prefix ~/.local
```

> **Warning**
> Remember to add prefix-path-you-choose/bin to your PATH, to make it work.

### Git

Alternatively you can clone the project using `git clone` or using the latest
release [HERE](https://github.com/89luca89/distrobox/releases/latest).

Enter the directory and run `./install`, by default it will attempt to install
in `~/.local` but if you run the script as root, it will default to `/usr/local`.
You can specify a custom directory with the `--prefix` flag
such as `./install --prefix ~/.distrobox`.

Prefix explained: main distrobox files get installed to `${prefix}/bin` whereas
the manpages get installed to `${prefix}/share/man`.

### Flatpak

⚠️ ⚠️ ⚠️  This is experimental! ⚠️ ⚠️ ⚠️

You can find flatpak builds of distrobox here:
[io.github.luca.distrobox](https://github.com/89luca89/io.github.luca.distrobox/releases)  
Download the latest release flatpak and run

```sh
flatpak install io.github.luca.distrobox.flatpak
```

You can then run distrobox with:

```sh
flatpak run io.github.luca.distrobox create ...
flatpak run io.github.luca.distrobox enter ...
flatpak run io.github.luca.distrobox list ...
[...]
```

It will  be handy to add an `alias distrobox="flatpak run io.github.luca.distrobox"` to your shell,
so that you can run distrobox commands normally.

Being experimental, please if you encounter problems, report them!

---

Check the [Host Distros](compatibility.md#host-distros) compatibility list for
distro-specific instructions.

## Dependencies

Distrobox depends on a container manager to work, you can choose to install
either podman or docker.

Please look in the [Compatibility Table](compatibility.md#host-distros) for your
distribution notes.

There are ways to install [Podman without root privileges and in home.](compatibility.md#install-podman-in-a-static-manner)
This should play well with completely sudoless setups and with devices like the Steam Deck.

---

## Uninstallation

If you installed distrobox using the `install` script in the default install
directory use this:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/uninstall | sudo sh
```

or if you specified a custom path:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/uninstall | sh -s -- --prefix ~/.local
```

Else if cloned the project using `git clone` or using the latest archive release
from [HERE](https://github.com/89luca89/distrobox/releases/latest),

enter the directory and run `./uninstall`, by default it will assume the install
directory was `/usr/local` if ran as root or `~/.local`,
you can specify another directory if needed with `./uninstall --prefix ~/.local`

---

![distrobox-box](https://user-images.githubusercontent.com/598882/144294113-ab3c62b0-4ff0-488f-8e85-dfecc308e561.png)

---
