package providers

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/internal/userenv"
)

func TestPodman_makeCreateCommand(t *testing.T) {
	podman := NewPodman(false, "sudo", false)

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

	cmd := podman.makeCreateCommand(
		t.Context(),
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

	cmdStr := strings.Join(cmd, " ")

	// Check Podman-specific flags are present
	requiredFlags := []string{
		"--annotation run.oci.keep_original_groups=1",
		"--ulimit host",
		"--systemd=always",
		"--userns keep-id",
	}

	for _, flag := range requiredFlags {
		assert.Contains(t, cmdStr, flag)
	}

	// Check Docker-specific flags are NOT present
	dockerOnlyFlags := []string{
		"--cgroupns host",
		"--stop-signal SIGRTMIN+3",
		"type=tmpfs,destination=/run",
		"type=tmpfs,destination=/run/lock",
		"type=tmpfs,destination=/var/lib/journal",
	}

	for _, flag := range dockerOnlyFlags {
		assert.NotContains(t, cmdStr, flag)
	}

	// Check common flags are present
	commonFlags := []string{
		"--hostname my-hostname",
		"--name my-container",
		"--privileged",
		"--env container=podman",
	}

	for _, flag := range commonFlags {
		assert.Contains(t, cmdStr, flag)
	}
}

func TestPodman_makeCreateCommandRootful(t *testing.T) {
	// Test rootful mode - should NOT have --userns keep-id
	podman := NewPodman(true, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		[]string{},
		"my-hostname",
		[]string{},
		[]string{},
		"",
		"",
		false,
		false,
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	// Rootful mode should NOT have --userns keep-id
	assert.NotContains(t, cmdStr, "--userns keep-id")

	// Should still have other Podman-specific flags
	assert.Contains(t, cmdStr, "--annotation run.oci.keep_original_groups=1")
	assert.Contains(t, cmdStr, "--ulimit host")
}

func TestPodman_makeCreateCommandNoInit(t *testing.T) {
	// Test without init - should NOT have --systemd=always
	podman := NewPodman(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		[]string{},
		"my-hostname",
		[]string{},
		[]string{},
		"",
		"",
		false,
		false, // init = false
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	// Without init, should NOT have --systemd=always
	assert.NotContains(t, cmdStr, "--systemd=always")

	// Should still have other Podman-specific flags
	assert.Contains(t, cmdStr, "--annotation run.oci.keep_original_groups=1")
}

func TestPodman_makeCreateCommandWithCrun(t *testing.T) {
	// This test checks that if crun exists, --runtime=crun is added
	// Note: This test will pass or fail depending on whether crun is installed
	// on the test system. In a real scenario, you might want to mock commandExists()
	podman := NewPodman(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		[]string{},
		"my-hostname",
		[]string{},
		[]string{},
		"",
		"",
		false,
		false,
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	// If crun exists, it should be in the command
	if commandExists("crun") {
		assert.Contains(t, cmdStr, "--runtime=crun")
	} else {
		assert.NotContains(t, cmdStr, "--runtime=crun")
	}
}

func TestPodman_makeCreateCommandWithPlatform(t *testing.T) {
	podman := NewPodman(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		[]string{},
		"my-hostname",
		[]string{},
		[]string{},
		"",
		"linux/arm64", // containerPlatform
		false,
		false,
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	assert.Contains(t, cmdStr, "--platform=linux/arm64")
}

func TestPodman_makeCreateCommandWithCustomHome(t *testing.T) {
	podman := NewPodman(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		[]string{},
		"my-hostname",
		[]string{},
		[]string{},
		"/custom/home", // containerUserCustomHome
		"",
		false,
		false,
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	// Check custom home is set
	assert.Contains(t, cmdStr, "--env HOME=/custom/home")

	// Check DISTROBOX_HOST_HOME is set to original home
	assert.Contains(t, cmdStr, "--env DISTROBOX_HOST_HOME=/home/user")

	// Check custom home is mounted
	assert.Contains(t, cmdStr, "--volume /custom/home:/custom/home:rslave")

	// Check entrypoint args use custom home
	assert.Contains(t, cmdStr, "--home /custom/home")
}

func TestPodman_makeCreateCommandWithAdditionalFlags(t *testing.T) {
	podman := NewPodman(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	additionalFlags := []string{
		"--cap-add=SYS_ADMIN",
		"--device=/dev/fuse",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		additionalFlags,
		"my-hostname",
		[]string{},
		[]string{},
		"",
		"",
		false,
		false,
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	assert.Contains(t, cmdStr, "--cap-add=SYS_ADMIN")
	assert.Contains(t, cmdStr, "--device=/dev/fuse")
}

func TestPodman_makeCreateCommandWithAdditionalPackages(t *testing.T) {
	podman := NewPodman(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	additionalPackages := []string{
		"vim",
		"git",
		"htop",
	}

	cmd := podman.makeCreateCommand(
		t.Context(),
		"my-container",
		"my-image",
		[]string{},
		"my-hostname",
		additionalPackages,
		[]string{},
		"",
		"",
		false,
		false,
		"",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		userEnv,
		"/path/to/distrobox-init",
		"/path/to/distrobox-export",
		"/path/to/distrobox-hostexec",
	)

	cmdStr := strings.Join(cmd, " ")

	assert.Contains(t, cmdStr, "--additional-packages vim git htop")
}

func TestPodman_makeCreateCommandUnshareOptions(t *testing.T) {
	tests := []struct {
		name           string
		unshareIPC     bool
		unshareNetNS   bool
		unshareProcess bool
		wantIPC        bool
		wantNetwork    bool
		wantPID        bool
	}{
		{
			name:           "no unshare - all host shared",
			unshareIPC:     false,
			unshareNetNS:   false,
			unshareProcess: false,
			wantIPC:        true,
			wantNetwork:    true,
			wantPID:        true,
		},
		{
			name:           "unshare IPC",
			unshareIPC:     true,
			unshareNetNS:   false,
			unshareProcess: false,
			wantIPC:        false,
			wantNetwork:    true,
			wantPID:        true,
		},
		{
			name:           "unshare network",
			unshareIPC:     false,
			unshareNetNS:   true,
			unshareProcess: false,
			wantIPC:        true,
			wantNetwork:    false,
			wantPID:        true,
		},
		{
			name:           "unshare process",
			unshareIPC:     false,
			unshareNetNS:   false,
			unshareProcess: true,
			wantIPC:        true,
			wantNetwork:    true,
			wantPID:        false,
		},
		{
			name:           "unshare all",
			unshareIPC:     true,
			unshareNetNS:   true,
			unshareProcess: true,
			wantIPC:        false,
			wantNetwork:    false,
			wantPID:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podman := NewPodman(false, "sudo", false)

			userEnv := &userenv.UserEnvironment{
				User:    "user",
				UserID:  "1000",
				GroupID: "1000",
				Home:    "/home/user",
				Shell:   "/bin/sh",
			}

			cmd := podman.makeCreateCommand(
				t.Context(),
				"my-container",
				"my-image",
				[]string{},
				"my-hostname",
				[]string{},
				[]string{},
				"",
				"",
				false,
				false,
				"",
				"",
				false,
				false,
				false,
				tt.unshareIPC,
				tt.unshareNetNS,
				tt.unshareProcess,
				userEnv,
				"/path/to/distrobox-init",
				"/path/to/distrobox-export",
				"/path/to/distrobox-hostexec",
			)

			cmdStr := strings.Join(cmd, " ")

			if tt.wantIPC {
				assert.Contains(t, cmdStr, "--ipc host")
			} else {
				assert.NotContains(t, cmdStr, "--ipc host")
			}

			if tt.wantNetwork {
				assert.Contains(t, cmdStr, "--network host")
			} else {
				assert.NotContains(t, cmdStr, "--network host")
			}

			if tt.wantPID {
				assert.Contains(t, cmdStr, "--pid host")
			} else {
				assert.NotContains(t, cmdStr, "--pid host")
			}
		})
	}
}

func TestPodman_Name(t *testing.T) {
	podman := NewPodman(false, "sudo", false)
	assert.Equal(t, "podman", podman.Name())

	launcher := NewPodmanLauncher(false, "sudo", false)
	assert.Equal(t, "podman-launcher", launcher.Name())
}

func TestParsePodmanContainerList(t *testing.T) {
	// Test parsing JSON array output from podman ps --format json
	output := `[{"Command":"sleep infinity","CreatedAt":"2024-01-31 10:00:00 +0000 UTC","ID":"abc123def456789012345678901234567890","Image":"fedora:39","Labels":{"manager":"distrobox","distrobox.unshare_groups":"0"},"Mounts":"","Names":["my-container"],"Networks":"","Ports":"","State":"running","Status":"Up 2 hours"},{"Command":"/bin/bash","CreatedAt":"2024-01-31 09:00:00 +0000 UTC","ID":"xyz789abc123456789012345678901234567890","Image":"ubuntu:22.04","Labels":{"manager":"distrobox"},"Mounts":"","Names":["test-box"],"Networks":"","Ports":"","State":"exited","Status":"Exited (0) 1 hour ago"}]`

	containers, err := parsePodmanContainerList(output)
	require.NoError(t, err)
	require.Len(t, containers, 2)

	// Check first container
	assert.Equal(t, "abc123def456", containers[0].ID)
	assert.Equal(t, "my-container", containers[0].Name)
	assert.Equal(t, "fedora:39", containers[0].Image)
	assert.Equal(t, "distrobox", containers[0].Labels["manager"])
	assert.Equal(t, "0", containers[0].Labels["distrobox.unshare_groups"])

	// Check second container
	assert.Equal(t, "xyz789abc123", containers[1].ID)
	assert.Equal(t, "test-box", containers[1].Name)
}

func TestParsePodmanContainerListEmpty(t *testing.T) {
	output := ""
	containers, err := parsePodmanContainerList(output)
	require.NoError(t, err)
	assert.Empty(t, containers)
}

func TestParsePodmanContainerListEmptyNames(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantName string
		wantID   string
	}{
		{
			name:     "empty names array falls back to empty string",
			json:     `[{"ID":"abc123def456789012345678","Image":"fedora:39","Names":[],"Status":"running","Labels":{}}]`,
			wantName: "",
			wantID:   "abc123def456",
		},
		{
			name:     "null names falls back to empty string",
			json:     `[{"ID":"xyz789abc123456789012345","Image":"ubuntu:22.04","Names":null,"Status":"exited","Labels":{}}]`,
			wantName: "",
			wantID:   "xyz789abc123",
		},
		{
			name:     "short ID (under 12 chars) is used as-is",
			json:     `[{"ID":"shortid","Image":"alpine:latest","Names":[],"Status":"running","Labels":{}}]`,
			wantName: "",
			wantID:   "shortid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containers, err := parsePodmanContainerList(tt.json)
			require.NoError(t, err)
			require.Len(t, containers, 1)
			assert.Equal(t, tt.wantName, containers[0].Name)
			assert.Equal(t, tt.wantID, containers[0].ID)
		})
	}
}

func TestPodman_runUsesCorrectBinary(t *testing.T) {
	tests := []struct {
		name           string
		constructor    func() *Podman
		expectedPrefix string
	}{
		{
			name:           "NewPodman uses podman binary",
			constructor:    func() *Podman { return NewPodman(false, "sudo", false) },
			expectedPrefix: "podman ",
		},
		{
			name:           "NewPodmanLauncher uses podman-launcher binary",
			constructor:    func() *Podman { return NewPodmanLauncher(false, "sudo", false) },
			expectedPrefix: "podman-launcher ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.constructor()

			// Capture stdout during DryRun
			origStdout := os.Stdout
			r, w, err := os.Pipe()
			require.NoError(t, err)
			os.Stdout = w //nolint:reassign // redirect stdout to capture dry-run output in test

			_, _ = p.run(t.Context(), []string{"info"}, runOptions{DryRun: true})

			w.Close()
			os.Stdout = origStdout //nolint:reassign // restore original stdout

			var buf bytes.Buffer
			_, err = buf.ReadFrom(r)
			require.NoError(t, err)
			got := buf.String()

			assert.True(t, strings.HasPrefix(got, tt.expectedPrefix),
				"expected output to start with %q, got %q", tt.expectedPrefix, got)
		})
	}
}

func TestCommandExists(t *testing.T) {
	// Test with a command that should exist on all systems
	assert.True(t, commandExists("sh"), "Expected 'sh' command to exist")

	// Test with a command that should not exist
	assert.False(t, commandExists("this-command-definitely-does-not-exist-12345"), "Did not expect non-existent command to be found")
}
