# Install Podman in a static manner

If on your distribution (eg. SteamOS) can be difficult to install something and keep it
between updates, then you could use this script to install `podman` in your `$HOME`.

Installing distrobox in HOME is quite straightforward:

1. Install `distrobox` in your HOME following the `curl` instructions:
   - [INSTALL](../README.md#curl-or-wget)
2. Add the Path you've chosen to install to your PATH (by default it's `$HOME/.local/bin`.
   - [See here how to do it](https://www.howtogeek.com/658904/how-to-add-a-directory-to-your-path-in-linux/)

This is particularly indicated also for completely *sudoless* setups, where you don't
have any superuser access to the system, like for example company provided computers.

Download the latest release of [podman-launcher](https://github.com/89luca89/podman-launcher/releases)
and put it somewhere in your $PATH

Provided the only dependency on the host (`newuidmap/newgidmap`, of the package `uidmap` or `shadow`),
you should be good to go.

To uninstall, just delete the binary.

On some systems, like SteamOS, you might have to enable this command in order to have graphical applications working:
[SEE THESE NOTES](../compatibility.md#compatibility-notes)

To resolve add this line to your `~/.distroboxrc`:

  `xhost +si:localuser:$USER`

---

After this, you can open a new terminal, and have Distrobox working!
