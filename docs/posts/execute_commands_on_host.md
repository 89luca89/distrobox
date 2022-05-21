- [Distrobox](../README.md)
  - [Execute a command on the Host](execute_commands_on_host.md)
    - [With distrobox-host-exec](#distrobox-host-exec)
    - [Manually](#manually)
      - [The easy one](#the-easy-one)
      - [The not so easy one](#the-not-so-easy-one)
  - [Integrate host with container seamlessly](#integrate-host-with-container-seamlessly)

---

# Execute a command on the host

It may be needed to execute commands back on the host. Be it the filemanager, an
archive manager, a container manager and so on.

Here are a couple of solutions.

## With distrobox-host-exec

distrobox offers the `distrobox-host-exec` helper, that can be used exactly for this.

See [distrobox-host-exec](../usage/distrobox-host-exec.md).

## Manually

### The easy one

Install `flatpak-spawn` inside the container, this example is running on a
Fedora Distrobox:

```shell
~$ distrobox create --image fedora:35 --name fedora-distrobox
~$ distrobox enter --name fedora-distrobox
user@fedora-distrobox:~$ sudo dnf install -y flatpak-spawn
```

With `flatpak-swpan` we can easily execute commands on the host using:

```shell
user@fedora-distrobox:~$ flatpak-spawn --host bash -l
~$  # We're back on host!
```

### The not so easy one

Alternatively you may don't have `flatpak-spawn` in the repository of your container,
or simply want an alternative.

We can use `chroot` to enter back into the host, and execute what we need!

Create an executable file with this content:

```shell
#!/bin/sh

result_command="sudo -E chroot --userspec=$(id -u):$(id -g) /run/host/ /usr/bin/env "
for i in $(printenv | grep "=" | grep -Ev ' |"' | grep -Ev "^(_)"); do
 result_command="$result_command $i"
done

exec ${result_command} sh -c " cd ${PWD} && $@"
```

in `~/.local/bin/host-exec` and make it executable with `chmod +x ~/.local/bin/host-exec`

Now we can simply use this to exec stuff back on the host:

```shell
user@fedora-distrobox:~$ host-exec bash -l
~$  # We're back on host!
```

# Integrate host with container seamlessly

Another cool trick we can pull, is to use the handy `command_not_found_handle` function
to try and execute missing commands in the container on the host.

## bash / zsh

Place this in your `~/.profile`:

```shell
command_not_found_handle() {
 # don't run if not in a container
 if [ ! -e /run/.containerenv ] &&
  [ ! -e /.dockerenv ]; then
  exit 127
 fi

 if command -v flatpak-spawn >/dev/null 2>&1; then
  flatpak-spawn --host "${@}"
 elif command -v host-exec >/dev/null 2>&1; then
  host-exec "$@"
 else
  exit 127
 fi
}

if [ -n "${ZSH_VERSION-}" ]; then
 command_not_found_handler() {
  command_not_found_handle "$@"
 }
fi
```

## fish

Place this snippet in a new fish function file (`~/.config/fish/functions/fish_command_not_found.fish`):

```fish
function fish_command_not_found
    # "In a container" check
    if test -e /run/.containerenv -o -e /.dockerenv
        if command -q flatpak-spawn
            flatpak-spawn --host $argv
        else if command -q host-exec
            host-exec "$argv"
        else
            __fish_default_command_not_found_handler $argv
        end
    else
        __fish_default_command_not_found_handler $argv
    end
end
```

And restart your terminal. Now when a command does not exist on your container,
it will be automatically executed back on the host:

```shell
user@fedora-distrobox:~$ which podman
/usr/bin/which: no podman in [...]
user@fedora-distrobox:~$ podman version # <-- this is automatically executed on host.
Client:
Version:      3.4.2
API Version:  3.4.2
Go Version:   go1.16.6
Built:        Thu Jan  1 01:00:00 1970
OS/Arch:      linux/amd64

Server:
Version:      3.4.2
API Version:  3.4.2
Go Version:   go1.16.6
Built:        Thu Jan  1 01:00:00 1970
OS/Arch:      linux/amd64
```

This is also useful to open `code`, `xdg-open`, or `flatpak` from within the container
seamlessly.
