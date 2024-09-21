+++
title = "Home"
insert_anchor_links = "right"
+++

![splash](assets/splash.svg#no-hover)

# Distrobox

<small>

Current logo by [daudix](https://daudix.one).  
Previous logo by [hexared](https://github.com/hexared).
</small>

<div id="badges">

[![Lint](https://github.com/89luca89/distrobox/actions/workflows/main.yml/badge.svg#transparent#no-hover)](https://github.com/89luca89/distrobox/actions/workflows/main.yml)
[![CI](https://github.com/89luca89/distrobox/actions/workflows/compatibility.yml/badge.svg#transparent#no-hover)](https://github.com/89luca89/distrobox/actions/workflows/compatibility.yml)
[![GitHub](https://img.shields.io/github/license/89luca89/distrobox?color=blue#transparent#no-hover)](https://github.com/89luca89/distrobox/blob/main/COPYING.md)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/89luca89/distrobox#transparent#no-hover)](https://github.com/89luca89/distrobox/releases/latest)
[![Packaging status](https://repology.org/badge/tiny-repos/distrobox.svg#transparent#no-hover)](https://repology.org/project/distrobox/versions)
[![GitHub issues by-label](https://img.shields.io/github/issues-search/89luca89/distrobox?query=is%3Aissue%20is%3Aopen%20label%3Abug%20-label%3Await-on-user%20&label=Open%20Bug%20Reports&color=red#transparent#no-hover)](https://github.com/89luca89/distrobox/issues?q=is%3Aissue+is%3Aopen+label%3Abug+-label%3Await-on-user)
</div>

Use any Linux distribution inside your terminal. Enable both backward and forward
compatibility with software and freedom to use whatever distribution you’re more
comfortable with.
Distrobox uses `podman`, `docker` or
[`lilipod`](https://github.com/89luca89/lilipod) to create containers using the
Linux distribution of your choice.
The created container will be tightly integrated with the host, allowing sharing
of the HOME directory of the user, external storage, external USB devices and
graphical apps (X11/Wayland), and audio.

---

[Matrix Room](https://matrix.to/#/%23distrobox:matrix.org) -
[Telegram Group](https://t.me/distrobox)

---

![overview](https://user-images.githubusercontent.com/598882/144294862-f6684334-ccf4-4e5e-85f8-1d66210a0fff.png)

---

- [What It Does](#what-it-does)
  - [See It in Action](#see-it-in-action)
- [Why](#why)
  - [Aims](#aims)
    - [Security Implications](#security-implications)
- [Quick Start](#quick-start)
- [Assemble Distrobox](#assemble-distrobox)
- [Configure Distrobox](#configure-distrobox)
- [Installation](#installation)
  - [Alternative Methods](#alternative-methods)
    - [Curl or Wget](#curl-or-wget)
    - [Upgrading](#upgrading)
  - [Git](#git)
- [Dependencies](#dependencies)
- [Uninstallation](#uninstallation)

## What It Does

Simply put it's a fancy wrapper around `podman`, `docker`, or `lilipod` to create and start
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

It is divided into 12 commands:

- `distrobox-assemble` - creates and destroy containers based on a config file
- `distrobox-create` - creates the container
- `distrobox-enter`  - to enter the container
- `distrobox-ephemeral`  - create a temporal container, destroy it when exiting the shell
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

Please check [the usage docs here](@/usage/_index.md) and [see some handy tips on how to use it](@/useful_tips.md)

### See It in Action

Thanks to [castrojo](https://github.com/castrojo), you can see Distrobox in
action in this explanatory video on his setup with Distrobox, Toolbx,
Fedora Silverblue for the [uBlue](https://github.com/ublue-os) project
(check it out!)

{{ youtube(id="Q2PrISAOtbY") }}

## Why

- Provide a mutable environment on an immutable OS, like [ChromeOS, Endless OS,
  Fedora Silverblue, OpenSUSE Aeon/Kalpa, Vanilla OS](@/compatibility.md#host-distros), or [SteamOS3](@/posts/steamdeck_guide.md)
- Provide a locally privileged environment for sudoless setups
  (eg. company-provided laptops, security reasons, etc...)
- To mix and match a stable base system (eg. Debian Stable, Ubuntu LTS, RedHat)
  with a bleeding-edge environment for development or gaming
  (eg. Arch, OpenSUSE Tumbleweed, or Fedora with the latest Mesa)
- Leverage a high abundance of curated distro images for `docker`/`podman` to
  manage multiple environments.

Refer to the compatibility list for an overview of the supported host distros
[HERE](@/compatibility.md#host-distros) and container's distro [HERE](@/compatibility.md#containers-distros).

### Aims

This project aims to bring **any distro userland to any other distro**
supporting `podman`, `docker`, or `lilipod`.
It has been written in POSIX shell to be as portable as possible and it does not have
problems with dependencies and `glibc` version's compatibility.

Refer [HERE](@/compatibility.md#supported-container-managers) for a list of
supported container managers and minimum supported versions.

It also aims to enter the container **as fast as possible**, every millisecond
adds up if you use the container as your default environment for your terminal:

These are some sample results of `distrobox-enter` on the same container on my
weak laptop:

```bash
~$ hyperfine --warmup 3 --runs 100 "distrobox enter bench -- whoami"
Benchmark 1: distrobox enter bench -- whoami
  Time (mean ± σ):     395.6 ms ±  10.5 ms    [User: 167.4 ms, System: 62.4 ms]
  Range (min … max):   297.3 ms … 408.9 ms    100 runs
```

#### Security Implications

Isolation and sandboxing are **not** the main aims of the project, on the contrary
it aims to tightly integrate the container with the host.
The container will have complete access to your home, pen drive, and so on,
so do not expect it to be highly sandboxed like a plain
`docker`/`podman` container or a Flatpak.

{% alert(caution=true) %}
If you use `docker`, or you use `podman`/`lilipod` with the `--root/-r` flag,
the containers will run as root, so **root inside the rootful container can modify
system stuff outside the container**,
Be also aware that **In rootful mode, you'll be asked to setup the user's password**, this will
ensure at least that the container is not a passwordless gate to root,
but if you have security concerns for this, **use `podman` or `lilipod` that runs in rootless mode**.
Rootless `docker` is still not working as intended and will be included in the future
when it will be complete.
{% end %}

That said, it is in the works to implement some sort of decoupling with the host,
as discussed here: [#28 Sandboxed mode](https://github.com/89luca89/distrobox/issues/28)

## Quick Start

Create a new distrobox:

```bash
distrobox create -n test
```

Create a new distrobox with Systemd (acts similar to an LXC):

```bash
distrobox create --name test --init --image debian:latest --additional-packages "systemd libpam-systemd pipewire-audio-client-libraries"
```

Enter created distrobox:

```bash
distrobox enter test
```

Add one with a [different distribution](@/compatibility.md#host-distros),
eg. Ubuntu 20.04:

```bash
distrobox create -i ubuntu:20.04
```

Execute a command in a distrobox:

```bash
distrobox enter test -- command-to-execute
```

List running distroboxes:

```bash
distrobox list
```

Stop a running distrobox:

```bash
distrobox stop test
```

Remove a distrobox:

```bash
distrobox rm test
```

You can check [HERE for more advanced usage](@/usage/_index.md)
and check a [comprehensive list of useful tips HERE](@/useful_tips.md)

## Assemble Distrobox

Manifest files can be used to declare a set of distroboxes and use
`distrobox-assemble` to create/destroy them in batch.

Head over the [usage docs of distrobox-assemble](@/usage/distrobox-assemble.md)
for a more detailed guide.

## Configure Distrobox

Configuration files can be placed in the following paths, from the least important
to the most important:

- /usr/share/distrobox/distrobox.conf
- /usr/etc/distrobox/distrobox.conf
- /etc/distrobox/distrobox.conf
- ${HOME}/.config/distrobox/distrobox.conf
- ${HOME}/.distroboxrc

You can specify inside distrobox configurations and distrobox-specific Environment
variables.

Example configuration file:

```conf
container_always_pull="1"
container_generate_entry=0
container_manager="docker"
container_image_default="registry.opensuse.org/opensuse/toolbox:latest"
container_name_default="test-name-1"
container_user_custom_home="$HOME/.local/share/container-home-test"
container_init_hook="~/.local/distrobox/a_custom_default_init_hook.sh"
container_pre_init_hook="~/a_custom_default_pre_init_hook.sh"
container_manager_additional_flags="--env-file /path/to/file --custom-flag"
container_additional_volumes="/example:/example1 /example2:/example3:ro"
non_interactive="1"
skip_workdir="0"
PATH="$PATH:/path/to/custom/podman"
```

Alternatively, it is possible to specify preferences using ENV variables:

- DBX_CONTAINER_ALWAYS_PULL
- DBX_CONTAINER_CUSTOM_HOME
- DBX_CONTAINER_IMAGE
- DBX_CONTAINER_MANAGER
- DBX_CONTAINER_NAME
- DBX_CONTAINER_ENTRY
- DBX_NON_INTERACTIVE
- DBX_SKIP_WORKDIR

## Installation

Distrobox is packaged in the following distributions, if your distribution is
on this list, you can refer to your repos for installation:

[![Packaging status](https://repology.org/badge/vertical-allrepos/distrobox.svg#no-hover)](https://repology.org/project/distrobox/versions)

Thanks to the maintainers for their work: [M0Rf30](https://github.com/M0Rf30),
[alcir](https://github.com/alcir), [dfaggioli](https://github.com/dfaggioli),
[AtilaSaraiva](https://github.com/AtilaSaraiva), [michel-slm](https://github.com/michel-slm)

### Alternative Methods

Here is a list of alternative ways to install `distrobox`.

#### Curl or Wget

If you like to live your life dangerously, or you want the latest release,
you can trust me and simply run this in your terminal:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh
```

or using wget

```bash
wget -qO- https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh
```

or if you want to select a custom directory to install without sudo:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --prefix ~/.local
```

or using wget

```bash
wget -qO- https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --prefix ~/.local
```

If you want to install the last development version, directly from the last commit on Git, you can use:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh -s -- --next
```

or using wget

```bash
wget -qO- https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh -s -- --next
```

or:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --next --prefix ~/.local
```

or using wget

```bash
wget -qO- https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --next --prefix ~/.local
```

#### Upgrading

Just run the `curl` or `wget` command again.

{% alert(warning=true) %}
Remember to add prefix-path-you-choose/bin to your PATH, to make it work.
{% end %}

### Git

Alternatively, you can clone the project using `git clone` or using the latest
release [HERE](https://github.com/89luca89/distrobox/releases/latest).

Enter the directory and run `./install`, by default it will attempt to install
in `~/.local` but if you run the script as root, it will default to `/usr/local`.
You can specify a custom directory with the `--prefix` flag
such as `./install --prefix ~/.distrobox`.

Prefix explained: main distrobox files get installed to `${prefix}/bin` whereas
the manpages get installed to `${prefix}/share/man`.

---

Check the [Host Distros](@/compatibility.md#host-distros) compatibility list for
distro-specific instructions.

## Dependencies

Distrobox depends on a container manager to work, you can choose to install
either `podman`, `docker` or [`lilipod`](https://github.com/89luca89/lilipod).

Please look in the [Compatibility Table](@/compatibility.md#host-distros) for your
distribution notes.

There are ways to install
[Podman without root privileges and in home.](@/posts/install_podman_static.md) or
[Lilipod without root privileges and in home.](@/posts/install_lilipod_static.md)
This should play well with completely sudoless setups and with devices like the Steam Deck (SteamOS).

## Uninstallation

If you installed `distrobox` using the `install` script in the default install
directory use this:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/uninstall | sudo sh
```

or if you specified a custom path:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/uninstall | sh -s -- --prefix ~/.local
```

Else if cloned the project using `git clone` or using the latest archive release
from [HERE](https://github.com/89luca89/distrobox/releases/latest),

enter the directory and run `./uninstall`, by default it will assume the install
directory was `/usr/local` if ran as root or `~/.local`,
you can specify another directory if needed with `./uninstall --prefix ~/.local`

---

![distro-box](assets/distro-box.webp)

<small>

This artwork uses:

- [Cardboard Box](https://sketchfab.com/3d-models/a-cardboard-box-d527ec1f29a14513a6e8712a1e980f91)
model by [J0Y](https://sketchfab.com/lloydrostek),
licensed under [Creative Commons Attribution 4.0](http://creativecommons.org/licenses/by/4.0).  
- [GTK Loop Animation](https://github.com/gnome-design-team/gnome-mockups/blob/1ce3f25304e31540a7fc65aa775e854c15404a20/gtk/loop6.blend)
by the [GNOME Project](https://www.gnome.org),
licensed under [Creative Commons Attribution-ShareAlike 3.0](https://creativecommons.org/licenses/by-sa/3.0)
as a pre-configured scene.
- [Distribution Icons](https://www.reddit.com/r/linux/comments/nt1tm9/i_made_a_uniform_icon_set_of_linux_distribution)
by [u/walrusz](https://www.reddit.com/user/walrusz/).
</small>
