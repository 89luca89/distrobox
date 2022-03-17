- [Distrobox](README.md)
  * [Execute complex commands directly from distrobox-enter](#execute-complex-commands-directly-from-distrobox-enter)
  * [Improve distrobox-enter performance](#improve-distrobox-enter-performance)
  * [Slow creation on podman and image size getting bigger with distrobox-create](#slow-creation-on-podman-and-image-size-getting-bigger-with-distrobox-create)
  * [Container save and restore](#container-save-and-restore)
  * [Check used resources](#check-used-resources)
  * [Using podman inside a distrobox](#using-podman-inside-a-distrobox)
  * [Using docker inside a distrobox](#using-docker-inside-a-distrobox)
  * [Using init system inside a distrobox](#using-init-system-inside-a-distrobox)
  * [Using distrobox as main cli](#using-distrobox-as-main-cli)
  * [Build a Gentoo distrobox container](distrobox_gentoo.md)
  * [Build a Dedicated distrobox container](distrobox_custom.md)

---

# Useful tips

## Execute complex commands directly from distrobox-enter

Sometimes it is necessary to execure complex commands from a distrobox enter, like multiple concatenated commands using variables
declared **inside** the container.

For example:

`distrobox enter test -- bash -l -c '"echo \$HOME && whoami"'`

Note the use of **single quotes around double quotes**, this is necessary so that quotes are preserved inside the arguments.
Also note the **dollar escaping** needed so that $HOME is not evaluated at the time of the command launch, but directly 
inside the container.

## Enable SSH X-Forwarding when SSH-ing in a distrobox

SSH X-forwarding by default will not work because the container hostname is different from the host's one.
You can create a distrobox with will have the same hostname as the host by creating it with the following
init-hook:

`distrobox-create --name test --image your-choosen-image:tag --init-hooks "$(uname -n)" > /etc/hostname`

This will ensure SSH X-Forwarding will work when SSH-ing inside the distrobox:

`ssh -X myhost distrobox-enter test -- xclock`

## Improve distrobox-enter performance

If you are experiencing a bit slow performance using `podman` you should enable
the podman socket using

`systemctl --user enable --now podman.socket`

this will improve a lot `podman`'s command performances.

## Slow creation on podman and image size getting bigger with distrobox-create

For rootless podman 3.4.0 and upward, adding this to your `~/.config/containers/storage.conf` file
will improve container creation speed and fix issues with images getting bigger when using
rootless containers.

```
[storage]
driver = "overlay"

[storage.options.overlay]
mount_program = "/usr/bin/fuse-overlayfs"
```

## Container save and restore

To save, export and reuse an already configured container, you can leverage `podman save` or `docker save` and `podman import` or `docker import`
to create snapshots of your environment.

---

To save a container to an image:

with podman:

```
podman container commit -p distrobox_name image_name_you_choose
podman save image_name_you_choose:latest | gzip > image_name_you_choose.tar.gz
```

with docker:

```
docker container commit -p distrobox_name image_name_you_choose
docker save image_name_you_choose:latest | gzip > image_name_you_choose.tar.gz
```

This will create a tar.gz of the container of your choice at that exact moment.

---

Now you can backup that archive or transfer it to another host, and to restore it
just run

```
podman load < image_name_you_choose.tar.gz
```

or

```
docker load < image_name_you_choose.tar.gz
```

And create a new container based on that image:

```
distrobox-create --image image_name_you_choose:latest --name distrobox_name
distrobox-enter --name distrobox_name
```

And you're good to go, now you can reproduce your personal environment everywhere
in simple (and scriptable) steps.

## Check used resources

- You can always check how much space a `distrobox` is taking by using `podman` command:

`podman system df -v` or `docker system df -v`

## Using podman inside a distrobox

If `distrobox` is using `podman` as the container engine, you can use `podman socket` to
control host's podman from inside a `distrobox`, just use:

`podman --remote`

inside the `distrobox` to use it.

It may be necessary to enable the socket on your host system by using:

`systemctl --user enable --now podman.socket`

## Using docker inside a distrobox

You can use `docker` to control host's podman from inside a `distrobox`,
by default if `distrobox` is using docker as a container engine, it will mount the
docker.sock into the container.

So in the container just install `docker`, add yourself to the `docker` group, and
you should be good to go.

## Using init system inside a distrobox

You can use an init system inside the container on supported images. Example of such images are:

- docker.io/almalinux/8-init
- registry.access.redhat.com/ubi7/ubi-init
- registry.access.redhat.com/ubi8/ubi-init

You can use such feature using:

`distrobox create -i docker.io/almalinux/8-init --init --name test`

Note however that in this mode, you'll not be able to access host's processes from within the container.

Example use:

```shell
~$ distrobox create -i docker.io/almalinux/8-init --init --name test

user@test:~$ sudo systemctl enable --now sshd

user@test:~$ sudo systemctl status sshd
    ● sshd.service - OpenSSH server daemon
       Loaded: loaded (/usr/lib/systemd/system/sshd.service; enabled; vendor preset: enabled)
       Active: active (running) since Fri 2022-01-28 22:54:50 CET; 17s ago
         Docs: man:sshd(8)
               man:sshd_config(5)
     Main PID: 291 (sshd)

```

## Using distrobox as main cli

In case you want (like me) to use your container as the main CLI environment, it comes
handy to use `gnome-terminal` profiles to create a dedicated setup for it:

![Screenshot from 2021-12-19 22-29-08](https://user-images.githubusercontent.com/598882/146691460-b8a5bb0a-a83d-4e32-abd0-4a0ff9f50eb7.png)

Personally, I just bind `Ctrl-Alt-T` to the Distrobox profile and `Super+Enter` to the Host profile.

For other terminals, there are similar features (profiles) or  you can set up a dedicated shortcut to
launch a terminal directly in the distrobox
