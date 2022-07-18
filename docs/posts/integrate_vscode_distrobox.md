- [Distrobox](../README.md)
  - [Integrate VSCode and Distrobox](#integrate-vscode-and-distrobox)
    - [The easy one](#the-easy-one)
    - [The not so easy one](#the-not-so-easy-one)
      - [First step, install it](#first-step-install-it)
      - [Second step, extensions](#second-step-extensions)
      - [Third step, podman wrapper](#third-step-podman-wrapper)
      - [Fourth step, configure the container](#fourth-step-configure-the-container)
    - [Final Result](#final-result)

---

# Integrate VSCode and Distrobox

VScode doesn't need presentations, and it's a powerful tool for development.
You may want to use it, but how to handle the dualism between host and container?

In this experiment we will use [VSCodium](https://vscodium.com/) as an opensource
alternative to VSCode.

Here are a couple of solutions.

## The easy one

Well, you could just install VSCode in your Distrobox of choice, and export it!

For example using an Arch Linux container:

```shell
~$ distrobox create --image archlinux:latest --name arch-distrobox
~$ distrobox enter --name arch-distrobox
user@arch-distrobox:~$
```

Download the deb file
[HERE](https://github.com/VSCodium/vscodium/releases), or in Arch case just install

```shell
user@arch-distrobox:~$ sudo pacman -S code
```

Now that we have installed it, we can export it:

```shell
user@ubuntu-distrobox:~$ distrobox-export --app code
```

And that's really it, you'll have VSCode in your app list, and it will run from
the Distrobox itself, so it will have access to all the software and tools inside
it without problems.

![image](https://user-images.githubusercontent.com/598882/149206335-1a2d0edd-8b2f-437d-aae0-44b9723d2c30.png)
![image](https://user-images.githubusercontent.com/598882/149206414-56bdbc5a-3728-45ef-8dd4-2e168a0d7ccc.png)

## The not so easy one

Alternatively you may want to install VSCode on your host. We will explore how
to integrate VSCode installed via **Flatpak** with Distrobox.

Note that this integration process is inspired by the awesome project [toolbox-vscode](https://github.com/owtaylor/toolbox-vscode)
so many thanks to @owtaylor for the heavy lifting!

### First step install it

```shell
~$ flatpak install --user app/com.visualstudio.code
```

### Second step, extensions

Now we want to install VSCode [Remote Container extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

![image](https://user-images.githubusercontent.com/598882/149207447-76a82e91-dd3f-43fa-8c52-9c2e85ae8fee.png)

### Third step podman wrapper

Being in a Flatpak, we will need access to host's `podman` (or `docker`) to be
able to use the containers. Place this in your `~/.local/bin/podman-host`

```shell
#!/bin/bash
set -x
if [ "$1" == "exec" ]; then
 # Remove 'exec' from $@
 shift
 script='
     result_command="podman exec"
        for i in $(printenv | grep "=" | grep -Ev " |\"" |
            grep -Ev "^(HOST|HOSTNAME|HOME|PATH|SHELL|USER|_)"); do

            result_command=$result_command --env="$i"
     done

        exec ${result_command} "$@"
    '
 exec flatpak-spawn --host sh -c "$script" - "$@"
else
 exec flatpak-spawn --host podman "$@"
fi
```

Open VSCode settings (Ctrl+,) and head to `Remote>Containers>Docker Path` and
set it to the path of `podman-exec`, like in the example

![image](https://user-images.githubusercontent.com/598882/149208525-5ad630c9-fcbc-4ee6-9d77-e50d2c782a56.png)

This will give a way to execute host's container manager from within the
flatpak app.

### Fourth step configure the container

We need not to deploy a configuration for our container. We should create one for
each Distrobox we choose to integrate with VSCode:

```json
{
  "name" : // PUT YOUR DISTROBOX NAME HERE
  "remoteUser": "${localEnv:USER}",
  "settings": {
    "remote.containers.copyGitConfig": false,
    "remote.containers.gitCredentialHelperConfigLocation": "none",
    "terminal.integrated.profiles.linux": {
      "shell": {
        "path": "${localEnv:SHELL}",
        "args": [
          "-l"
        ]
      }
    },
    "terminal.integrated.defaultProfile.linux": "shell"
  },
  "remoteEnv": {
    "COLORTERM": "${localEnv:COLORTERM}",
    "DBUS_SESSION_BUS_ADDRESS": "${localEnv:DBUS_SESSION_BUS_ADDRESS}",
    "DESKTOP_SESSION": "${localEnv:DESKTOP_SESSION}",
    "DISPLAY": "${localEnv:DISPLAY}",
    "LANG": "${localEnv:LANG}",
    "SHELL": "${localEnv:SHELL}",
    "SSH_AUTH_SOCK": "${localEnv:SSH_AUTH_SOCK}",
    "TERM": "${localEnv:TERM}",
    "VTE_VERSION": "${localEnv:VTE_VERSION}",
    "XDG_CURRENT_DESKTOP": "${localEnv:XDG_CURRENT_DESKTOP}",
    "XDG_DATA_DIRS": "${localEnv:XDG_DATA_DIRS}",
    "XDG_MENU_PREFIX": "${localEnv:XDG_MENU_PREFIX}",
    "XDG_RUNTIME_DIR": "${localEnv:XDG_RUNTIME_DIR}",
    "XDG_SESSION_DESKTOP": "${localEnv:XDG_SESSION_DESKTOP}",
    "XDG_SESSION_TYPE": "${localEnv:XDG_SESSION_TYPE}"
  }
}
```

And place it under `${HOME}/.var/app/com.visualstudio.code/config/Code/User/globalStorage/ms-vscode-remote.remote-containers/nameConfigs/your-distrobox-name.json`

## Final Result

After that, we're good to go! Open VSCode and Attach to Remote Container:

![image](https://user-images.githubusercontent.com/598882/149210561-2f1839ae-9a57-42fc-a122-21652588e327.png)

And let's choose our Distrobox

![image](https://user-images.githubusercontent.com/598882/149210690-8bcb9a0d-1dc5-4937-9494-8c6aa6b26fd5.png)

And we're good to go! We have our VSCode remote session inside our Distrobox container!

![image](https://user-images.githubusercontent.com/598882/149210881-749a8146-c69d-4382-bbef-91e4b477b7ba.png)
