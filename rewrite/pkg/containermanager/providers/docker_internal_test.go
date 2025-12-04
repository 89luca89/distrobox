package providers

import (
	"strings"
	"testing"

	"github.com/89luca89/distrobox/internal/userenv"
)

func TestDocker_makeCreateCommand(t *testing.T) {
	docker := NewDocker(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	containerAdditionalFlags := []string{}
	containerAdditionalPackages := []string{}
	containerAdditionalVolumes := []string{
		"/path/to/my-volume:/var/local/my-volume:ro",
	}

	cmd := docker.makeCreateCommand(
		"my-container",                // containerName
		"my-image",                    // containerImage
		containerAdditionalFlags,      // containerAdditionalFlags
		"my-hostname",                 // containerHostname
		containerAdditionalPackages,   // containerAdditionalPackages
		containerAdditionalVolumes,    // containerAdditionalVolumes
		"",                            // containerUserCustomHome
		"",                            // containerPlatform
		false,                         // nopasswd
		true,                          // init
		"",                            // containerPreInitHook
		"",                            // containerInitHook
		false,                         // nvidia
		false,                         // unshareDevsys
		false,                         // unshareGroups
		false,                         // unshareIPC
		false,                         // unshareNetNS
		false,                         // unshareProcess
		userEnv,                       // userEnv
		"/path/to/distrobox-init",     // distroboxInitPath
		"/path/to/distrobox-export",   // distroboxExportPath
		"/path/to/distrobox-hostexec", // distroboxHostexecPath
	)

	expected := oneline(`
create --hostname my-hostname --name my-container --privileged --security-opt label=disable --security-opt
 apparmor=unconfined --pids-limit=-1 --user root:root --ipc host --network host --pid host --label manager=distrobox
 --label distrobox.unshare_groups=0 --env SHELL=sh --env HOME=/home/user --env container=docker
 --env TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo --env CONTAINER_ID=my-container
 --volume /tmp:/tmp:rslave --volume /path/to/distrobox-export:/usr/bin/distrobox-export:ro
 --volume /path/to/distrobox-hostexec:/usr/bin/distrobox-host-exec:ro --volume /home/user:/home/user:rslave
 --volume /:/run/host/:rslave --volume /dev:/dev:rslave --volume /sys:/sys:rslave --cgroupns host
 --stop-signal SIGRTMIN+3 --mount type=tmpfs,destination=/run --mount type=tmpfs,destination=/run/lock
 --mount type=tmpfs,destination=/var/lib/journal --volume /dev/pts --volume /dev/null:/dev/ptmx
 --volume /sys/fs/selinux --volume /var/log/journal --volume /etc/hosts:/etc/hosts:ro
 --volume /etc/resolv.conf:/etc/resolv.conf:ro --volume /path/to/my-volume:/var/local/my-volume:ro
 --volume /path/to/distrobox-init:/usr/bin/entrypoint:ro --entrypoint /usr/bin/entrypoint my-image --verbose
 --name my-container --user 1000 --group 1000 --home /home/user --init "1" --nvidia "0" --pre-init-hooks
 --additional-packages  -- ''
`)

	got := oneline(strings.Join(cmd, " "))

	if got != expected {
		t.Errorf("Expected command:\n'%s'\nGot:\n'%s'", expected, got)
	}
}

// oneline is a helper function that removes newlines and trims spaces from a string.
func oneline(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "  ", " "))
}
