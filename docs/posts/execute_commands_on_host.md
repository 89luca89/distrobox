- [Distrobox](../README.md)
  - [Execute a command on the host](#execute-a-command-on-the-host)
    - [With distrobox-host-exec](#with-distrobox-host-exec)
    - [Using symlinks](#using-symlinks)
  - [Integrate host with container seamlessly](#integrate-host-with-container-seamlessly)
    - [bash or zsh](#bash-or-zsh)
    - [fish](#fish)

---

# Execute a command on the host

It may be needed to execute commands back on the host. Be it the filemanager, an
archive manager, a container manager and so on.

Here are a couple of solutions.

## With distrobox-host-exec

distrobox offers the `distrobox-host-exec` helper, that can be used exactly for this.

See [distrobox-host-exec](../usage/distrobox-host-exec.md).

```console
user@fedora-distrobox:~$ which podman
/usr/bin/which: no podman in [...]
user@fedora-distrobox:~$ distrobox-host-exec podman version # <-- this is executed on host.
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

## Using symlinks

Another way to execute commands on the host, is to create executables symlinking `distrobox-host-exec`:

```console
user@fedora-distrobox:~$ ln -s /usr/bin/distrobox-host-exec /usr/local/bin/podman
user@fedora-distrobox:~$ ls -l /usr/local/bin/podman
lrwxrwxrwx. 1 root root 51 Jul 11 19:26 /usr/local/bin/podman -> /usr/bin/distrobox-host-exec
user@fedora-distrobox:~$ podman version # <-- this is executed on host. Equivalent to "distrobox-host-exec podman version"
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

# Integrate host with container seamlessly

Another cool trick we can pull, is to use the handy `command_not_found_handle` function
to try and execute missing commands in the container on the host.

## bash or zsh

Place this in your `~/.profile`:

```shell
command_not_found_handle() {
# don't run if not in a container
  if [ ! -e /run/.containerenv ] && [ ! -e /.dockerenv ]; then
    exit 127
  fi
  
  distrobox-host-exec "${@}"
}
if [ -n "${ZSH_VERSION-}" ]; then
  command_not_found_handler() {
    command_not_found_handle "$@"
 }
fi
```

And then, run `source ~/.profile` to reload `.profile` in the current session.

## fish

Place this snippet in a new fish function file (`~/.config/fish/functions/fish_command_not_found.fish`):

```fish
function fish_command_not_found
    # "In a container" check
    if test -e /run/.containerenv -o -e /.dockerenv
        distrobox-host-exec $argv
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
