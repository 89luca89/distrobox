- [Distrobox](../README.md)
  - [Run Libvirt using distrobox](run_libvirt_in_distrobox.md)
    - [Prepare the container](#prepare-the-container)
    - [Launch from the container](#launch-from-the-container)
    - [Connect via SSH](#connect-via-ssh)

# Using an immutable distribution

If you are on an immutable distribution (Silverblue/Kionite, MicroOS) chances are that
installing lots and lots of packages on the base system is not advisable.

One way is to use a distrobox for them.

## Prepare the container

To run libvirt/qemu/kvm we need a systemd container and we need a **rootful** container
to be able to use it, see [this tip](../useful_tips.md#using-init-system-inside-a-distrobox)
to have a list of compatible images.
We will use in this example OpenSUSE's dedicated distrobox image:

Assembly file:

```ini
[libvirt]
image=registry.opensuse.org/opensuse/distrobox:latest
pull=true
init=true
root=true
entry=true
start_now=false
unshare_all=true
additional_packages="systemd"
# Basic utilities for terminal use
init_hooks="zypper in -y --no-recommends openssh-server patterns-server-kvm_server patterns-server-kvm_tools qemu-arm qemu-ppc qemu-s390x qemu-extra qemu-linux-user"
init_hooks="systemctl enable sshd.service"
init_hooks="systemctl enable virtqemud.socket virtnetworkd.socket virtstoraged.socket virtnodedevd.socket"
# Add the default user to the libvirt group
init_hooks="usermod -aG libvirt ${USER}"
# Expose container ssh on host
additional_flags="-p 2222:22"
```

Alternatively, command line:

```console
distrobox create --pull --root --init --unshare-all --image registry.opensuse.org/opensuse/distrobox:latest --name libvirtd --additional-flags "-p 2222:22" \
  --init-hooks "zypper in -y --no-recommends openssh-server patterns-server-kvm_server patterns-server-kvm_tools qemu-arm qemu-ppc qemu-s390x qemu-extra qemu-linux-user && systemctl enable sshd.service && systemctl enable virtqemud.socket virtnetworkd.socket virtstoraged.socket virtnodedevd.socket && usermod -aG libvirt $USER"
```

## Launch from the container

You can launch VirtManager from the container, by simply opening your container from the application menu:

![image](https://github.com/89luca89/distrobox/assets/598882/04d06687-51ef-46b6-8443-2d2e4bf7d7a6)

Then launch `virt-manager` from it:

![image](https://github.com/89luca89/distrobox/assets/598882/1faaac45-cdd3-402a-b763-527a0823b2dc)

![image](https://github.com/89luca89/distrobox/assets/598882/23db4141-30e8-4e64-bffb-9aff75c648dd)

You can use it directly as if it were on the host now.

## Connect via SSH

You can alternatively connect from an existing VirtManager

Now you will need to **Add a connection**:

![image](https://user-images.githubusercontent.com/598882/208441337-4dbade85-4c72-4342-b9ee-acd76b9b1675.png)

Then set it like this:

![Screenshot from 2024-02-19 19-50-04](https://github.com/89luca89/distrobox/assets/598882/bff78725-63c9-4da6-9d25-318c58162673)

- Tick the "Use ssh" option
- username: `<your-user-name>`
- hostname: 127.0.0.1:2222

Optionally you can set it to autoconnect.

Now you can simply double click the connection to activate it, you'll be prompted
with your password, insert the same password as the host:

![image](https://github.com/89luca89/distrobox/assets/598882/27bba705-223f-4876-a2fc-b6d102b7130a)

And you should be good to go!

![image](https://user-images.githubusercontent.com/598882/208442009-fe9df606-e6a8-44f9-94c2-1c2bfba4ca15.png)
