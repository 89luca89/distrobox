- [Distrobox](README.md)
  - [Useful tips](useful_tips.md)

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
- findutils
- ncurses
- passwd
- pinentry
- procps-ng
- shadow-utils
- sudo
- util-linux (that provides the mount command)
- vte-profile

If all those dependencies are met, then the `distrobox-init`
will simply skip the installation process and work as expected.

To test if all packages requirements are met just run this in the container:

```shell
if ! command -v mount || ! command -v mount || ! command -v passwd ||
 ! command -v sudo || ! command -v useradd || ! command -v usermod ||
 ! command -v "${SHELL}"; then

 echo "Missing dependencies"

fi
```
