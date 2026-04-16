package providers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

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
		"echo 'pre-init-hook'",        // containerPreInitHook
		"echo 'init-hook'",            // containerInitHook
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

	// Build expected string dynamically for paths that depend on host filesystem
	selinuxVolume := ""
	if pathExists("/sys/fs/selinux") {
		selinuxVolume = " --volume /sys/fs/selinux"
	}

	expected := oneline(`
 create
 --hostname my-hostname
 --name my-container
 --privileged
 --security-opt label=disable
 --security-opt apparmor=unconfined
 --pids-limit=-1
 --user root:root
 --ipc host
 --network host
 --pid host
 --label manager=distrobox
 --label distrobox.unshare_groups=0
 --env SHELL=sh
 --env HOME=/home/user
 --env container=docker
 --env TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo
 --env CONTAINER_ID=my-container
 --volume /tmp:/tmp:rslave
 --volume /path/to/distrobox-export:/usr/bin/distrobox-export:ro
 --volume /path/to/distrobox-hostexec:/usr/bin/distrobox-host-exec:ro
 --volume /home/user:/home/user:rslave
 --volume /:/run/host/:rslave
 --volume /dev:/dev:rslave
 --volume /sys:/sys:rslave
 --cgroupns host
 --stop-signal SIGRTMIN+3
 --mount type=tmpfs,destination=/run
 --mount type=tmpfs,destination=/run/lock
 --mount type=tmpfs,destination=/var/lib/journal
 --volume /dev/pts
 --volume /dev/null:/dev/ptmx
 ` + selinuxVolume + `
 --volume /var/log/journal
 --volume /etc/hosts:/etc/hosts:ro
 --volume /etc/resolv.conf:/etc/resolv.conf:ro
 --volume /path/to/my-volume:/var/local/my-volume:ro
 --volume /path/to/distrobox-init:/usr/bin/entrypoint:ro
 --entrypoint /usr/bin/entrypoint
 my-image
 --verbose
 --name user
 --user 1000
 --group 1000
 --home /home/user
 --init 1
 --nvidia 0
 --pre-init-hooks echo 'pre-init-hook'
 --additional-packages
 -- echo 'init-hook'
`)

	got := oneline(strings.Join(cmd, " "))

	assert.Equal(t, expected, got)
}

func TestBuildContainerPath(t *testing.T) {
	tests := []struct {
		name          string
		cleanPath     bool
		hostPath      string
		containerPath string
		want          string
	}{
		{
			name:          "cleanPath returns only standard paths",
			cleanPath:     true,
			hostPath:      "/custom/path:/other/path",
			containerPath: "/container/path",
			want:          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:      "cleanPath ignores hostPath and containerPath",
			cleanPath: true,
			hostPath:  "",
			want:      "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:          "hostPath has all standard paths - returns hostPath only",
			cleanPath:     false,
			hostPath:      "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			containerPath: "/container/path",
			want:          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:          "hostPath missing some standard paths - adds them",
			cleanPath:     false,
			hostPath:      "/usr/bin:/bin",
			containerPath: "/container/path",
			want:          "/usr/bin:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/sbin",
		},
		{
			name:          "hostPath with custom paths and missing standard paths",
			cleanPath:     false,
			hostPath:      "/custom/bin:/usr/bin:/another/path",
			containerPath: "/container/path",
			want:          "/custom/bin:/usr/bin:/another/path:/usr/local/sbin:/usr/local/bin:/usr/sbin:/sbin:/bin",
		},
		{
			name:          "empty hostPath - returns containerPath",
			cleanPath:     false,
			hostPath:      "",
			containerPath: "/container/path",
			want:          "/container/path",
		},
		{
			name:      "empty hostPath and containerPath - returns standard paths",
			cleanPath: false,
			hostPath:  "",
			want:      "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:      "hostPath with standard paths at beginning",
			cleanPath: false,
			hostPath:  "/bin:/usr/bin:/custom/path",
			want:      "/bin:/usr/bin:/custom/path:/usr/local/sbin:/usr/local/bin:/usr/sbin:/sbin",
		},
		{
			name:      "hostPath with standard paths at end",
			cleanPath: false,
			hostPath:  "/custom/path:/bin:/usr/bin",
			want:      "/custom/path:/bin:/usr/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/sbin",
		},
		{
			name:      "hostPath with similar but not exact standard paths",
			cleanPath: false,
			hostPath:  "/usr/binary:/binfo",
			want:      "/usr/binary:/binfo:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:          "single custom path in hostPath",
			cleanPath:     false,
			hostPath:      "/opt/custom/bin",
			containerPath: "/container/path",
			want:          "/opt/custom/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:      "empty containerPath with non-empty hostPath",
			cleanPath: false,
			hostPath:  "/custom/path",
			want:      "/custom/path:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name: "empty containerPath with empty hostPath",
			want: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildContainerPath(tt.cleanPath, tt.hostPath, tt.containerPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuildContainerPathEdgeCases tests edge cases and boundary conditions
func TestBuildContainerPathEdgeCases(t *testing.T) {
	t.Run("hostPath with colon at start", func(t *testing.T) {
		got := buildContainerPath(false, ":/usr/bin", "")
		// Should treat this as hostPath not containing standard paths initially
		assert.True(t, contains(got, "/usr/local/sbin"), "Expected standard paths to be added")
	})

	t.Run("hostPath with colon at end", func(t *testing.T) {
		got := buildContainerPath(false, "/usr/bin:", "")
		assert.True(t, contains(got, "/usr/local/sbin"), "Expected standard paths to be added")
	})

	t.Run("hostPath with multiple colons", func(t *testing.T) {
		got := buildContainerPath(false, "/usr/bin::/sbin", "")
		// Should still add missing standard paths
		assert.True(t, contains(got, "/usr/local/sbin"), "Expected standard paths to be added")
	})
}

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name          string
		customCommand string
		user          string
		noTTY         bool
		unshareGroups bool
		want          string
	}{
		{
			name:          "custom command without unshare",
			customCommand: "/bin/bash",
			user:          "testuser",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/bash",
		},
		{
			name:          "custom command with unshare, no TTY",
			customCommand: "/usr/bin/python3",
			user:          "testuser",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|testuser|-s|/bin/sh|-c|"$0" "$@"|--|/usr/bin/python3`,
		},
		{
			name:          "custom command with unshare and TTY",
			customCommand: "/bin/zsh",
			user:          "testuser",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|--pty|testuser|-s|/bin/sh|-c|"$0" "$@"|--|/bin/zsh`,
		},
		{
			name:          "default shell without unshare",
			customCommand: "",
			user:          "alice",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'alice' | cut -f 7 -d :) -l",
		},
		{
			name:          "default shell with unshare, no TTY",
			customCommand: "",
			user:          "bob",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|bob|-s|/bin/sh|-c|"$0" "$@"|--|/bin/sh|-c|$(getent passwd 'bob' | cut -f 7 -d :) -l`,
		},
		{
			name:          "default shell with unshare and TTY",
			customCommand: "",
			user:          "charlie",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|--pty|charlie|-s|/bin/sh|-c|"$0" "$@"|--|/bin/sh|-c|$(getent passwd 'charlie' | cut -f 7 -d :) -l`,
		},
		{
			name:          "empty custom command treated as default",
			customCommand: "",
			user:          "testuser",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'testuser' | cut -f 7 -d :) -l",
		},
		{
			name:          "custom command with spaces without unshare",
			customCommand: "/bin/bash -c 'echo hello'",
			user:          "testuser",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/bash|-c|'echo|hello'",
		},
		{
			name:          "custom command with spaces with unshare",
			customCommand: "/bin/bash -c 'echo hello'",
			user:          "testuser",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|testuser|-s|/bin/sh|-c|"$0" "$@"|--|/bin/bash|-c|'echo|hello'`,
		},
		{
			name:          "user with special characters in default shell",
			customCommand: "",
			user:          "user-name.test",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'user-name.test' | cut -f 7 -d :) -l",
		},
		{
			name:          "root user without unshare",
			customCommand: "",
			user:          "root",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'root' | cut -f 7 -d :) -l",
		},
		{
			name:          "root user with unshare",
			customCommand: "",
			user:          "root",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|root|-s|/bin/sh|-c|"$0" "$@"|--|/bin/sh|-c|$(getent passwd 'root' | cut -f 7 -d :) -l`,
		},
		{
			name:          "empty user with default shell",
			customCommand: "",
			user:          "",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd '' | cut -f 7 -d :) -l",
		},
		{
			name:          "single space as custom command treated as default",
			customCommand: " ",
			user:          "user",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'user' | cut -f 7 -d :) -l",
		},
		{
			name:          "very long custom command with unshare",
			customCommand: "/bin/bash -c 'for i in {1..100}; do echo $i; done'",
			user:          "user",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|user|-s|/bin/sh|-c|"$0" "$@"|--|/bin/bash|-c|'for|i|in|{1..100};|do|echo|$i;|done'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCommandArgs(tt.customCommand, tt.user, tt.noTTY, tt.unshareGroups)
			gotStr := strings.Join(got, "|")

			assert.Equal(t, tt.want, gotStr)
		})
	}
}

// TestBuildCommandArgsMatrix tests all combinations of boolean flags
func TestBuildCommandArgsMatrix(t *testing.T) {
	tests := []struct {
		name            string
		customCommand   string
		noTTY           bool
		unshareGroups   bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "no flags with custom command",
			customCommand:   "/custom/cmd",
			noTTY:           false,
			unshareGroups:   false,
			wantContains:    []string{"/custom/cmd"},
			wantNotContains: []string{"su", "--pty"},
		},
		{
			name:            "unshare only with custom command",
			customCommand:   "/custom/cmd",
			noTTY:           false,
			unshareGroups:   true,
			wantContains:    []string{"su", "/custom/cmd"},
			wantNotContains: []string{"--pty"},
		},
		{
			name:            "noTTY and unshare with custom command",
			customCommand:   "/custom/cmd",
			noTTY:           true,
			unshareGroups:   true,
			wantContains:    []string{"su", "--pty", "/custom/cmd"},
			wantNotContains: []string{},
		},
		{
			name:            "noTTY without unshare",
			customCommand:   "/custom/cmd",
			noTTY:           true,
			unshareGroups:   false,
			wantContains:    []string{"/custom/cmd"},
			wantNotContains: []string{"su", "--pty"},
		},
		{
			name:            "default shell without flags",
			customCommand:   "",
			noTTY:           false,
			unshareGroups:   false,
			wantContains:    []string{"/bin/sh", "-c", "getent passwd"},
			wantNotContains: []string{"su", "--pty"},
		},
		{
			name:            "default shell with unshare",
			customCommand:   "",
			noTTY:           false,
			unshareGroups:   true,
			wantContains:    []string{"su", "/bin/sh", "-c", "getent passwd"},
			wantNotContains: []string{"--pty"},
		},
		{
			name:            "default shell with both flags",
			customCommand:   "",
			noTTY:           true,
			unshareGroups:   true,
			wantContains:    []string{"su", "--pty", "/bin/sh", "-c", "getent passwd"},
			wantNotContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCommandArgs(tt.customCommand, "user", tt.noTTY, tt.unshareGroups)
			resultStr := strings.Join(result, "|")

			for _, want := range tt.wantContains {
				assert.Contains(t, resultStr, want)
			}

			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, resultStr, notWant)
			}
		})
	}
}

// oneline is a helper function that removes newlines and trims spaces from a string.
func oneline(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "  ", " "))
}

// Helper function for edge case tests
func contains(path, substring string) bool {
	return len(path) >= len(substring) &&
		(path[:len(substring)] == substring ||
			len(path) > len(substring) && path[len(path)-len(substring):] == substring ||
			len(path) > len(substring)+1 && hasSubstring(path, ":"+substring+":") ||
			hasSubstring(path, ":"+substring) ||
			hasSubstring(path, substring+":"))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
