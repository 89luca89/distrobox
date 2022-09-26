# Install on SteamOS 3

## Step 1

Installing distrobox on SteamOS is quite straightforward:

1. Install `distrobox` in your HOME following the `curl` instructions:
   - [INSTALL](../README.md#curl)
2. Add the Path you've choosen to install to your PATH (by default it's `$HOME/.local/bin`.
   - [See here how to do it](https://www.howtogeek.com/658904/how-to-add-a-directory-to-your-path-in-linux/)

## Step 2

We now need `podman` to be installed in order for `distrobox` to work.
The easiest way is to use the script to install it in HOME, so it will survive future updates:

1. Install `podman` in your HOME following the `curl` command:
   - [INSTALL](../compatibility.md#install-podman-in-a-static-manner)
2. Add the Path you've choosen to install to your PATH (by default it's `$HOME/.local/podman/bin`.
   - [See here how to do it](https://www.howtogeek.com/658904/how-to-add-a-directory-to-your-path-in-linux/)

## Step 3

On some systems, you might have to enable this command in order to have graphical applications working: [SEE THESE NOTES](../compatibility.md#compatibility-notes)

To resolve add this line to your `~/.bashrc` or `~/.profile` or `~/.xinitrc`

  `xhost +si:localuser:$USER`

---

After this, you can open a new terminal, and have Distrobox working!
