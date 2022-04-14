- [Distrobox](../README.md)
  - [Run latest GNOME and KDE using distrobox](run_latest_gnome_kde_on_distrobox.md)
    - [Using a stable-release distribution](#using-a-stable-release-distribution)
      - [Initializing the distrobox](#initializing-the-distrobox)
      - [Running Latest GNOME](#running-latest-gnome)
        - [Generate session file - GNOME](#generate-session-file-gnome)
      - [Running Latest KDE](#running-latest-kde)
        - [Generate session file - KDE](#generate-session-file-kde)
        - [Add a couple of fixes](#add-a-couple-of-fixes)
    - [Using apps from host](#using-apps-from-host)

---

# Using a stable-release distribution

Lots of people prefer to run a distribution following a stable-LTS release cycle
like Debian, UbuntuLTS or CentOS family (Almalinux, Rocky Linux).
This ensures great stability on one hand, but package staling on the other.

One way to counter this effect is to use a pet-container managed by Distrobox
to run packages from much newer distributions without giving up on core base os stability.

## Initializing the distrobox

For this experiment we'll use Fedora Rawhide as our distrobox, and Centos 8 Stream
as our host, so:

```shell
distrobox create --name fedora-rawhide --image registry.fedoraproject.org/fedora:rawhide
```

and

```shell
distrobox enter fedora-rawhide
```

## Running Latest GNOME

First we need to change a couple of bits in the distrobox container to make host's
systemd session accessible from within the host:

```shell
~$ distrobox enter fedora-rawhide
user@fedora-rawhide:~$ rm -rf /run/systemd/system
user@fedora-rawhide:~$ ln -s /run/host/run/systemd/system /run/systemd
```

Then we can proceed to install GNOME in the container:

```shell
user@fedora-rawhide:~$ sudo dnf groupinstall GNOME
```

And let's grab a coffee while it finishes :-)

After the `dnf` process finishes, we have GNOME installed in our container,
now how do we use it?

### Generate session file - GNOME

First in the host we need a reliable way to fix the permissions problem of the
`/tmp/.X11-unix` directory. This directory should either belong to `root` or
`$USER`. But in a rootless container, host's `root` is not mapped inside the
container so we need to change the ownership from `root` to `$USER` each time.

Let's add:

```shell
chown -R $USER:$USER /tmp/.X11-unix
```

to `/etc/profile.d/fix_tmp.sh` file.

This is needed for the XWayland session to work properly which right now is
necessary to run gnome-shell even on wayland.

Then we need to add a desktop file for the session on the host's file system,
so that it appears on your login manager (Be it SSDM or GDM)

```shell
[Desktop Entry]
Name=GNOME on Wayland (fedora-rawhide distrobox)
Comment=This session logs you into GNOME
Exec=/usr/local/bin/distrobox-enter -n fedora-rawhide -- /usr/bin/gnome-session --builtin
Type=Application
DesktopNames=GNOME
X-GDM-SessionRegisters=true
```

This file should be placed under `/usr/local/share/wayland-sessions/distrobox-gnome.desktop`

Let's log out and voilá!

![image](https://user-images.githubusercontent.com/598882/148703229-82905d23-f3d0-41bc-a048-d12cdf8066d0.png)
![Screenshot from 2021-12-25 19-56-52](https://user-images.githubusercontent.com/598882/147391814-cb49e7b8-64bc-4975-a8d1-93f6fb23f28b.png)
![Screenshot from 2021-12-25 20-03-16](https://user-images.githubusercontent.com/598882/147391867-ca29576b-8fb9-448c-a181-579482fb448d.png)

We now are in a GNOME 42 session inside Fedora Rawhide while our main OS remains
Centos.

## Running Latest KDE

We can do the same with KDE also, let's first set up the host's systemd session
sharing with the container:

```shell
~$ distrobox enter fedora-rawhide
user@fedora-rawhide:~$ rm -rf /run/systemd/system
user@fedora-rawhide:~$ ln -s /run/host/run/systemd/system /run/systemd
```

Then we can proceed to install KDE in the container:

```shell
user@fedora-rawhide:~$ sudo dnf groupinstall KDE
```

### Generate session file - KDE

We need to add a desktop file for the session on the host's file system,
so that it appears on your login manager (Be it SSDM or GDM)

```shell
[Desktop Entry]
Exec=/usr/local/bin/distrobox-enter -- /usr/libexec/plasma-dbus-run-session-if-needed /usr/bin/startplasma-wayland
DesktopNames=KDE
Name=Plasma on Wayland (fedora-rawhide distrobox)
X-KDE-PluginInfo-Version=5.23.3
```

This file should be placed under `/usr/local/share/wayland-sessions/distrobox-plasma.desktop`

### Add a couple of fixes

To make KDE work we need a couple more fixes to run both on the host and in the container.

First in the host we need a reliable way to fix the permissions problem of the
`/tmp/.X11-unix` directory. This directory should either belong to `root` or
`$USER`. But in a rootless container, host's `root` is not mapped inside the
container so we need to change the ownership from `root` to `$USER` each time.

Let's add:

```shell
chown -R $USER:$USER /tmp/.X11-unix
```

to `/etc/profile.d/fix_tmp.sh` file.

We also need to add a process in autostart on which Plasma shell relies on a
process called `kactivitymanagerd`. Not having host's systemd at disposal we
can start it simply adding it to the ~/.profile file, add:

```shell
if [ -f /usr/libexec/kactivitymanagerd ]; then
  /usr/libexec/kactivitymanagerd & disown
fi
```

to `~/.profile` file.

Let's log out and voilá!

![image](https://user-images.githubusercontent.com/598882/148704789-3d799a85-51cc-4de7-9ee3-f54add4949bc.png)
![image](https://user-images.githubusercontent.com/598882/148705044-7271af0c-0675-42f8-9f45-ad20ec53deca.png)

We now are in latest KDE session inside Fedora Rawhide while our main OS remains
Centos.

# Using apps from host

Now that we're in a container session, we may want to still use some of the host's
apps. Refer to [THIS](execute_commands_on_host.md) to create handlers and wrappers
to use the complete selection of host's apps and binaries inside the container.
