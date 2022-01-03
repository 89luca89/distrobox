- [Distrobox](README.md)
  * [Improve distrobox-enter performance](#improve-distrobox-enter-performance)
  * [Slow creation on podman and image size getting bigger with distrobox-create](#slow-creation-on-podman-and-image-size-getting-bigger-with-distrobox-create)
  * [Container save and restore](#container-save-and-restore)
  * [Check used resources](#check-used-resources)
  * [Using podman inside a distrobox](#using-podman-inside-a-distrobox)
  * [Using docker inside a distrobox](#using-docker-inside-a-distrobox)
  * [Using distrobox as main cli](#using-distrobox-as-main-cli)

# Useful tips

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

## Using distrobox as main cli

In case you want (like me) to use your container as the main CLI environment, it comes
handy to use `gnome-terminal` profiles to create a dedicated setup for it:

![Screenshot from 2021-12-19 22-29-08](https://user-images.githubusercontent.com/598882/146691460-b8a5bb0a-a83d-4e32-abd0-4a0ff9f50eb7.png)

Personally, I just bind `Ctrl-Alt-T` to the Distrobox profile and `Super+Enter` to the Host profile.

For other terminals, there are similar features (profiles) or  you can set up a dedicated shortcut to
launch a terminal directly in the distrobox
