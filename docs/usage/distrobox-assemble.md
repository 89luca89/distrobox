<!-- markdownlint-disable MD010 MD036 -->
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

	--file:			path to the distrobox manifest/ini file
	--replace/-R:		replace already existing distroboxes with matching names
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version

# EXAMPLES

This is an example manifest file to create two containers:

	[ubuntu]
	additional_packages=git vim tmux nodejs
	image=ubuntu:latest
	init=false
	nvidia=false
	pull=true
	root=false
	replace=true
	start_now=false

	# You can add comments using this #
	[arch] # also inline comments are supported
	additional_packages=git vim tmux nodejs
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
	volume=/tmp/test:/run/a /tmp/test:/run/b

**Create**

We can bring them up simply using

	distrobox assemble create

If the file is called `distrobox.ini` and is in the same directory you're launching
the command, no further arguments are needed.
You can specify a custom path for the file using

	distrobox assemble create --file /my/custom/path.ini

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

**Available options**

This is a list of available options with the corresponding type:

| Flag Name | Type |
| - | - |
| additional_flags | string
| additional_packages | string
| home | string
| image | string
| init_hooks | string
| pre_init_hooks | string
| volume | string
| entry | bool
| start_now | bool
| init | bool
| nvidia | bool
| pull | bool
| root | bool
| unshare_ipc | bool
| unshare_netns | bool

boolean options default to false if not specified.
string options can be broken in multiple declarations additively in order to improve
readability of the file:

	[ubuntu]
	image=ubuntu:latest
	additional_packages=git vim tmux nodejs
	additional_packages=htop iftop iotop
	additional_packages=zsh fish

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
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install github.com/onsi/ginkgo/v2/ginkgo@latest
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install golang.org/x/tools/cmd/goimports@latest
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install golang.org/x/tools/gopls@latest
	init_hooks=GOPATH="${HOME}/.local/share/system-go" GOBIN=/usr/local/bin go install sigs.k8s.io/kind@latest
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/conmon
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/crun
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/docker
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/docker-compose
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/flatpak
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/podman
	init_hooks=ln -sf /usr/bin/distrobox-host-exec /usr/local/bin/xdg-open
