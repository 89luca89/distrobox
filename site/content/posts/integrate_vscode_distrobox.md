+++
title = "Integrate Vscode and Distrobox"
[extra]
toc = true
+++

VScode doesn't need presentations, and it's a powerful tool for development.
You may want to use it, but how to handle the dualism between host and container?

In this experiment we will use [VSCodium](https://vscodium.com/) as an opensource
alternative to VSCode.

Here are a couple of solutions.

## From Distrobox

Well, you could just install VSCode in your Distrobox of choice, and export it!

For example using an Arch Linux container:

```bash
~$ distrobox create --image archlinux:latest --name arch-distrobox
~$ distrobox enter --name arch-distrobox
user@arch-distrobox:~$
```

Download the deb file
[HERE](https://github.com/VSCodium/vscodium/releases), or in Arch case just install

```bash
user@arch-distrobox:~$ sudo pacman -S code
```

Now that we have installed it, we can export it:

```bash
user@ubuntu-distrobox:~$ distrobox-export --app code
```

And that's really it, you'll have VSCode in your app list, and it will run from
the Distrobox itself, so it will have access to all the software and tools inside
it without problems.

![image](https://user-images.githubusercontent.com/598882/149206335-1a2d0edd-8b2f-437d-aae0-44b9723d2c30.png)
![image](https://user-images.githubusercontent.com/598882/149206414-56bdbc5a-3728-45ef-8dd4-2e168a0d7ccc.png#transparent)

## From Flatpak

Alternatively you may want to install VSCode on your host. We will explore how
to integrate VSCode installed via **Flatpak** with Distrobox.

For this one you'll need to use VSCode from Microsoft, and not VSCodium, in order
to have access to the remote containers extension.

### First Step Install It

```bash
~$ flatpak install --user app/com.visualstudio.code 
```

### Second Step, Extensions

Now we want to install VSCode [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

![image](https://user-images.githubusercontent.com/598882/149207447-76a82e91-dd3f-43fa-8c52-9c2e85ae8fee.png#transparent)

### Third Step Podman Wrapper

Being in a Flatpak, we will need access to host's `podman` to be
able to use the containers. Place this in your `~/.local/bin/podman-host`
In case of access to host's `docker` to be
able to use the containers, use `~/.local/bin/docker-host`

For podman:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/extras/podman-host -o ~/.local/bin/podman-host
chmod +x ~/.local/bin/podman-host
```

For docker:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/extras/docker-host -o ~/.local/bin/docker-host
chmod +x ~/.local/bin/docker-host
```

Open VSCode settings (Ctrl+,) and head to `Remote>Containers>Docker Path` and
set it to the path of `/home/<your-user>/.local/bin/podman-host` (or docker-host in case of docker), like in the example

![image](https://user-images.githubusercontent.com/598882/149208525-5ad630c9-fcbc-4ee6-9d77-e50d2c782a56.png#transparent)

This will give a way to execute host's container manager from within the
flatpak app.

**This works for Distrobox both inside and outside a flatpak**
This will act only for containers created with Distrobox, you can still use regular devcontainers
without transparently if needed.

## Final Result

After that, we're good to go! Open VSCode and Attach to Remote Container:

![image](https://user-images.githubusercontent.com/598882/149210561-2f1839ae-9a57-42fc-a122-21652588e327.png)

And let's choose our Distrobox

![image](https://user-images.githubusercontent.com/598882/149210690-8bcb9a0d-1dc5-4937-9494-8c6aa6b26fd5.png)

And we're good to go! We have our VSCode remote session inside our Distrobox container!

![image](https://user-images.githubusercontent.com/598882/149210881-749a8146-c69d-4382-bbef-91e4b477b7ba.png#transparent)

## Open VSCode Directly Attached to Our Distrobox

You may want to instead have a more direct way to launch your VSCode when you're already in your project directory,
in this case you can use `vscode-distrobox` script:

```bash
curl -s https://raw.githubusercontent.com/89luca89/distrobox/main/extras/vscode-distrobox -o ~/.local/bin/vscode-distrobox
chmod +x ~/.local/bin/vscode-distrobox
```

This will make it easy to launch VSCode attached to target distrobox, on a target path:

`vscode-distrobox my-distrobox /path/to/project`
