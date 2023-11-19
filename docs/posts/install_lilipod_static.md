# Install Lilipod in a static manner

If on your distribution (eg. SteamOS) can be difficult to install something and keep it
between updates, then you could use this guide to install [lilipod](https://github.com/89luca89/lilipod) in your `$HOME`.

[Lilipod](https://github.com/89luca89/lilipod) is a very simple container manager with minimal features to:

- Download and manager images
- Create and run containers

To install `lilipod`:

1. Add the Path you've chosen to install to your PATH (by default it's `$HOME/.local/bin`.
   - [See here how to do it](https://www.howtogeek.com/658904/how-to-add-a-directory-to-your-path-in-linux/)
2. Ensure you have /etc/subuid and /etc/subgid, if you don't do:
   - `sudo touch /etc/subuid /etc/subgid`
   - `sudo usermod --add-subuid 100000-165535 --add-subgid 100000-165535 $USER`

This is particularly indicated also for completely *sudoless* setups, where you don't
have any superuser access to the system, like for example company provided computers.

Download the latest release of [lilipod](https://github.com/89luca89/lilipod/releases)
and put it somewhere in your $PATH

Provided the only dependency on the host (`newuidmap/newgidmap`, of the package `uidmap` or `shadow`),
you should be good to go.

To uninstall, just delete the binary.
