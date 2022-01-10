# Gentoo as a guest

### Image
You need to build your own image. The official resource is [here](https://github.com/gentoo/gentoo-docker-images#using-the-portage-container-in-a-multi-stage-build) but here is a simple Dockerfile:
``` Dockerfile
FROM registry.hub.docker.com/gentoo/portage:latest

FROM registry.hub.docker.com/gentoo/stage3:systemd

COPY --from=portage /var/db/repos/gentoo /var/db/repos/gentoo
```
