# distrobox-enter

## DESCRIPTION

distrobox-enter takes care of entering the container with the name
specified. Default command executed is your SHELL, but you can specify
different shells or entire commands to execute. If using it inside a
script, an application, or a service, you can specify the
**\--headless** mode to disable tty and interactivity.

Usage: distrobox-enter **\--name** fedora-toolbox-35 **\--** bash **-l**

## OPTIONS

**\--name**/-n:

:   name for the distrobox default: fedora-toolbox-35

**\--**/-e:

:   end arguments execute the rest as command to execute at login
    default: bash **-l**

**\--headless**/-H:

:   do not instantiate a tty

**\--help**/-h:

:   show this message

**\--verbose**/-v:

:   show more verbosity

**\--version**/-V:

:   show version

## SEE ALSO

GitHub repository is located at https://github.com/89luca89/distrobox
