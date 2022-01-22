### Enter the distrobox

distrobox-enter takes care of entering the container with the name specified.
Default command executed is your SHELL, but you can specify different shells or
entire commands to execute.
If using it inside a script, an application, or a service, you can specify the
--headless mode to disable tty and interactivity.

Usage:

	distrobox-enter --name fedora-toolbox-35 -- bash -l
	distrobox-enter my-alpine-container -- sh -l
	distrobox-enter --additional-flags "--preserve-fds" --name test -- bash -l
	distrobox-enter --additional-flags "--env MY_VAR=value" --name test -- bash -l
	MY_VAR=value distrobox-enter --additional-flags "--preserve-fds" --name test -- bash -l

Supported environment variables:

	DBX_CONTAINER_NAME

Options:

	--name/-n:		name for the distrobox						default: fedora-toolbox-35
	--/-e:			end arguments execute the rest as command to execute at login	default: bash -l
	--headless/-H:		do not instantiate a tty
	--additional-flags/-a:	additional flags to pass to the container manager command
	--help/-h:		show this message
	--verbose/-v:		show more verbosity
	--version/-V:		show version

This is used to enter the distrobox itself. Personally, I just create multiple profiles in my `gnome-terminal` to have multiple distros accessible.


The `--additional-flags` or `-a` is useful to modify default command when executing in the container.
For example:

	distrobox enter -n dev-arch --additional-flags "--env my_var=test" -- printenv &| grep my_var
	my_var=test

This is possible also using normal env variables:

	my_var=test distrobox enter -n dev-arch --additional-flags -- printenv &| grep my_var
	my_var=test
