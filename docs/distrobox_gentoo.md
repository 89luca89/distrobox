- [Distrobox](README.md)
  - [Useful tips](useful_tips.md)

---

# Gentoo as a distrobox container

## Image

You need to build your own image. The official resource is [here](https://github.com/gentoo/gentoo-docker-images#using-the-portage-container-in-a-multi-stage-build)
but here is a simple Dockerfile:

``` Dockerfile
FROM registry.hub.docker.com/gentoo/portage:latest as portage
FROM registry.hub.docker.com/gentoo/stage3:systemd
COPY --from=portage /var/db/repos/gentoo /var/db/repos/gentoo
```

Build it using either podman or docker:

```shell
podman build . -t gentoo-distrobox
```

or

```shell
docker build . -t gentoo-distrobox
```

and it's ready to be used:

```shell
distrobox create --image localhost/gentoo-distrobox:latest
```
