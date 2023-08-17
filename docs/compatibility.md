- [Distrobox](README.md)
  - [Compatibility](#compatibility)
    - [Supported container managers](#supported-container-managers)
    - [Host Distros](#host-distros)
      - [Install Podman in a static manner](#install-podman-in-a-static-manner)
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

`distrobox` can run on either `podman` or `docker`

It depends either on `podman` configured in `rootless mode`
or on `docker` configured without sudo (follow [THESE instructions](https://docs.docker.com/engine/install/linux-postinstall/))

- Minimum podman version: **2.1.0**
- Minimum docker client version: **19.03.15**

Follow the official installation guide here:

- <https://podman.io/getting-started/installation>
- <https://docs.docker.com/engine/install>
- <https://docs.docker.com/engine/install/linux-postinstall/>

## Host Distros

Distrobox has been successfully tested on:

|    Distro  |    Version    | Notes |
| --- | --- | --- |
| Alpine Linux | | To setup rootless podman, look [HERE](https://wiki.alpinelinux.org/wiki/Podman) |
| Arch Linux | | `distrobox` and `distrobox-git` are available in AUR (thanks [M0Rf30](https://github.com/M0Rf30)!). <br> To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| CentOS | 8 <br> 8 Stream <br> 9 Stream | `distrobox` is available in epel repos. (thanks [alcir](https://github.com/alcir)!) |
| ChromeOS | Debian 11 (docker with make-shared workaround #non-shared-mounts) <br> Debian 12 (podman) | using built-in Linux on ChromeOS mode which is debian-based, which can be [upgraded](https://wiki.debian.org/DebianUpgrade) from 11 bullseye to 12 bookworm (in fact 12 is recommended) |
| Debian | 11 <br> Testing <br> Unstable | `distrobox` is available in default repos in `testing` and `unstable` (thanks [michel-slm!](https://github.com/michel-slm!)!) |
| deepin | 23 <br> Testing <br> Unstable | `distrobox` is available in default repos in `testing` and `unstable` |
| EndlessOS | 4.0.0 | |
| Fedora Silverblue/Kinoite | 35 <br> 36 <br> 37 <br> Rawhide | `distrobox` is available in default repos.(thanks [alcir](https://github.com/alcir)!) |
| Fedora | 35 <br> 36 <br> 37 <br> 38 <br> Rawhide | `distrobox` is available in default repos.(thanks [alcir](https://github.com/alcir)!) |
| Gentoo | | To setup rootless podman, look [HERE](https://wiki.gentoo.org/wiki/Podman) |
| KDE neon | | `distrobox` is available in default repo |
| Manjaro | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| NixOS | 21.11 | Make sure to mind your executable paths. Sometimes a container will not have nix paths, and sometimes it will not have its own paths.  <br>  Distrobox is available in Nixpkg collection (thanks [AtilaSaraiva](https://github.com/AtilaSaraiva)!)< <br> To setup Docker, look [HERE](https://nixos.wiki/wiki/Docker)  <br> To setup Podman, look [HERE](https://nixos.wiki/wiki/Podman) and [HERE](https://gist.github.com/adisbladis/187204cb772800489ee3dac4acdd9947) |
| openSUSE | Leap 15.4 <br> Leap 15.3 <br> Leap 15.2 | Packages are available [here](https://software.opensuse.org/download/package?package=distrobox&project=home%3Adfaggioli%3Amicroos-desktop) (thanks [dfaggioli](https://github.com/dfaggioli)!).<br> To install on openSUSE Leap 15, Use the following repository links in the `zypper addrepo` command: [15.4](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.4/home:dfaggioli:microos-desktop.repo), [15.3](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.3/home:dfaggioli:microos-desktop.repo), [15.2](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.2/home:dfaggioli:microos-desktop.repo). Then: <br>  `zypper refresh && zypper install distrobox`. <br> `Podman` under SUSE Leap, cannot initialize correctly the containers managed by ``distrobox`` until [this openSUSE bug](https://bugzilla.opensuse.org/show_bug.cgi?id=1199871) is fixed, or ``podman`` logging is configured properly. |
| openSUSE | Tumbleweed <br> MicroOS | `distrobox` is available in default repos (thanks [dfaggioli](https://github.com/dfaggioli)!) <br> For Tumbleweed, do: `zypper install distrobox`. <br> For MicroOS, **distrobox is installed by default**. |
| SUSE Linux Enterprise Server | 15&nbsp;Service&nbsp;Pack&nbsp;4 <br> 15&nbsp;Service&nbsp;Pack&nbsp;3 <br> 15&nbsp;Service&nbsp;Pack&nbsp;2 | Same procedure as the one for openSUSE (Leap, respective versions, of course). Use the following repository links in the `zypper addrepo` command: [SLE-15-SP4](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.4/home:dfaggioli:microos-desktop.repo), [SLE-15-SP3](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.3/home:dfaggioli:microos-desktop.repo), [SLE-15-SP4](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/SLE_15_SP2/home:dfaggioli:microos-desktop.repo). Then: <br>  `zypper refresh && zypper install distrobox`. <br> `Podman` under SUSE Leap, cannot initialize correctly the containers managed by ``distrobox`` until [this openSUSE bug](https://bugzilla.opensuse.org/show_bug.cgi?id=1199871) is fixed, or ``podman`` logging is configured properly. |
| SteamOS 3 | | You can use `steamos-readonly disable` and follow `Arch Linux` instructions. This will **NOT** survive updates.<br>Alternatively you can follow the [Install Podman in a static manner](posts/install_rootless.md) guide, this will install it in your $HOME and it will survive updates.|
| RedHat | 8 <br> 9  | `distrobox` is available in epel repos. (thanks [alcir](https://github.com/alcir)!) |
| Ubuntu | 18.04 <br> 20.04 <br> 22.04 <br> 22.10 <br> 23.04 <br>| Older versions based on 20.04 or earlier may need external repos to install newer Podman and Docker releases. <br> Derivatives like Pop_OS!, Mint and Elementary OS should work the same. <br> [Now PPA available!](https://launchpad.net/~michel-slm/+archive/ubuntu/distrobox), also `distrobox` is available in default repos from `22.10` onward (thanks [michel-slm](https://github.com/michel-slm)!)  |
| Void Linux | glibc | |

### Install Podman in a static manner

If on your distribution (eg. SteamOS) can be difficult to install something and keep it
between updates, then you could use this script to install `podman` in your `$HOME`.

This has some limitations, for starters, it won't work in `rootful` mode for now,
but otherwise it's working for normal use.

This is particularly indicated also for completely *sudoless* setups, where you don't
have any superuser access to the system, like for example company provided computers.

Run:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/extras/install-podman | sh -s -- --prefix ~/.local
```

Provided the only dependency on the host (`newuidmap/newgidmap`, of the package `uidmap`),
you should be good to go.

To uninstall:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/extras/install-podman | sh -s -- --prefix ~/.local --remove
```

> **Warning**
> Remember to add ~/.local/podman/bin to your PATH, to make it work.
> Remember to add ~/.local/podman/bin to your ~/.distroboxrc also.

#### Compatibility notes

If your container is not able to connect to your host xserver, make sure to
install `xhost` on the host machine and run `xhost +si:localuser:$USER`.
If you wish to enable this functionality on future reboots add it to your `~/.xinitrc`
or somewhere else tailored to your use case where it would be ran on every startup.

#### Non shared mounts

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

### List of distributions including distrobox in their repositories

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
| AlmaLinux (Toolbox) | 8 <br> 9 | quay.io/toolbx-images/almalinux-toolbox:8 <br> quay.io/toolbx-images/almalinux-toolbox:9 |
| Alpine (Toolbox) | 3.16 <br> 3.17 <br> 3.18 <br> edge | quay.io/toolbx-images/alpine-toolbox:3.16 <br> quay.io/toolbx-images/alpine-toolbox:3.17 <br> quay.io/toolbx-images/alpine-toolbox:3.18 <br> quay.io/toolbx-images/alpine-toolbox:edge |
| AmazonLinux (Toolbox) | 2 <br> 2022 | quay.io/toolbx-images/amazonlinux-toolbox:2 <br> quay.io/toolbx-images/amazonlinux-toolbox:2022 |
| Archlinux (Toolbox) | | quay.io/toolbx-images/archlinux-toolbox |
| Centos (Toolbox) | stream8 <br> stream9 | quay.io/toolbx-images/centos-toolbox:stream8 <br> quay.io/toolbx-images/centos-toolbox:stream9 |
| Debian (Toolbox) | 10 <br> 11 <br> 12 <br> testing <br> unstable <br> | quay.io/toolbx-images/debian-toolbox:10 <br> quay.io/toolbx-images/debian-toolbox:11 <br> quay.io/toolbx-images/debian-toolbox:12 <br>  quay.io/toolbx-images/debian-toolbox:testing <br>  quay.io/toolbx-images/debian-toolbox:unstable |
| Fedora (Toolbox) | 36 <br> 37 <br> 38 <br> 39 <br> Rawhide | registry.fedoraproject.org/fedora-toolbox: 37 <br> registry.fedoraproject.org/fedora-toolbox:38 <br> registry.fedoraproject.org/fedora-toolbox:39 |
| openSUSE (Toolbox) | | registry.opensuse.org/opensuse/distrobox:latest <br> quay.io/toolbx-images/opensuse-toolbox:latest |
| RedHat (Toolbox) | 8 <br> 9 | registry.access.redhat.com/ubi8/toolbox <br> registry.access.redhat.com/ubi9/toolbox |
| Rocky Linux (Toolbox) | 8 <br> 9 | quay.io/toolbx-images/rockylinux-toolbox:8 <br> quay.io/toolbx-images/rockylinux-toolbox:9 |
| Ubuntu (Toolbox) | 16.04 <br> 18.04 <br> 20.04 <br> 22.04 | quay.io/toolbx-images/ubuntu-toolbox:16.04 <br> quay.io/toolbx-images/ubuntu-toolbox:18.04 <br> quay.io/toolbx-images/ubuntu-toolbox:20.04 <br> quay.io/toolbx-images/ubuntu-toolbox:22.04 |
|  |  |  |
| AlmaLinux (UBI) | 8 | quay.io/almalinux/8-base:8 <br> quay.io/almalinux/8-init:8 |
| AlmaLinux | 8 <br> 8-minimal <br> 9 <br> 9-minimal | quay.io/almalinux/almalinux:8 <br> quay.io/almalinux/almalinux:9 <br> quay.io/almalinux/almalinux:9-minimal |
| Alpine Linux    | 3.15 <br> 3.16 | docker.io/library/alpine:3.15 <br> docker.io/library/alpine:3.16 <br> docker.io/library/alpine:latest |
| AmazonLinux | 1 <br> 2 <br> 2022 | public.ecr.aws/amazonlinux/amazonlinux:1 <br> public.ecr.aws/amazonlinux/amazonlinux:2 <br>  public.ecr.aws/amazonlinux/amazonlinux:2022.0.20220531.0 |
| Archlinux     | | docker.io/library/archlinux:latest    |
| CentOS Stream | 8 <br> 9 | quay.io/centos/centos:stream8 <br> quay.io/centos/centos:stream9  |
| CentOS | 7 | quay.io/centos/centos:7  |
| Chainguard Wolfi | Small note: sudo is missing, use su-exec instead. | cgr.dev/chainguard/wolfi-base:latest |
| ClearLinux |      | docker.io/library/clearlinux:latest <br> docker.io/library/clearlinux:base    |
| Crystal Linux | | registry.getcryst.al/crystal/misc/docker:latest |
| Debian | 7 <br> 8 <br> 9 <br> 10 <br> 11 <br> 12 | docker.io/debian/eol:wheezy <br> docker.io/library/debian:buster-backports <br> docker.io/library/debian:bullseye-backports <br> docker.io/library/debian:bookworm-backports <br> docker.io/library/debian:stable-backports |
| Debian | Testing    | docker.io/library/debian:testing  <br>  docker.io/library/debian:testing-backports    |
| Debian | Unstable | docker.io/library/debian:unstable    |
| deepin | 20 (apricot) <br> 23 (beige) | docker.io/linuxdeepin/apricot  <br>  docker.io/linuxdeepin/beige    |
| Fedora | 36 <br> 37 <br> 38 <br> 39 <br> Rawhide | quay.io/fedora/fedora:36 <br> registry.fedoraproject.org/fedora:37 <br> quay.io/fedora/fedora:38 <br> quay.io/fedora/fedora:39  |
| Gentoo Linux | rolling | docker.io/gentoo/stage3:latest |
| KDE neon | Latest <br> Testing <br> Unstable | invent-registry.kde.org/neon/docker-images/plasma:latest <br> invent-registry.kde.org/neon/docker-images/plasma:testing <br> invent-registry.kde.org/neon/docker-images/plasma:unstable |
| Kali Linux | rolling | docker.io/kalilinux/kali-rolling:latest |
| Mint | 21.1 | docker.io/linuxmintd/mint21.1-amd64 |
| Neurodebian | nd100 | docker.io/library/neurodebian:nd100 |
| openEULER |  | docker.io/openeuler/openeuler:latest |
| openSUSE | Leap | registry.opensuse.org/opensuse/leap:latest    |
| openSUSE | Tumbleweed | registry.opensuse.org/opensuse/tumbleweed:latest  <br>  registry.opensuse.org/opensuse/toolbox:latest    |
| Oracle Linux | 7 <br> 7-slim <br> 8 <br> 8-slim <br> 9 <br> 9-slim |container-registry.oracle.com/os/oraclelinux:7 <br> container-registry.oracle.com/os/oraclelinux:7-slim <br> container-registry.oracle.com/os/oraclelinux:8 <br> container-registry.oracle.com/os/oraclelinux:8-slim <br> container-registry.oracle.com/os/oraclelinux:9 <br> container-registry.oracle.com/os/oraclelinux:9-slim  |
| RedHat (UBI) | 7 <br> 8 <br> 9 | registry.access.redhat.com/ubi7/ubi <br> registry.access.redhat.com/ubi7/ubi-init <br> registry.access.redhat.com/ubi8/ubi <br> registry.access.redhat.com/ubi8/ubi-init <br> registry.access.redhat.com/ubi8/ubi-minimal <br> registry.access.redhat.com/ubi9/ubi <br> registry.access.redhat.com/ubi9/ubi-init <br> registry.access.redhat.com/ubi9/ubi-minimal |
| Rocky Linux | 8 <br> 8-minimal <br> 9 | quay.io/rockylinux/rockylinux:8 <br> quay.io/rockylinux/rockylinux:8-minimal <br> quay.io/rockylinux/rockylinux:9 <br> quay.io/rockylinux/rockylinux:latest    |
| Scientific Linux | 7 | docker.io/library/sl:7    |
| Slackware | 14.2 | docker.io/vbatts/slackware:14.2    |
| Ubuntu | 14.04 <br> 16.04 <br> 18.04 <br> 20.04 <br> 22.04 <br> 23.04 | docker.io/library/ubuntu:14.04 <br> docker.io/library/ubuntu:16.04 <br> docker.io/library/ubuntu:18.04 <br> docker.io/library/ubuntu:20.04 <br> docker.io/library/ubuntu:22.04 |
| Void Linux | | ghcr.io/void-linux/void-linux:latest-full-x86_64 |

Images marked with **Toolbox** are tailored images, so they are more indicated
for desktop use, and first setup will take less time.
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
need to build a [custom distrobox image to ensure basic dependencies are met](./distrobox_custom.md).

### GPU Acceleration support

For Intel and AMD Gpus, the support is baked in, as the containers will install
their latest available mesa/dri drivers.

For NVidia, you can use the `--nvidia` flag during create, see [distrobox-create](./usage/distrobox-create.md)
documentation to discover how to use it.

Alternatively, you can use the [nvidia-container-toolkit](./useful_tips.md#using-nvidia-container-toolkit)
utility to set up the integration independently from the distrobox's own flag.
