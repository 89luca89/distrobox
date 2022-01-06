distrobox-export

## DESCRIPTION

distrobox-export takes care of exporting an app a binary or a service
from the container to the host.

The exported app will be easily available in your normal launcher and it
will automatically be launched from the container it is exported from.

The exported services will be available in the host\'s user\'s systemd
session, so systemctl **\--user** status exported_service_name

will show the status of the service exported.

The exported binaries will be exported in the \"\--export-path\" of
choice as a wrapper script that acts naturally both on the host and in
the container. Note that \"\--export-path\" is NOT OPTIONAL, you have to
explicitly set it.

You can specify additional flags to add to the command, for example if
you want to export an electron app, you could add the \"\--foreground\"
flag to the command:

distrobox-export **\--app** atom **\--extra-flags** \"\--foreground\"
distrobox-export **\--bin** */usr/bin/vim* **\--export-path**
\~/.local/bin **\--extra-flags** \"-p\" distrobox-export **\--service**
syncthing **\--extra-flags** \"-allow-newer-config\"

This works for services, binaries, and apps. Extra flags are only used
then the exported app, binary, or service is used from the host, using
them inside the container will not include them.

The option \"\--delete\" will un-export an app, binary, or service.
distrobox-export **\--app** atom **\--delete** distrobox-export
**\--bin** */usr/bin/vim* **\--export-path** \~/.local/bin **\--delete**
distrobox-export **\--service** syncthing **\--delete** distrobox-export
**\--service** nginx **\--delete**

The option \"\--sudo\" will launch the exported item as root inside the
distrobox.

Note you can use **\--app** OR **\--bin** OR **\--service** but not
together. distrobox-export **\--service** nginx **\--sudo**

Usage: distrobox-export **\--app** mpv \[\--extra-flags \"flags\"\]
\[\--delete\] \[\--sudo\] distrobox-export **\--service** syncthing
\[\--extra-flags \"flags\"\] \[\--delete\] \[\--sudo\] distrobox-export
**\--bin** */path/to/bin* **\--export-path** \~/.local/bin
\[\--extra-flags \"flags\"\] \[\--delete\] \[\--sudo\]

## OPTIONS

**\--app**/-a:

:   name of the application to export

**\--bin**/-b:

:   absolute path of the binary to export

**\--service**/-s:

:   name of the service to export

**\--delete**/-d:

:   delete exported application or service

**\--export-path**/-ep:

:   path where to export the binary

**\--extra-flags**/-ef:

:   extra flags to add to the command

**\--sudo**/-S:

:   specify if the exported item should be ran as sudo

**\--help**/-h:

:   show this message

**\--verbose**/-v:

:   show more verbosity

**\--version**/-V:

:   show version

## SEE ALSO

GitHub repository is located at https://github.com/89luca89/distrobox
