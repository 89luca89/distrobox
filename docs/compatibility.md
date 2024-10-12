- [Distrobox](README.md)
  - [Compatibility](#compatibility)
    - [Supported container managers](#supported-container-managers)
    - [Host Distros](#host-distros)
      - [Compatibility notes](#compatibility-notes)
      - [Non shared mounts](#non-shared-mounts)
      - [List of distributions including distrobox in their repositories](#list-of-distributions-including-distrobox-in-their-repositories)
      - [New Host Distro support](#new-host-distro-support)
    - [Containers Distros](#containers-distros)
      - [New Distro support](#new-distro-support)
      - [Older distributions](#older-distributions)

---

# Compatibility

This project **does not need a dedicated image**. It can use any OCI images from
docker-hub, quay.io, or any registry of your choice.

Many cloud images are stripped down on purpose to save size and may not include
commands such as `which`, `mount`, `less` or `vi`). Additional packages can be
installed once inside the container. We recommend using your preferred automation
tool inside the container if you find yourself having to repeatedly create new containers.
Maintaining your own custom image is also an option.

The main concern is having basic Linux utilities (`mount`), basic user management
utilities (`usermod, passwd`), and `sudo` correctly set.

## Supported container managers

`distrobox` can run on either `podman`, `docker` or [`lilipod`](https://github.com/89luca89/lilipod)

It depends either on `podman` configured in `rootless mode`
or on `docker` configured without sudo (follow [THESE instructions](https://docs.docker.com/engine/install/linux-postinstall/))

- Minimum podman version: **2.1.0**
- Minimum docker client version: **19.03.15**
- Minimum lilipod version: **v0.0.1**

Follow the official installation guide here:

- <https://podman.io/getting-started/installation>
- <https://docs.docker.com/engine/install>
- <https://docs.docker.com/engine/install/linux-postinstall/>

## Host Distros

Distrobox has been successfully tested on:

|    Distro  |    Version    | Notes |
| --- | --- | --- |
| Alpine Linux | | To setup rootless podman, look [HERE](https://wiki.alpinelinux.org/wiki/Podman) |
| Arch Linux | | `distrobox` is available in the `extra` repository and `distrobox-git` is available in the AUR (thanks [M0Rf30](https://github.com/M0Rf30)!). <br> To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| Bazzite | 38 | `distrobox-git` is preinstalled. |
| CentOS | 8 <br> 8 Stream <br> 9 Stream | `distrobox` is available in epel repos. (thanks [alcir](https://github.com/alcir)!) |
| ChromeOS | Debian 11 (docker with make-shared workaround #non-shared-mounts) <br> Debian 12 (podman) | using built-in Linux on ChromeOS mode which is debian-based, which can be [upgraded](https://wiki.debian.org/DebianUpgrade) from 11 bullseye to 12 bookworm (in fact 12 is recommended) |
| Debian | 11 <br> 12 <br> Testing <br> Unstable | `distrobox` is available in default repos starting from version 12 (thanks [michel-slm!](https://github.com/michel-slm!)!) |
| deepin | 23 <br> Testing <br> Unstable | `distrobox` is available in default repos in `testing` and `unstable` |
| EndlessOS | 4.0.0 | |
| Fedora Silverblue/Kinoite | 35 <br> 36 <br> 37 <br> Rawhide | `distrobox` is available in default repos.(thanks [alcir](https://github.com/alcir)!) |
| Fedora | 35 <br> 36 <br> 37 <br> 38 <br> Rawhide | `distrobox` is available in default repos.(thanks [alcir](https://github.com/alcir)!) |
| Gentoo | | To setup rootless podman, look [HERE](https://wiki.gentoo.org/wiki/Podman) |
| KDE neon | | `distrobox` is available in default repo |
| Manjaro | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| NixOS | 21.11 | Make sure to mind your executable paths. Sometimes a container will not have nix paths, and sometimes it will not have its own paths.  <br>  Distrobox is available in Nixpkg collection (thanks [AtilaSaraiva](https://github.com/AtilaSaraiva)!)< <br> To setup Docker, look [HERE](https://wiki.nixos.org/wiki/Docker)  <br> To setup Podman, look [HERE](https://wiki.nixos.org/wiki/Podman) and [HERE](https://gist.github.com/adisbladis/187204cb772800489ee3dac4acdd9947) |
| openSUSE | Leap | `distrobox` is available in default repos (thanks [dfaggioli](https://github.com/dfaggioli)!). <br> Prior to Leap 15.6 ``podman`` logging needs to be configured properly, more details in [this openSUSE bug](https://bugzilla.opensuse.org/show_bug.cgi?id=1199871). |
| openSUSE | Tumbleweed <br> Slowroll <br> Aeon/Kalpa | `distrobox` is available in default repos (thanks [dfaggioli](https://github.com/dfaggioli)!) <br> For Tumbleweed/Slowroll, do: `zypper install distrobox`. <br> For Aeon/Kalpa, **distrobox is installed by default**. |
| SUSE Linux Enterprise Server | 15 SP5 <br> or later | `distrobox` is available in `SUSE Package Hub` repo. <br> Enable this repo and then: <br> `zypper install distrobox`. <br>Prior to SLES 15 SP6 ``podman`` logging needs to be configured properly, more details in [this openSUSE bug](https://bugzilla.opensuse.org/show_bug.cgi?id=1199871). |
| SteamOS | | You can follow the [Install Podman in a static manner](posts/install_podman_static.md) or [Install Lilipod in a static manner](posts/install_lilipod_static.md) guide, this will install it in your $HOME and it will survive updates. |
| RedHat | 8 <br> 9  | `distrobox` is available in epel repos. (thanks [alcir](https://github.com/alcir)!) |
| Ubuntu | 18.04 <br> 20.04 <br> 22.04 <br> 22.10 <br> 23.04 <br>| Older versions based on 20.04 or earlier may need external repos to install newer Podman and Docker releases. <br> Derivatives like Pop_OS!, Mint and Elementary OS should work the same. <br> [Now PPA available!](https://launchpad.net/~michel-slm/+archive/ubuntu/distrobox), also `distrobox` is available in default repos from `22.10` onward (thanks [michel-slm](https://github.com/michel-slm)!)  |
| Vanilla OS | 22.10 <br> Orchid | `distrobox` should be installed in the home directory using the official script |
| Void Linux | glibc <br> musl | |
| Windows | Oracle Linux 9 | using built-in Windows Subsystem for Linux |

### Compatibility notes

### Non shared mounts

Note also that in some distributions, root filesystem is **not** mounted as a shared mount,
this will give an error like:

```sh
$ distrobox-enter
Error response from daemon: path /sys is mounted on /sys but it is not a shared or slave mount
Error: failed to start containers: ...
```

To resolve this, use this command:

```sh
mount --make-rshared /
```

To make it permanent, you can place it in `/etc/rc.local`.

## List of distributions including distrobox in their repositories

[![Packaging status](https://repology.org/badge/vertical-allrepos/distrobox.svg)](https://repology.org/project/distrobox/versions)

### New Host Distro support

If your distro of choice is not on the list, open an issue requesting support
for it, we can work together to check if it is possible to add support for it.

Or just try using it anyway, if it works, open an issue
and it will be added to the list!

---

## Containers Distros

Distrobox guests tested successfully with the following container images:

|    Distro  |    Version | Images    |
| --- | --- | --- |
| AlmaLinux (Toolbox) | 8 <br> 9 | quay.io/toolbx-images/almalinux-toolbox:8 <br> quay.io/toolbx-images/almalinux-toolbox:9 <br> quay.io/toolbx-images/almalinux-toolbox:latest |
| Alpine (Toolbox) | 3.16 <br> 3.17 <br> 3.18 <br> 3.19 <br> 3.20 <br> edge | quay.io/toolbx-images/alpine-toolbox:3.16 <br> quay.io/toolbx-images/alpine-toolbox:3.17 <br> quay.io/toolbx-images/alpine-toolbox:3.18 <br> quay.io/toolbx-images/alpine-toolbox:3.19 <br> quay.io/toolbx-images/alpine-toolbox:3.20 <br> quay.io/toolbx-images/alpine-toolbox:edge <br> quay.io/toolbx-images/alpine-toolbox:latest |
| AmazonLinux (Toolbox) | 2 <br> 2022 | quay.io/toolbx-images/amazonlinux-toolbox:2 <br> quay.io/toolbx-images/amazonlinux-toolbox:2023 <br> quay.io/toolbx-images/amazonlinux-toolbox:latest |
| Archlinux (Toolbox) | | quay.io/toolbx/arch-toolbox:latest |
| Alt Linux | p10 <br> p11 <br> sisyphus | docker.io/library/alt:p10 <br> docker.io/library/alt:p11 <br> docker.io/library/alt:sisyphus |
| Bazzite Arch | | ghcr.io/ublue-os/bazzite-arch:latest <br> ghcr.io/ublue-os/bazzite-arch-gnome:latest |
| Centos (Toolbox) | stream8 <br> stream9 | quay.io/toolbx-images/centos-toolbox:stream8 <br> quay.io/toolbx-images/centos-toolbox:stream9 <br> quay.io/toolbx-images/centos-toolbox:latest |
| Debian (Toolbox) | 10 <br> 11 <br> 12 <br> testing <br> unstable <br> | quay.io/toolbx-images/debian-toolbox:10 <br> quay.io/toolbx-images/debian-toolbox:11 <br> quay.io/toolbx-images/debian-toolbox:12 <br> quay.io/toolbx-images/debian-toolbox:testing <br> quay.io/toolbx-images/debian-toolbox:unstable <br> quay.io/toolbx-images/debian-toolbox:latest |
| Fedora (Toolbox) | 37 <br> 38 <br> 39 <br> 40 <br> Rawhide | registry.fedoraproject.org/fedora-toolbox:37 <br> registry.fedoraproject.org/fedora-toolbox:38 <br> registry.fedoraproject.org/fedora-toolbox:39 <br> registry.fedoraproject.org/fedora-toolbox:40 <br> registry.fedoraproject.org/fedora-toolbox:rawhide |
| openSUSE (Toolbox) | | registry.opensuse.org/opensuse/distrobox:latest |
| RedHat (Toolbox) | 8 <br> 9 | registry.access.redhat.com/ubi8/toolbox <br> registry.access.redhat.com/ubi9/toolbox |
| Rocky Linux (Toolbox) | 8 <br> 9 | quay.io/toolbx-images/rockylinux-toolbox:8 <br> quay.io/toolbx-images/rockylinux-toolbox:9 <br> quay.io/toolbx-images/rockylinux-toolbox:latest |
| Ubuntu (Toolbox) | 16.04 <br> 18.04 <br> 20.04 <br> 22.04 <br> 24.04 | quay.io/toolbx/ubuntu-toolbox:16.04 <br> quay.io/toolbx/ubuntu-toolbox:18.04 <br> quay.io/toolbx/ubuntu-toolbox:20.04 <br> quay.io/toolbx/ubuntu-toolbox:22.04 <br> quay.io/toolbx/ubuntu-toolbox:24.04 <br> quay.io/toolbx/ubuntu-toolbox:latest |
| Chainguard Wolfi (Toolbox) | | quay.io/toolbx-images/wolfi-toolbox:latest |
| Ublue | bluefin-cli <br> ubuntu-toolbox <br> fedora-toolbox <br> wolfi-toolbox <br> archlinux-distrobox <br> powershell-toolbox | ghcr.io/ublue-os/bluefin-cli <br> ghcr.io/ublue-os/bluefin-cli <br> ghcr.io/ublue-os/ubuntu-toolbox <br> ghcr.io/ublue-os/fedora-toolbox <br> ghcr.io/ublue-os/wolfi-toolbox <br> ghcr.io/ublue-os/arch-distrobox <br> ghcr.io/ublue-os/powershell-toolbox |
|  |  |  |
| AlmaLinux | 8 <br> 8-minimal <br> 9 <br> 9-minimal | docker.io/library/almalinux:8 <br> docker.io/library/almalinux:9  |
| Alpine Linux    | 3.15 <br> 3.16 <br> 3.17 <br> 3.18 <br> 3.19 <br> 3.20 <br> edge | docker.io/library/alpine:3.15 <br> docker.io/library/alpine:3.16 <br> docker.io/library/alpine:3.17 <br> docker.io/library/alpine:3.18 <br> docker.io/library/alpine:3.19 <br> docker.io/library/alpine:3.20 <br> docker.io/library/alpine:edge <br> docker.io/library/alpine:latest |
| AmazonLinux | 1 <br> 2 <br> 2023 | public.ecr.aws/amazonlinux/amazonlinux:1 <br> public.ecr.aws/amazonlinux/amazonlinux:2 <br>  public.ecr.aws/amazonlinux/amazonlinux:2023 |
| Archlinux     | | docker.io/library/archlinux:latest    |
| Blackarch     | | docker.io/blackarchlinux/blackarch:latest    |
| CentOS Stream | 8 <br> 9 | quay.io/centos/centos:stream8 <br> quay.io/centos/centos:stream9  |
| Chainguard Wolfi | | cgr.dev/chainguard/wolfi-base:latest |
| ClearLinux |      | docker.io/library/clearlinux:latest <br> docker.io/library/clearlinux:base    |
| Crystal Linux | | registry.gitlab.com/crystal-linux/misc/docker:latest  |
| Debian | 7 <br> 8 <br> 9 <br> 10 <br> 11 <br> 12 | docker.io/debian/eol:wheezy <br> docker.io/library/debian:buster <br> docker.io/library/debian:bullseye-backports <br> docker.io/library/debian:bookworm-backports <br> docker.io/library/debian:stable-backports |
| Debian | Testing    | docker.io/library/debian:testing  <br>  docker.io/library/debian:testing-backports    |
| Debian | Unstable | docker.io/library/debian:unstable    |
| deepin | 20 (apricot) <br> 23 (beige) | docker.io/linuxdeepin/apricot |
| Fedora | 36 <br> 37 <br> 38 <br> 39 <br> 40 <br> Rawhide | quay.io/fedora/fedora:36 <br> quay.io/fedora/fedora:37 <br> quay.io/fedora/fedora:38 <br> quay.io/fedora/fedora:39 <br> quay.io/fedora/fedora:40 <br> quay.io/fedora/fedora:rawhide  |
| Gentoo Linux | rolling | docker.io/gentoo/stage3:latest |
| KDE neon | Latest | invent-registry.kde.org/neon/docker-images/plasma:latest |
| Kali Linux | rolling | docker.io/kalilinux/kali-rolling:latest |
| Mint | 21.1 | docker.io/linuxmintd/mint21.1-amd64 |
| Neurodebian | nd100 | docker.io/library/neurodebian:nd100 |
| openSUSE | Leap | registry.opensuse.org/opensuse/leap:latest    |
| openSUSE | Tumbleweed | registry.opensuse.org/opensuse/distrobox:latest  <br> registry.opensuse.org/opensuse/tumbleweed:latest  <br>  registry.opensuse.org/opensuse/toolbox:latest    |
| Oracle Linux | 7 <br> 7-slim <br> 8 <br> 8-slim <br> 9 <br> 9-slim |container-registry.oracle.com/os/oraclelinux:7 <br> container-registry.oracle.com/os/oraclelinux:7-slim <br> container-registry.oracle.com/os/oraclelinux:8 <br> container-registry.oracle.com/os/oraclelinux:8-slim <br> container-registry.oracle.com/os/oraclelinux:9 <br> container-registry.oracle.com/os/oraclelinux:9-slim  |
| RedHat (UBI) | 7 <br> 8 <br> 9 | registry.access.redhat.com/ubi7/ubi <br> registry.access.redhat.com/ubi8/ubi <br> registry.access.redhat.com/ubi8/ubi-init <br> registry.access.redhat.com/ubi8/ubi-minimal <br> registry.access.redhat.com/ubi9/ubi <br> registry.access.redhat.com/ubi9/ubi-init <br> registry.access.redhat.com/ubi9/ubi-minimal |
| Rocky Linux | 8 <br> 8-minimal <br> 9 | quay.io/rockylinux/rockylinux:8 <br> quay.io/rockylinux/rockylinux:8-minimal <br> quay.io/rockylinux/rockylinux:9 <br> quay.io/rockylinux/rockylinux:latest    |
| Slackware | | docker.io/vbatts/slackware:current |
| SteamOS | | ghcr.io/linuxserver/steamos:latest |
| Ubuntu | 14.04 <br> 16.04 <br> 18.04 <br> 20.04 <br> 22.04 <br> 23.04 | docker.io/library/ubuntu:14.04 <br> docker.io/library/ubuntu:16.04 <br> docker.io/library/ubuntu:18.04 <br> docker.io/library/ubuntu:20.04 <br> docker.io/library/ubuntu:22.04 <br> docker.io/library/ubuntu:23.04 |
| Vanilla OS | VSO | ghcr.io/vanilla-os/vso:main |
| Void Linux | glibc <br> musl | ghcr.io/void-linux/void-glibc-full:latest <br> ghcr.io/void-linux/void-musl-full:latest |

Images marked with **Toolbox** are tailored images made by the community efforts in [toolbx-images/images](https://github.com/toolbx-images/images),
so they are more indicated for desktop use, and first setup will take less time.
Note however that if you use a non-toolbox preconfigured image,
the **first** `distrobox-enter` you'll perform
can take a while as it will download and install the missing dependencies.

A small time tax to pay for the ability to use any type of image.
This will **not** occur after the first time, **subsequent enters will be much faster.**

NixOS is not a supported container distro, and there are currently no plans to
bring support to it. If you are looking for unprivileged NixOS environments,
we suggest you look into [nix-shell](https://nixos.org/manual/nix/unstable/command-ref/nix-shell.html)
or [nix portable](https://github.com/DavHau/nix-portable)

### New Distro support

If your distro of choice is not on the list, open an issue requesting support
for it, we can work together to check if it is possible to add support for it.

Or just try using it anyway, if it works, open an issue
and it will be added to the list!

### Older distributions

For older distributions like CentOS 5, CentOS 6, Debian 6, Ubuntu 12.04,
compatibility is not assured.

Their `libc` version is incompatible with kernel releases after `>=4.11`.
A work around this is to use the `vsyscall=emulate` flag in the bootloader of the
host.

Keep also in mind that mirrors could be down for such old releases, so you will
need to build a [custom distrobox image to ensure basic dependencies are met](./posts/distrobox_custom.md).

### GPU Acceleration support

For Intel and AMD Gpus, the support is baked in, as the containers will install
their latest available mesa/dri drivers.

For NVidia, you can use the `--nvidia` flag during create, see [distrobox-create](./usage/distrobox-create.md)
documentation to discover how to use it.

Alternatively, you can use the [nvidia-container-toolkit](./useful_tips.md#using-nvidia-container-toolkit)
utility to set up the integration independently from the distrobox's own flag.
