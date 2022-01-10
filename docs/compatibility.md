- [Distrobox](README.md)
    + [Supported container managers](#supported-container-managers)
    + [Host Distros](#host-distros)
      - [New Host Distro support](#new-host-distro-support)
    + [Containers Distros](#containers-distros)
      - [New Distro support](#new-distro-support)

---

# Compatibility

This project **does not need a dedicated image**. It can use any OCI images from docker-hub, quay.io, or any registry of your choice.

Granted, they may not be as featureful as expected (some of them do not even have `which`, `mount`, `less` or `vi`)
but that's all doable in the container itself after bootstrapping it.

The main concern is having basic Linux utilities (`mount`), basic user management utilities (`usermod, passwd`), and `sudo` correctly set.

### Supported container managers

`distrobox` can run on either `podman` or `docker`

It depends either on `podman` configured in `rootless mode`
or on `docker` configured without sudo (you're in the `docker` group)

- Minimum podman version: **2.1.0**
- Minimum docker version: **18.06.1**

Follow the official installation guide here:

  - https://podman.io/getting-started/installation
  - https://docs.docker.com/engine/install

### Host Distros

Distrobox has been successfully tested on:

|    Distro  |    Version    | Notes |
| --- | --- | --- |
| Alpine Linux | 3.14.3 | To setup rootless podman, look [HERE](https://wiki.alpinelinux.org/wiki/Podman) |
| Arch Linux | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| Manjaro | | To setup rootless podman, look [HERE](https://wiki.archlinux.org/title/Podman) |
| CentOS | 8<br>8 Stream | Works with corresponding RedHat releases. |
| Debian | 11<br>Testing<br>Unstable | |
| Fedora | 34<br>35 | |
| Fedora Silverblue | 34<br>35 | |
| Gentoo | | To setup rootless podman, look [HERE](https://wiki.gentoo.org/wiki/Podman) |
| Ubuntu | 20.04<br>21.10 | Older versions based on 20.04 needs external repos to install newer Podman and Docker releases. <br> Derivatives like Pop_OS!, Mint and Elementary OS should work the same. |
| EndlessOS | 4.0.0 | |
| OpenSUSE | Leap 15<br>Tumbleweed | |
| OpenSUSE MicroOS | 20211209 | |
| Void Linux | glibc | Systemd service export will not work. |
| NixOS | 21.11 | __NOTE NixOS support is preliminary, and there are many bugs present, any help in improving support is appreciated__ <br> Currently you must have your default shell set to Bash, if it is not, make sure you edit your configuration.nix so that it is. <br> To install distrobox:<br>`mkdir -p ~/.local/bin`<br>Add `PATH=$PATH:$HOME/.local/bin` to your bashrc<br>Execute [THIS](#installation) command without sudo.<br>To setup Docker, look [HERE](https://nixos.wiki/wiki/Docker) <br>To setup Podman, look [HERE](https://nixos.wiki/wiki/Podman) and [HERE](https://gist.github.com/adisbladis/187204cb772800489ee3dac4acdd9947) |
| Windows WSL2 | | __NOTE WSL2 support is preliminary, and there are many bugs present, any help in improving support is appreciated__ <br> Currently you must work around some incompatibility between WSL2 and Podman, namely [THIS](https://github.com/containers/podman/issues/12236). <br>Install into WSL2 any of the supported distributions in this list. <br> Ensure you have an entry in the `fstab` for the `/tmp` folder:<br> `echo 'tmpfs /tmp tmps defaults 0 0' >> /etc/fstab`.<br>Then reboot the WSL machine `wsl --shutdown` <br>Note that `distrobox export` is not supported on WSL2 and will not work. |

If your container is not able to connect to your host xserver, make sure to install `xhost` on the host machine
and run `xhost +si:localuser:$USER`. If you wish to enable this functionality on future reboots add it to your `~/.xinitrc`
or somewhere else tailored to your use case where it would be ran on every startup.

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
| Alpine Linux    | 3.14<br>3.15 | docker.io/library/alpine:latest    |
| AmazonLinux | 2  | docker.io/library/amazonlinux:2.0.20211005.0    |
| Archlinux     | | docker.io/library/archlinux:latest    |
| CentOS | 7<br>8 | quay.io/centos/centos:7<br>quay.io/centos/centos:8  |
| CentOS Stream | 8<br>9 | quay.io/centos/centos:stream8<br>quay.io/centos/centos:stream9  |
| Debian | 8<br>9<br>10<br>11 | docker.io/library/debian:8<br>docker.io/library/debian:9<br>docker.io/library/debian:10<br>docker.io/library/debian:stable<br>docker.io/library/debian:stable-backports    |
| Debian | Testing    | docker.io/library/debian:testing <br> docker.io/library/debian:testing-backports    |
| Debian | Unstable | docker.io/library/debian:unstable    |
| Neurodebian | nd100 | docker.io/library/neurodebian:nd100 |
| Fedora | 34<br>35 | registry.fedoraproject.org/fedora-toolbox:34<br> docker.io/library/fedora:34<br>registry.fedoraproject.org/fedora-toolbox:35<br>docker.io/library/fedora:35    |
| Mageia | 8 | docker.io/library/mageia |
| Opensuse | Leap | registry.opensuse.org/opensuse/leap:latest    |
| Opensuse | Tumbleweed | registry.opensuse.org/opensuse/tumbleweed:latest <br> registry.opensuse.org/opensuse/toolbox:latest    |
| Oracle Linux | 7<br>8 | container-registry.oracle.com/os/oraclelinux:7<br>container-registry.oracle.com/os/oraclelinux:8    |
| Rocky Linux | 8 | docker.io/rockylinux/rockylinux:8    |
| Scientific Linux | 7 | docker.io/library/sl:7    |
| Slackware | 14.2 | docker.io/vbatts/slackware:14.2    |
| Slackware | current | docker.io/vbatts/slackware:current    |
| Ubuntu | 14.04<br>16.04<br>18.04<br>20.04<br>21.10<br>22.04 | docker.io/library/ubuntu:14.04<br>docker.io/library/ubuntu:16.04<br>docker.io/library/ubuntu:18.04<br>docker.io/library/ubuntu:20.04<br>docker.io/library/ubuntu:21.10<br>docker.io/library/ubuntu:22.04    |
| Kali Linux | rolling | docker.io/kalilinux/kali-rolling:latest |
| Void Linux | | ghcr.io/void-linux/void-linux:latest-thin-bb-x86_64 <br> ghcr.io/void-linux/void-linux:latest-thin-bb-x86_64-musl <br> ghcr.io/void-linux/void-linux:latest-full-x86_64 <br> ghcr.io/void-linux/void-linux:latest-full-x86_64-musl |
| Gentoo Linux | rolling | You will have to [Build your own](distrobox_gentoo.md) to have a complete Gentoo docker image |

Note however that if you use a non-toolbox preconfigured image (e.g. images pre-baked to work with https://github.com/containers/toolbox), the **first** `distrobox-enter` you'll perform
can take a while as it will download and install the missing dependencies.

A small time tax to pay for the ability to use any type of image.
This will **not** occur after the first time, **subsequent enters will be much faster.**

#### New Distro support

If your distro of choice is not on the list, open an issue requesting support for it,
we can work together to check if it is possible to add support for it.

Or just try using it anyway, if it works, open an issue
and it will be added to the list!
