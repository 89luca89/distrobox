- [Distrobox](../README.md)
  - [Run Libvirt using distrobox](run_libvirt_in_distrobox.md)
    - [Prepare the container](#prepare-the-container)
    - [Connect from the host](#connect-from-the-host)

# Using an immutable distribution

If you are on an immutable distribution (Silverblue/Kionite, MicroOS) chances are that
installing lots and lots of packages on the base system is not advisable.

One way is to use a distrobox for them.

## Prepare the container

To run libvirt/qemu/kvm we need a systemd container and we need a **rootful** container
to be able to use it, see [this tip](../useful_tips.md#using-init-system-inside-a-distrobox)
to have a list of compatible images.
We will use in this example AlmaLinux 8:

```console
:~> distrobox create --root --init --image quay.io/almalinux/8-init:8 --name libvirtd-container
:~> distrobox enter --root libvirtd-container
```

Let it initialize, then we can install all the packages we need:

```console
:~> distrobox enter --root libvirtd-container
:~$ # We're now inside the container
:~$ sudo dnf groupinstall Virtualization Host --allowerasing 
...
:~$ sudo systemctl enable --now libvirtd
```

Now we need to allow host to connect to the guest's libvirt session, we will use
ssh for it:

```console
:~$ # We're now inside the container
:~$ sudo dnf install openssh-server
:-$ echo "ListenAddress 127.0.0.1
Port 2222" | sudo tee -a /etc/ssh/sshd_config
:-$ sudo systemctl enable --now sshd
:-$ sudo systemctl restart sshd
:~$ sudo su -
:~# passwd
```

Now set a password for root user.

## Connect from the host

You can now install VirtManager, you can either use a normal (non root) distrobox, and export the app

Now you will need to **Add a connection**:

![image](https://user-images.githubusercontent.com/598882/208441337-4dbade85-4c72-4342-b9ee-acd76b9b1675.png)

Then set it like this:

![image](https://user-images.githubusercontent.com/598882/208441499-e612868f-d9d1-452c-8bfb-110440e2e891.png)

- Tick the "Use ssh" option
- username: root
- hostname: 127.0.0.1:2222

Optionally you can set it to autoconnect.

Now you can simply double click the connection to activate it, you'll be prompted
with a password, insert the one you used in the `passwd` step previously:

![image](https://user-images.githubusercontent.com/598882/208441932-f561af0b-9c19-45f7-bacc-d690d80b75e1.png)

And you should be good to go!

![image](https://user-images.githubusercontent.com/598882/208442009-fe9df606-e6a8-44f9-94c2-1c2bfba4ca15.png)
