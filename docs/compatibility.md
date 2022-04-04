- [Distrobox](README.md)
  - [Supported container managers](#supported-container-managers)
  - [Host Distros](#host-distros)
    - [New Host Distro support](#new-host-distro-support)
  - [Containers Distros](#containers-distros)
    - [New Distro support](#new-distro-support)
    - [Older Distributions](#older-distributions)

---

# Compatibility

This project **does not need a dedicated image**. It can use any OCI images from docker-hub, quay.io, or any registry of your choice.

Granted, they may not be as featureful as expected (some of them do not even have `which`, `mount`, `less` or `vi`)
but that's all doable in the container itself after bootstrapping it.

The main concern is having basic Linux utilities (`mount`), basic user management utilities (`usermod, passwd`), and `sudo` correctly set.

### Supported container managers

`distrobox` can run on either `podman` or `docker`

It depends either on `podman` configured in `rootless mode`
or on `docker` configured without sudo (follow [THIS instructions](https://docs.docker.com/engine/install/linux-postinstall/))

- Minimum podman version: **2.1.0**
- Minimum docker version: **18.06.1**

Follow the official installation guide here:

- <https://podman.io/getting-started/installation>
- <https://docs.docker.com/engine/install>
- <https://docs.docker.com/engine/install/linux-postinstall/>

### Host Distros

Distrobox has been successfully tested on:

|    Distro  |    Version    | Notes |
| --- | --- | --- |
| Alpine Linux | 3.14.3 | To setup rootless podman, look [HERE](https://wiki.alpinelinux.org/wiki/Podman) |
| Arch Linux | | `distrobox` and `distrobox-git` are available in AUR (thanks [M0Rf30](https://github.com/M0Rf30)!).<br>To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| Manjaro | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| CentOS | 8<br>8 Stream<br>9 Stream | `distrobox` is available in epel repos. (thanks [alcir](https://github.com/alcir)!) |
| RedHat | 8<br>9beta  | `distrobox` is available in epel repos. (thanks [alcir](https://github.com/alcir)!) |
| Debian | 11<br>Testing<br>Unstable | |
| Fedora | 34<br>35<br>36 | `distrobox` is available in default repos.(thanks [alcir](https://github.com/alcir)!) |
| Fedora Silverblue/Kinoite | 34<br>35<br>36 | `distrobox` is available in default repos.(thanks [alcir](https://github.com/alcir)!) |
| Gentoo | | To setup rootless podman, look [HERE](https://wiki.gentoo.org/wiki/Podman) |
| Ubuntu | 18.04<br>20.04<br>21.10 | Older versions based on 20.04 or earlier may need external repos to install newer Podman and Docker releases.<br>Please follow their installation guide: [Podman](https://podman.io/getting-started/installation) [Docker](https://docs.docker.com/engine/install/ubuntu/)<br> Derivatives like Pop_OS!, Mint and Elementary OS should work the same. |
| EndlessOS | 4.0.0 | |
| openSUSE | Tumbleweed<br>MicroOS | `distrobox` is available in default repos (thanks [dfaggioli](https://github.com/dfaggioli)!)<br>For Tumbleweed, do: `zypper install distrobox`.<br>For MicroOS enter in a [transactional update](https://kubic.opensuse.org/documentation/transactional-update-guide/transactional-update.html) shell like this: `tukit --continue execute /bin/bash` (or `transactional-update shell --continue`, if you have `transactional-update` installed). Once inside: `zypper install distrobox`. Then exit the shell (`CTRL+D` is fine) and reboot the system. |
| openSUSE | Leap 15.4<br>Leap 15.3<br>Leap 15.2 | Packages are available [here](https://software.opensuse.org/download/package?package=distrobox&project=home%3Adfaggioli%3Amicroos-desktop) (thanks [dfaggioli](https://github.com/dfaggioli)!)<br>To install on openSUSE Leap 15.4, do:<br>`zypper addrepo https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.4/home:dfaggioli:microos-desktop.repo && zypper refresh && zypper install distrobox`.<br>For earlier versions, the procedure is the same, the link to the repository (i.e., the last argument of the `zypper addrepo` command) is the only thing that changes: [Leap 15.3](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/openSUSE_Leap_15.3/home:dfaggioli:microos-desktop.repo), [Leap 15.2](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/openSUSE_Leap_15.2/home:dfaggioli:microos-desktop.repo). |
| SUSE Linux Enterprise Server | 15&nbsp;Service&nbsp;Pack&nbsp;4<br>15&nbsp;Service&nbsp;Pack&nbsp;3<br>15&nbsp;Service&nbsp;Pack&nbsp;2 | Same procedure as the one for openSUSE (Leap, respective versions, of course). Use the following repository links in the `zypper addrepo` command: [SLE-15-SP4](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.4/home:dfaggioli:microos-desktop.repo), [SLE-15-SP3](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/15.3/home:dfaggioli:microos-desktop.repo), [SLE-15-SP4](https://download.opensuse.org/repositories/home:dfaggioli:microos-desktop/SLE_15_SP2/home:dfaggioli:microos-desktop.repo). |
| Void Linux | glibc | Systemd service export will not work. |
| NixOS | 21.11 | Currently you must have your default shell set to Bash, if it is not, make sure you edit your configuration.nix so that it is. <br>Also make sure to mind your executable paths. Sometimes a container will not have nix paths, and sometimes it will not have its own paths. <br> Distrobox is available in Nixpkg collection (thanks [AtilaSaraiva](https://github.com/AtilaSaraiva)!)<<br>To setup Docker, look [HERE](https://nixos.wiki/wiki/Docker) <br>To setup Podman, look [HERE](https://nixos.wiki/wiki/Podman) and [HERE](https://gist.github.com/adisbladis/187204cb772800489ee3dac4acdd9947) |
| Windows WSL2 | | __NOTE WSL2 support is preliminary, and there are many bugs present, any help in improving support is appreciated__ <br> Currently you must work around some incompatibility between WSL2 and Podman, namely [THIS](https://github.com/containers/podman/issues/12236). <br>Install into WSL2 any of the supported distributions in this list. <br> Ensure you have an entry in the `fstab` for the `/tmp` folder:<br> `echo 'tmpfs /tmp tmps defaults 0 0' >> /etc/fstab`.<br>Then reboot the WSL machine `wsl --shutdown` <br>Note that `distrobox export` is not supported on WSL2 and will not work. |

If your container is not able to connect to your host xserver, make sure to install `xhost` on the host machine
and run `xhost +si:localuser:$USER`. If you wish to enable this functionality on future reboots add it to your `~/.xinitrc`
or somewhere else tailored to your use case where it would be ran on every startup.

List of distributions including distrobox in their repositories:

[![Packaging status](https://repology.org/badge/vertical-allrepos/distrobox.svg)](https://repology.org/project/distrobox/versions)

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
| AlmaLinux (UBI) | 8     | docker.io/almalinux/8-base<br>docker.io/almalinux/8-init    |
| Alpine Linux    | 3.14<br>3.15 | docker.io/library/alpine:latest    |
| AmazonLinux | 2  | docker.io/library/amazonlinux:2.0.20211005.0    |
| AmazonLinux | 2022  | public.ecr.aws/amazonlinux/amazonlinux:2022 |
| Archlinux     | | docker.io/library/archlinux:latest    |
| ClearLinux |      | docker.io/library/clearlinux:latest<br>docker.io/library/clearlinux:base    |
| CentOS | 7 | quay.io/centos/centos:7  |
| CentOS Stream | 8<br>9 | quay.io/centos/centos:stream8<br>quay.io/centos/centos:stream9  |
| RedHat (UBI) | 7<br>8 | registry.access.redhat.com/ubi7/ubi<br>registry.access.redhat.com/ubi7/ubi-init<br>registry.access.redhat.com/ubi8/ubi<br>registry.access.redhat.com/ubi8/ubi-init  |
| Debian | 8<br>9<br>10<br>11 | docker.io/library/debian:8<br>docker.io/library/debian:9<br>docker.io/library/debian:10<br>docker.io/library/debian:stable<br>docker.io/library/debian:stable-backports    |
| Debian | Testing    | docker.io/library/debian:testing <br> docker.io/library/debian:testing-backports    |
| Debian | Unstable | docker.io/library/debian:unstable    |
| Neurodebian | nd100 | docker.io/library/neurodebian:nd100 |
| Fedora | 34<br>35<br>36<br>37<br>Rawhide | registry.fedoraproject.org/fedora-toolbox:34<br> docker.io/library/fedora:34<br>registry.fedoraproject.org/fedora-toolbox:35<br>docker.io/library/fedora:35<br>docker.io/library/fedora:36<br>registry.fedoraproject.org/fedora:37<br>docker.io/library/fedora:rawhide    |
| Mageia | 8 | docker.io/library/mageia |
| Opensuse | Leap | registry.opensuse.org/opensuse/leap:latest    |
| Opensuse | Tumbleweed | registry.opensuse.org/opensuse/tumbleweed:latest <br> registry.opensuse.org/opensuse/toolbox:latest    |
| Oracle Linux | 7<br>8 | container-registry.oracle.com/os/oraclelinux:7<br>container-registry.oracle.com/os/oraclelinux:8    |
| Rocky Linux | 8 | docker.io/rockylinux/rockylinux:8    |
| Scientific Linux | 7 | docker.io/library/sl:7    |
| Slackware | 14.2 | docker.io/vbatts/slackware:14.2    |
| Ubuntu | 14.04<br>16.04<br>18.04<br>20.04<br>21.10<br>22.04 | docker.io/library/ubuntu:14.04<br>docker.io/library/ubuntu:16.04<br>docker.io/library/ubuntu:18.04<br>docker.io/library/ubuntu:20.04<br>docker.io/library/ubuntu:21.10<br>docker.io/library/ubuntu:22.04    |
| Kali Linux | rolling | docker.io/kalilinux/kali-rolling:latest |
| Void Linux | | ghcr.io/void-linux/void-linux:latest-full-x86_64 <br> ghcr.io/void-linux/void-linux:latest-full-x86_64-musl |
| Gentoo Linux | rolling | You will have to [Build your own](distrobox_gentoo.md) to have a complete Gentoo docker image |

Note however that if you use a non-toolbox preconfigured image (e.g. images pre-baked to work with <https://github.com/containers/toolbox),> the **first** `distrobox-enter` you'll perform
can take a while as it will download and install the missing dependencies.

A small time tax to pay for the ability to use any type of image.
This will **not** occur after the first time, **subsequent enters will be much faster.**

NixOS is not a supported container distro, and there are currently no plans to bring support to it. If you are looking for unprivlaged NixOS environments, we suggest you look into [nix-shell](https://nixos.org/manual/nix/unstable/command-ref/nix-shell.html).

#### New Distro support

If your distro of choice is not on the list, open an issue requesting support for it,
we can work together to check if it is possible to add support for it.

Or just try using it anyway, if it works, open an issue
and it will be added to the list!

#### Older distributions

For older distributions like CentOS 6, Debian 7, Ubuntu 12.04, compatibility is not
assured.

Their `libc` version is incompatible with kernel releases after `>=4.11`.
A work around this is to use the `vsyscall=emulate` flag in the bootloader of the
host.

Keep also in mind that mirrors could be down for such old releases, so you will
need to build a [custom distrobox image to ensure basic dependencies are met](./distrobox_custom.md).
