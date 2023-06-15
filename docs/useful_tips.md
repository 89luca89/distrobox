- [Distrobox](README.md)
  - [Launch a distrobox from you applications list](#launch-a-distrobox-from-you-applications-list)
  - [Create a distrobox with a custom HOME directory](#create-a-distrobox-with-a-custom-home-directory)
  - [Mount additional volumes in a distrobox](#mount-additional-volumes-in-a-distrobox)
  - [Use a different shell than the host](#use-a-different-shell-than-the-host)
  - [Run the container with real root](#run-the-container-with-real-root)
  - [Run Debian/Ubuntu container behind proxy](#run-debianubuntu-container-behind-proxy)
  - [Using a command other than sudo to run a rootful container](#using-a-command-other-than-sudo-to-run-a-rootful-container)
  - [Duplicate an existing distrobox](#duplicate-an-existing-distrobox)
  - [Export to the host](#export-to-the-host)
  - [Execute commands on the host](#execute-commands-on-the-host)
  - [Enable SSH X-Forwarding when SSH-ing in a distrobox](#enable-ssh-x-forwarding-when-ssh-ing-in-a-distrobox)
  - [Using podman or docker inside a distrobox](#using-podman-or-docker-inside-a-distrobox)
  - [Using init system inside a distrobox](#using-init-system-inside-a-distrobox)
  - [Using distrobox as main cli](#using-distrobox-as-main-cli)
  - [Using a different architecture](#using-a-different-architecture)
  - [Using the GPU inside the container](#using-the-gpu-inside-the-container)
  - [Using nvidia-container-toolkit](#using-nvidia-container-toolkit)
  - [Slow creation on podman and image size getting bigger with distrobox create](#slow-creation-on-podman-and-image-size-getting-bigger-with-distrobox-create)
  - [Container save and restore](#container-save-and-restore)
  - [Check used resources](#check-used-resources)
  - [Pre-installing additional package repositories](#pre-installing-additional-package-repositories)
  - [Apply resource limitation on the fly](#apply-resource-limitation-on-the-fly)

---

# Useful tips

## Launch a distrobox from you applications list

Starting from distrobox 1.4.0, containers created will automatically generate a desktop entry.
For containers generated with older versions, you can use:

`distrobox generate-entry you-container-name`

To delete it:

`distrobox generate-entry you-container-name --delete`

## Create a distrobox with a custom HOME directory

`distrobox create` supports the use of the `--home` flag, as specified in the
usage [HERE](./usage/distrobox-create.md)

Simply use:

`distrobox create --name test --image your-chosen-image:tag --home /your/custom/home`

## Mount additional volumes in a distrobox

`distrobox create` supports the use of the `--volume` flag, as specified in the
usage [HERE](./usage/distrobox-create.md)

Simply use:

`distrobox create --name test --image your-chosen-image:tag --volume /your/custom/volume/path`

## Use a different shell than the host

From version 1.4.0, `distrobox enter` will execute the login shell of the container's user
by default. So, just change the default shell in the container using:

`chsh -s /bin/shell-to-use`

exit and log back in the container.

For version older than 1.4.0, distrobox will pick up the shell from the host and use it inside the container.
If you want a different one you can use:

`SHELL=/bin/zsh distrobox create -n test`
`SHELL=/bin/zsh distrobox enter test`

## Run the container with real root

When using podman, distrobox will prefer to use rootless containers. In this mode the `root`
user inside the container is **not** the real `root` user of the host. But it still has
the same privileges as your normal `$USER`.

But what if you really really need those `root` privileges even inside the container?

Running `sudo distrobox` is not supported, instead, it is better to simply use normal
command with the `--root` or `-r` flag, so that distrobox can still integrate better
with your `$USER`.

```console
:~$ distrobox create --name test --image your-chosen-image:tag --root
```

Another use case, what if you want or need to run distrobox with the root user, in a login
shell?

Before the 1.4.3 release, it wasn't possible. We couldn't make a distinction between someone
running distrobox vi `sudo` from someone logged in as the root user in a shell. Now things are
as easy as it would be if you were creating a rootless container:

```console
:~# distrobox create --name your-container --pull --image your-chosen-image:tag`
```

And:

```console
:~# distrobox enter your-container`
```

We trust you already know the implications of running distrobox, as well as anything else,
with the root user and that with great power comes great responsibilities.

## Run Debian/Ubuntu container behind proxy

It might be that you're trying to set-up your distrobox, but you're stuck behind a proxy.
A simple solution can be crafted using `pre-init-hooks`

```console
proxy=http://my_proxy.domain.example:3128
t="echo 'Acquire::http::Proxy \\\""${proxy}"\\\";' > /etc/apt/apt.conf.d/proxy.conf; echo 'Acquire::https::Proxy \\\""${proxy}"\\\";' >> /etc/apt/apt.conf.d/proxy.conf;"
http_proxy="${proxy}" distrobox create --image debian --name deb --pre-init-hooks "${t}"
```

This way, we're configuring `apt` before using it.

## Using a command other than sudo to run a rootful container

When using the `--root` option with Distrobox, internally, it uses `sudo` to be able to
interact with the rootful container through podman/docker, which will prompt for a valid
root password on the terminal. However, some users might prefer to use a command other
than `sudo` in order to authenticate as root; for example, `pkexec` could be used to
display a graphical authentication prompt. If you need this, make sure to specify
the desired command through the `DBX_SUDO_PROGRAM` environment variable
(supported by most `distrobox` subcommands), alongside `--root`. Sample usage:

`DBX_SUDO_PROGRAM="pkexec" distrobox create --name test --image your-chosen-image:tag --root`

Additionally, you may also have any further distrobox commands use `pkexec` (for example)
for rootful containers by appending the line `distrobox_sudo_program="pkexec"`
(replace `pkexec` with the desired program) to one of the config file paths that
distrobox supports; for example, to '~/.distroboxrc'.

It is also worth noting that, if your sudo program does not have persistence
(i.e., cooldown before asking for the root password again after a successful authentication)
configured, then you may have to enter the root password multiple times, as distrobox
calls multiple podman/docker commands under the hood. In order to avoid this, it is
recommended to either configure your sudo program to be persistent, or, if that's
not feasible, use `sudo` whenever possible (which has persistence enabled by default).

However, if you'd like to have a graphical authentication prompt, but would also like
to benefit from `sudo`'s persistence (to avoid prompting for a password multiple times in a row),
you may specify `sudo --askpass` as the sudo program.
The `--askpass` option makes sudo launch the program in the path (or name, if it is in `$PATH`)
specified by the `SUDO_ASKPASS` environment variable, and uses its output (to stdout)
as the password input to authenticate as root. If unsuccessful, it launches the program again,
until either it outputs the correct password, the user cancels the operation, or
a limit of amount of authentication attempts is reached.

So, for example, assume you'd like to use `zenity --password` to prompt for the sudo password.
You may save a script, e.g. `my-password-prompt`, to somewhere in your machine - say,
to `~/.local/bin/my-password-prompt` - with the following contents:

```sh
#!/bin/sh
zenity --password
```

Make it executable using, for example, `chmod` (in the example, by running `chmod +x ~/.local/bin/my-password-prompt` -
replace with the path to your script). Afterwards, make sure `SUDO_ASKPASS` is set to your newly-created script's path,
and also ensure `DBX_SUDO_PROGRAM` is set to `sudo --askpass`, and you should be good to go. For example,
running the below command should only prompt the root authentication GUI once throughout the whole process:

`SUDO_ASKPASS="$HOME/.local/bin/my-password-prompt" DBX_SUDO_PROGRAM="sudo --askpass" distrobox-ephemeral -r`

You may make these options persist by specifying those environment variables in your shell's rc file (such as `~/.bashrc`).
Note that this will also work if `distrobox_sudo_program="sudo --askpass"` is specified in one of distrobox's config files
(such as `~/.distroboxrc`), alongside `export SUDO_ASKPASS="/path/to/password/prompt/program"` (for example - however, this
last line is usually better suited to your shell's rc file).

## Duplicate an existing distrobox

It can be useful to just duplicate an already set up environment, to do this,
`distrobox create` supports the use of the
`--clone` flag, as specified in the usage [HERE](./usage/distrobox-create.md)

Simply use:

`distrobox create --name test --clone name-of-distrobox-to-clone`

## Export to the host

Distrobox supports exporting to the host either binaries, applications or systemd
services. [Head over the usage page to have an explanation and examples.](usage/distrobox-export.md)

## Execute commands on the host

You can check this little post about [executing commands on the host.](posts/execute_commands_on_host.md)

## Enable SSH X-Forwarding when SSH-ing in a distrobox

SSH X-forwarding by default will not work because the container hostname is
different from the host's one.
You can create a distrobox with will have the same hostname as the host by
creating it with the following init-hook:

```sh
distrobox create --name test --image your-chosen-image:tag \
                  --init-hooks '"$(uname -n)" > /etc/hostname'`
```

This will ensure SSH X-Forwarding will work when SSH-ing inside the distrobox:

`ssh -X myhost distrobox enter test -- xclock`

## Using podman or docker inside a distrobox

You can easily control host's instance of docker or podman, using `distrobox-host-exec`
You can use:

```console
sudo ln -s /usr/bin/distrobox-host-exec /usr/local/bin/podman
```

or

```console
sudo ln -s /usr/bin/distrobox-host-exec /usr/local/bin/docker
```

This will create a `podman` or `docker` command inside the distrobox that will
transparently execute the command on the host.

## Using init system inside a distrobox

You can use an init system inside the container. You can either use supported
pre-created images, or have to add additional packages.

Example of such images are:

- docker.io/almalinux/8-init
- registry.access.redhat.com/ubi7/ubi-init
- registry.access.redhat.com/ubi8/ubi-init
- registry.access.redhat.com/ubi9/ubi-init
- registry.opensuse.org/opensuse/leap:latest
- registry.opensuse.org/opensuse/tumbleweed:latest

You can use such feature using:

`distrobox create -i docker.io/almalinux/8-init --init --name test`

If you want to use a non-pre-create image, you'll need to add the additional package:

`distrobox create -i debian:unstable --init --additional-packages "systemd" -n test`
`distrobox create -i ubuntu:18.04 --init --additional-packages "systemd" -n test`
`distrobox create -i registry.fedoraproject.org/fedora:38 --init --additional-packages "systemd" -n test`

Note however that in this mode, you'll not be able to access host's processes
from within the container.

Example use:

```shell
~$ distrobox create -i docker.io/almalinux/8-init --init --name test

user@test:~$ sudo systemctl enable --now sshd

user@test:~$ sudo systemctl status sshd
    ● sshd.service - OpenSSH server daemon
       Loaded: loaded (sshd.service; enabled; vendor preset: enabled)
       Active: active (running) since Fri 2022-01-28 22:54:50 CET; 17s ago
         Docs: man:sshd(8)
               man:sshd_config(5)
     Main PID: 291 (sshd)
```

## Using distrobox as main cli

In case you want (like me) to use your container as the main CLI environment,
it comes handy to use `gnome-terminal` profiles to create a dedicated setup for it:

![Screenshot from 2021-12-19 22-29-08](https://user-images.githubusercontent.com/598882/146691460-b8a5bb0a-a83d-4e32-abd0-4a0ff9f50eb7.png)

Personally, I just bind `Ctrl-Alt-T` to the Distrobox profile and `Super+Enter`
to the Host profile.

For other terminals, there are similar features (profiles) or  you can set up a
dedicated shortcut to launch a terminal directly in the distrobox

## Using a different architecture

In case you want to run a container with a different architecture from your host,
you can leverage the use of `qemu` and support from podman/docker.

Install on your host the following dependencies:

- qemu
- qemu-user-static
- binfmt-support

Then you can easily run the image you like:

```console
~$ uname -m
x86_64
~$ distrobox create -i aarch64/fedora -n fedora-arm64
~$ distrobox enter fedora-arm64
...
user@fedora-arm64:~$ uname -m
aarch64
```

![image](https://user-images.githubusercontent.com/598882/170837120-9170a9fa-6153-4684-a435-d60a0136b563.png)

## Using the GPU inside the container

For Intel and AMD Gpus, the support is backed in, as the containers will install
their latest available mesa/dri drivers.

For NVidia, you can use the `--nvidia` flag during create, see [distrobox-create](./usage/distrobox-create.md)
documentation to discover how to use it.

```console
~$ distrobox create --nvidia --name ubuntu-nvidia --image ubuntu:latest
```

### Using nvidia-container-toolkit

Alternatively from the `--nvidia` flag, you can use NVidia's own [nvidia-container-toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/user-guide.html).
After following the [official guide to set nvidia-ctk up](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/user-guide.html)
you can use it from distrobox doing:

```console
distrobox create --name example-nvidia-toolkit --additional-flags "--runtime=nvidia -e NVIDIA_VISIBLE_DEVICES=all -e NVIDIA_DRIVER_CAPABILITIES=all" --image nvidia/cuda
```

## Slow creation on podman and image size getting bigger with distrobox create

For rootless podman 3.4.0 and upward, adding this to your `~/.config/containers/storage.conf`
file will improve container creation speed and fix issues with images getting
bigger when using rootless containers.

```conf
[storage]
driver = "overlay"

[storage.options.overlay]
mount_program = "/usr/bin/fuse-overlayfs"
```

Note that this is necessary only on Kernel version older than `5.11` .
From version `5.11` onwards native `overlayfs` is supported and reports noticeable
gains in performance as explained [HERE](https://www.redhat.com/sysadmin/podman-rootless-overlay)

## Container save and restore

To save, export and reuse an already configured container, you can leverage
`podman save` or `docker save` and `podman import` or `docker import` to
create snapshots of your environment.

---

To save a container to an image:

with podman:

```sh
podman container commit -p distrobox_name image_name_you_choose
podman save image_name_you_choose:latest | gzip > image_name_you_choose.tar.gz
```

with docker:

```sh
docker container commit -p distrobox_name image_name_you_choose
docker save image_name_you_choose:latest | gzip > image_name_you_choose.tar.gz
```

This will create a tar.gz of the container of your choice at that exact moment.

---

Now you can backup that archive or transfer it to another host, and to restore it
just run

```sh
podman load < image_name_you_choose.tar.gz
```

or

```sh
docker load < image_name_you_choose.tar.gz
```

And create a new container based on that image:

```sh
distrobox create --image image_name_you_choose:latest --name distrobox_name
distrobox enter --name distrobox_name
```

And you're good to go, now you can reproduce your personal environment everywhere
in simple (and scriptable) steps.

## Check used resources

- You can always check how much space a `distrobox` is taking by using `podman` command:

`podman system df -v` or `docker system df -v`

## Pre-installing additional package repositories

On Red Hat Enterprise Linux and its derivatives, the amount of packages in the
base repositories is limited, and additional packages need to be brought in by
enabling additional repositories such as [EPEL](https://docs.fedoraproject.org/en-US/epel/).

You can use `--init-hooks` to automate this, but this does not solve the
issue for package installations done during initialization itself, e.g. if
the shell you use on the host is not available in the default repos (e.g.
`fish`).

Use the pre-initialization hooks for this:

```shell
distrobox create -i docker.io/almalinux/8-init --init --name test --pre-init-hooks "dnf -y install dnf-plugins-core && dnf config-manager --enable powertools && dnf -y install epel-release"
```

```shell
distrobox create -i docker.io/library/almalinux:9 -n alma9 --pre-init-hooks "dnf -y install dnf-plugins-core && dnf config-manager --enable crb && dnf -y install epel-release"
```

```shell
distrobox create -i quay.io/centos/centos:stream9 c9s --pre-init-hooks "dnf -y install dnf-plugins-core && dnf config-manager --enable crb && dnf -y install epel-next-release"
```

## Apply resource limitation on the fly

Podman has `--cpuset-cpus` and `--memory` flags to apply limitation on how much resources a container can use. However,
these flags only work during container creation (`podman create` / `podman run`) and not after it's created
(`podman exec`, which is used by Distrobox to execute commands inside of container), which means changing resource
limitation requires recreation of a container.

Nonetheless you can still apply resource limitation using systemd's resource control functionality. It's not recommended
to pass resource limitation arguments (e.g. `--cpuset-cpus` and `--memory`) to `distrobox create --additional-flags`
as systemd already provides much more flexible resource control functionality.

To list all distroboxes and their full IDs:

```bash
podman ps --all --no-trunc --format "{{.Names}} {{.ID}} {{.Labels}}" | grep "manager:distrobox" | cut -d " " -f1,2 | column -t
```

- Removing `--all` flag will cause the output to only contain currently running distroboxes

To check your container status with `systemctl`:

```bash
systemctl --user status libpod-$UUID.scope
```

- Your distrobox needs to be running for its scope to present (e.g. `distrobox enter` before running this command)
- Replace `$UUID` with your container's real full ID
- To make things easier when tweaking properties, optionally set a environment variable for the current shell:

  bash/zsh:

  ```bash
  UUID=XXXXXXXXX
  ```
  
  fish:

  ```fish
  set UUID XXXXXXXXX
  ```

Everything provided by `systemd.resource-control` could be applied to your distrobox. For example:

To make your distrobox only run on CPU0 and CPU1:

```bash
systemctl --user set-property libpod-$UUID.scope AllowedCPUs=0,1
```

To hard throttle your distrobox to not use above 20% of CPU:

```bash
systemctl --user set-property libpod-$UUID.scope CPUQuota=20%
```

To limit your distrobox's maximum amount of memory:

```bash
systemctl --user set-property libpod-$UUID.scope MemoryMax=2G
```

To give your distrobox less IO bandwidth when IO is overloaded:

```bash
systemctl --user set-property libpod-$UUID.scope IOWeight=1
```

- `IOWeight` accepts value from `1` to `10000`, higher means more bandwidth.

To see all applicable properties:

```bash
man systemd.resource-control
```

Changes are transient, meaning you lose the resource limitation properties when distrobox is stopped and restarted.

To make certain changes persistent, first check the currently active properties:

```bash
systemctl --user status libpod-$UUID.scope
```

Look for the `Drop-In` lines. Something like this should be shown:

```console
    Drop-In: /run/user/1000/systemd/transient/libpod-45ae38d61c9a636230b2ba89ea07792d662e01cd9ee38d04feb0a994b039a271.scope.d
             └─50-AllowedCPUs.conf
```

Move the transient overrides to persistent overrides:

```bash
mkdir -p ~/.config/systemd/user/libpod-$UUID.scope.d
mv --target-directory="$HOME/.config/systemd/user/libpod-$UUID.scope.d" \
  "/run/user/$(id -u)/systemd/transient/libpod-$UUID.scope.d/50-AllowedCPUs.conf"
```

- Replace `$(id -u)` with your real user id if it did not get expanded properly.
- `50-AllowedCPUs.conf` is only an example. Replace it with something you want to keep persistently.

Then reload systemd daemon to apply the changes:

```bash
systemctl --user daemon-reload
```
