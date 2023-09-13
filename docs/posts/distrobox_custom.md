- [Distrobox](README.md)

---

# Create a dedicated distrobox container

Distrobox wants to be as generic as possible in supporting OCI images,
but sometimes there could be some problems:

- The image you want to use is too old and the package manager mirrors are down
- The image you want to use has not a supported package manager or no package
  manager at all

## Requirements

The only required programs that must be available in the container so that
`distrobox-init` won't start the installation are:

- the $SHELL you use (bash, zsh, fish etc etc)
- bash-completion
- bc
- bzip2
- curl
- diffutils
- findutils
- gnupg2
- hostname
- iproute
- iputils
- keyutils
- krb5-libs
- less
- lsof
- man-db
- man-pages
- ncurses
- nss-mdns
- openssh-clients
- pam
- passwd
- pigz
- pinentry
- ping
- procps-ng
- rsync
- shadow-utils
- sudo
- tcpdump
- time
- traceroute
- tree
- tzdata
- unzip
- util-linux
- vte-profile
- wget
- which
- whois
- words
- xorg-x11-xauth
- xz
- zip

And optionally:

- mesa-dri-drivers
- mesa-vulkan-drivers
- vulkan

If all those dependencies are met, then the `distrobox-init`
will simply skip the installation process and work as expected.

To test if all packages requirements are met just run this in the container:

```shell
dependencies="
    bc
    bzip2
    chpasswd
    curl
    diff
    find
    findmnt
    gpg
    hostname
    less
    lsof
    man
    mount
    passwd
    pigz
    pinentry
    ping
    ps
    rsync
    script
    ssh
    sudo
    time
    tree
    umount
    unzip
    useradd
    wc
    wget
    xauth
    zip
"
for dep in ${dependencies}; do
    ! command -v "${dep}" && echo "missing $dep"
done
```
