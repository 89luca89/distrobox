### Create the distrobox

distrobox-create takes care of creating the container with input name and image.
The created container will be tightly integrated with the host, allowing sharing of
the HOME directory of the user, external storage, external usb devices and
graphical apps (X11/Wayland), and audio.

Usage:

	distrobox-create --image registry.fedoraproject.org/fedora-toolbox:35 --name fedora-toolbox-35
	distrobox-create --clone fedora-toolbox-35 --name fedora-toolbox-35-copy
	distrobox-create --image alpine my-alpine-container

You can also use environment variables to specify container name and image

	DBX_NON_INTERACTIVE=1 DBX_CONTAINER_NAME=test-alpine DBX_CONTAINER_IMAGE=alpine distrobox-create

Supported environment variables:

	DBX_NON_INTERACTIVE
	DBX_CONTAINER_NAME
	DBX_CONTAINER_IMAGE

Options:

	--image/-i:		image to use for the container	default: registry.fedoraproject.org/fedora-toolbox:35
	--name/-n:		name for the distrobox		default: fedora-toolbox-35
	--non-interactive/-N:	non-interactive, pull images without asking
	--clone/-c:		name of the distrobox container to use as base for a new container
				this will be useful to either rename an existing distrobox or have multiple copies
				of the same environment.
	--home/-H		select a custom HOME directory for the container. Useful to avoid host's home littering with temp files.
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version
