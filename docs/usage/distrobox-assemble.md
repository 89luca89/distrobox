<!-- markdownlint-disable MD010 MD036 MD046 -->
# NAME

	distrobox assemble
	distrobox-assemble

# DESCRIPTION

distrobox-assemble takes care of creating or destroying containers in batches,
based on a manifest file.
The manifest file by default is `./distrobox.ini`, but can be specified using the
`--file` flag.

# SYNOPSIS

**distrobox assemble**

	--file:			path or URL to the distrobox manifest/ini file
	--name/-n:		run against a single entry in the manifest/ini file
	--replace/-R:		replace already existing distroboxes with matching names
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

This is an example manifest file to create two containers:

	[ubuntu]
	additional_packages="git vim tmux nodejs"
	image=ubuntu:latest
	init=false
	nvidia=false
	pull=true
	root=false
	replace=true
	start_now=false

	# You can add comments using this #
	[arch] # also inline comments are supported
	additional_packages="git vim tmux nodejs"
	home=/tmp/home
	image=archlinux:latest
	init=false
	start_now=true
	init_hooks="touch /init-normal"
	nvidia=true
	pre_init_hooks="touch /pre-init"
	pull=true
	root=false
	replace=false
	volume="/tmp/test:/run/a /tmp/test:/run/b"

**Create**

We can bring them up simply using

	distrobox assemble create

If the file is called `distrobox.ini` and is in the same directory you're launching
the command, no further arguments are needed.
You can specify a custom path for the file using

	distrobox assemble create --file /my/custom/path.ini

Or even specify a remote file, by using an URL:

	distrobox-assemble create --file https://raw.githubusercontent.com/89luca89/dotfiles/master/distrobox.ini

**Replace**

By default, `distrobox assemble` will replace a container only if `replace=true`
is specified in the manifest file.

In the example of the manifest above, the ubuntu container will always be replaced
when running `distrobox assemble create`, while the arch container will not.

To force a replace for all containers in a manifest use the `--replace` flag

	distrobox assemble create --replace [--file my/custom/path.ini]

**Remove**

We can bring down all the containers in a manifest file by simply doing

	distrobox assemble rm

Or using a custom path for the ini file

	distrobox assemble rm --file my/custom/path.ini

**Test**

You can always test what distrobox **would do** by using the `--dry-run` flag.
This command will only print what commands distrobox would do without actually
running them.

**Clone**

**Disclaimer**: You need to start the container once to ensure it is fully initialized and created
before cloning it. The container being copied must also be stopped before the cloning process can proceed.

**Available options**

This is a list of available options with the corresponding type:

Types legend:

- bool: true or false
- string: a single string, for example `home="/home/luca-linux/dbox"`
- string_list: multiple strings, for example `additional_packages="htop vim git"`. Note that `string_list` can be
declared multiple times to be compounded:

	```ini
	[ubuntu]
	image=ubuntu:latest
	additional_packages="git vim tmux nodejs"
	additional_packages="htop iftop iotop"
	additional_packages="zsh fish"
	```

| Flag Name | Type | |
| - | - | - |
| additional_flags | string_list | Additional flags to pass to the container manager |
| additional_packages | string_list | Additional packages to install inside the container |
| home | string | Which home directory should the container use |
| image | string | Which image should the container use, look [here](../compatibility.md) for a list |
| clone | string | Name of the Distrobox container to use as the base for a new container (the container must be stopped). |
| init_hooks | string_list | Commands to run inside the container, after the packages setup |
| pre_init_hooks | string_list | Commands to run inside the container, before the packages setup |
| volume | string_list | Additional volumes to mount inside the containers |
| exported_apps | string_list | App names or desktopfile paths to export |
| exported_bins | string_list | Binaries to export |
| exported_bins_path | string | Optional path where to export binaries (default: $HOME/.local/bin) |
| entry | bool | Generate an entry for the container in the app list (default: false) |
| start_now | bool | Start the container immediately (default: false) |
| init | bool | Specify if this is an initful container (default: false) |
| nvidia | bool | Specify if you want to enable NVidia drivers integration (default: false) |
| pull | bool | Specify if you want to pull the image every time (default: false) |
| root | bool | Specify if the container is rootful (default: false) |
| unshare_groups | bool | Specify if the container should unshare users additional groups (default: false) |
| unshare_ipc | bool | Specify if the container should unshare the ipc namespace (default: false) |
| unshare_netns | bool | Specify if the container should unshare the network namespace (default: false) |
| unshare_process | bool | Specify if the container should unshare the process (pid) namespace (default: false) |
| unshare_devsys | bool | Specify if the container should unshare /dev (default: false) |
| unshare_all | bool | Specify if the container should unshare all the previous options (default: false) |

For further explanation of each of the option in the list, take a look at the [distrobox create usage](distrobox-create.md#synopsis),
each option corresponds to one of the `create` flags.

**Advanced example**

	[tumbleweed_distrobox]
	image=registry.opensuse.org/opensuse/distrobox
	pull=true
	additional_packages="acpi bash-completion findutils iproute iputils sensors inotify-tools unzip"
	additional_packages="net-tools nmap openssl procps psmisc rsync man tig tmux tree vim htop xclip yt-dlp"
	additional_packages="git git-credential-libsecret"
	additional_packages="patterns-devel-base-devel_basis"
	additional_packages="ShellCheck ansible-lint clang clang-tools codespell ctags desktop-file-utils gcc golang jq python3"
	additional_packages="python3-bashate python3-flake8 python3-mypy python3-pipx python3-pycodestyle python3-pyflakes python3-pylint python3-python-lsp-server python3-rstcheck python3-yapf python3-yamllint rustup shfmt"
	additional_packages="kubernetes-client helm"
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest;
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install github.com/onsi/ginkgo/v2/ginkgo@latest;
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install golang.org/x/tools/cmd/goimports@latest;
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install golang.org/x/tools/gopls@latest;
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install sigs.k8s.io/kind@latest;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/conmon;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/crun;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/docker;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/docker-compose;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/flatpak;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/podman;
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/xdg-open;
	exported_apps="htop"
	exported_bins="/usr/bin/htop /usr/bin/git"
	exported_bins_path="~/.local/bin"

**Clone example**

	[ubuntu]
	additional_packages="git vim tmux"
	image=ubuntu:latest
	init=false
	nvidia=false
	pull=true
	root=false
	replace=true
	start_now=true
	
	[deno_ubuntu]
	clone=ubuntu
	init=false
	nvidia=false
	pull=true
	root=false
	replace=true
	start_now=true
	pre_init_hooks=curl -fsSL https://deno.land/install.sh | sh;
	
	[bun_ubuntu]
	clone=ubuntu
	init=false
	nvidia=false
	pull=true
	root=false
	replace=true
	start_now=true
	pre_init_hooks=curl -fsSL https://bun.sh/install | bash;
