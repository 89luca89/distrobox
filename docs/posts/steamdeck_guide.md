Latest SteamOS (version 3.5 and later) already pre-installed `distrobox` and `podman`.

Before using `distrobox` on SteamOS, it may be necessary to upgrade to the latest version since the version provided by
SteamOS may be outdated. You can verify the currently installed version by running the command `distrobox version`. For
instance, on SteamOS 3.5, version 1.4.2.1-3 of `distrobox` is installed.

To upgrade `distrobox` on SteamOS, you have two options:

### Option 1: Install `distrobox` in `$HOME`

By installing `distrobox` in your `$HOME` directory, you can ensure that you have control over the version you're using,
independent of SteamOS updates. This method prevents your modifications from being reverted when SteamOS is updated.

Note that it's essential to add this new version of `distrobox` to your PATH to ensure it's utilized over the
SteamOS-provided version.

To install `distrobox` in the `$HOME` directory, run the following command:

```sh
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --prefix $HOME
```

For more detailed installation instructions, refer to the documentation
[here](https://github.com/89luca89/distrobox/blob/main/docs/README.md#alternative-methods).

To upgrade the version of `distrobox`, follow the instructions provided in the documentation link above.

### Option 2: Overwrite the provided `distrobox` installation in SteamOS

An alternative approach is to upgrade the version of `distrobox` provided by SteamOS. While this simplifies management
as you don't need to modify your PATH and you wouldn't have 2 versions of `distrobox` installed, it comes with the
downside that your upgrades will be overwritten when SteamOS is updated.

To upgrade the `distrobox` version provided by SteamOS, execute the following commands:

```sh
sudo steamos-readonly disable
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh -s -- --prefix /usr
sudo steamos-readonly enable
```

Please note that disabling the read-only state is necessary to perform this upgrade. You can find more information about
this requirement [here](https://help.steampowered.com/en/faqs/view/671A-4453-E8D2-323C).

Once `distrobox` is upgraded, you can use it as normal.

---

To run GUI application, add following line to `~/.distroboxrc`.

```sh
xhost +si:localuser:$USER
```

This is needed to ensure the graphical apps can talk to the Xwayland session.

You can now start using `distrobox` on the deck, open the terminal and go:

```sh
distrobox create && distrobox enter
```

Refer to the [quickstart guide](../README.md#quick-start) and to the [usage docs](../usage/usage.md)
And don't forget the [useful tips](../useful_tips.md)!

## SteamOS 3.4 and earlier

To install Distrobox on the steamdeck, we can install both `podman` and `distrobox`
inside the `$HOME` so that containers will survive updates.

## Install Podman

To install podman, [refer to the install guide](install_podman_static.md#):

- Download the latest release of `podman-launcher` and place it in your home and rename it to `podman`,
  this example will use `~/.local/bin`
- Make the `podman` binary executable:
  - `chmod +x ~/.local/bin/podman`
- Setup `deck` user password using:
  - `passwd`
- Setup `deck` user uidmap:
  - `sudo touch /etc/subuid /etc/subgid`
  - `sudo usermod --add-subuid 100000-165535 --add-subgid 100000-165535 deck`

And `podman` is ready to use!

### Alternative Install Lilipod

To install [lilipod](https://github.com/89luca89/lilipod), [refer to the install guide](install_lilipod_static.md#):

- Download the latest release of `lilipod` and place it in your home and rename it to `lilipod`,
  this example will use `~/.local/bin`
- Setup `deck` user password using:
  - `passwd`
- Setup `deck` user uidmap:
  - `sudo touch /etc/subuid /etc/subgid`
  - `sudo usermod --add-subuid 100000-165535 --add-subgid 100000-165535 deck`

And `lilipod` is ready to use!

## Install Distrobox

Installing distrobox in HOME is quite straightforward:

- Install `distrobox` in your HOME following the `curl` instructions:
  - [INSTALL](../README.md#curl-or-wget)

## Setup ~/.distroboxrc

We need to add some tweaks to our `~/.distroboxrc` to have GUI and Audio working
correctly in SteamOS

Ensure your `~/.distroboxrc` has this content:

```sh
xhost +si:localuser:$USER
export PIPEWIRE_RUNTIME_DIR=/dev/null
export PATH=$PATH:$HOME/.local/bin
```

This will force the use of `pulseaudio` inside the container, right now `pipewire`
is not working correctly inside the container, and it's a SteamOS specific issue.

`xhost` is needed to ensure the graphical apps can talk to the Xwayland session.

`PATH` is needed to ensure distrobox can find the `podman` binary we previously
downloaded.

## Start using it

You can now start using `distrobox` on the deck, open the terminal and go:

`distrobox create && distrobox enter`

Refer to the [quickstart guide](../README.md#quick-start) and to the [usage docs](../usage/usage.md)
And don't forget the [useful tips](../useful_tips.md)!
