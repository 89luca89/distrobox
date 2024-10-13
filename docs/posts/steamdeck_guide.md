### Install Distrobox and Podman PERMANENT on Steam Deck >= 3.5

**1 - Modify $PATH for binaries**
First, verify if ~/.bashrc contains the necessary $PATH modification. Open the file with:
`sudo nano ~/.bashrc`
Add the following line if it’s not already there:
`export PATH=/home/deck/.local/bin:$PATH`

**2 - Install and configure Distrobox**
To install Distrobox in the defined $PATH, use one of the following commands depending on whether you need the latest version (`--next`) or not:
`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sudo sh -s -- --next --prefix ~/.local`
or
`curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/install | sh -s -- --prefix $HOME/.local`

After installing, create the file `~/.distroboxrc` if it doesn't already exist. Open it with:
`sudo nano ~/.distroboxrc`
Add the following lines to configure Distrobox:
```
# Ensure the graphical apps can talk to the Xwayland session
xhost +si:localuser:$USER >/dev/null
# Force the use of pulseaudio inside the container
export PIPEWIRE_RUNTIME_DIR=/dev/null
# Needed to ensure distrobox can find the podman binary we previously downloaded
export PATH=/home/deck/.local/bin:$PATH
export PATH=$PATH:/home/deck/.local/bin
```

**3 - Install and configure Podman**
To install Podman, download the latest version from the GitHub releases page:
[](https://github.com/89luca89/podman-launcher/releases)
`curl -L -o /home/deck/Downloads/podman-launcher-amd64 https://github.com/89luca89/podman-launcher/releases/download/v0.0.5/podman-launcher-amd64`

Next, move and rename the Podman binary with ROOT permissions to the $PATH:
`sudo mv /home/deck/Downloads/podman-launcher-amd64 /home/deck/.local/bin/podman`

Then, make Podman executable:
`chmod +x /home/deck/.local/bin/podman`

Configure the deck user’s UID and GID mapping with the following commands:
`sudo touch /etc/subuid /etc/subgid`
`sudo usermod --add-subuid 100000-165535 --add-subgid 100000-165535 deck`

**4 - Configure Distrobox Icon folder** - (if you install distrobox with sudo)
To ensure Distrobox can store its icons correctly, set the proper permissions on the` /home/deck/.local/share/icons `folder with:
`sudo chown deck:deck /home/deck/.local/share/icons`

**5 - Verify installations**
After the installation steps, verify that both Distrobox and Podman are properly installed and configured. Use the following commands:
`which distrobox`
`which podman`
`distrobox --version`
`podman --version`
`podman info`

**6 - Create and test distros** - install pulseaudio within the distros
You can now create and test containers with Distrobox. To create and test a ROOTLESS container, run:
distrobox create --image docker.io/library/archlinux:latest --name arch
For a ROOT container, use:
distrobox create --image docker.io/library/archlinux:latest --name rarch --root

You can either remove the created distros later or keep them for regular use.
