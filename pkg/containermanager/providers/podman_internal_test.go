package providers

import (
	"strings"
	"testing"

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
		if !strings.Contains(cmdStr, flag) {
			t.Errorf("Expected command to contain '%s', but it was missing.\nCommand: %s", flag, cmdStr)
		}
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
		if strings.Contains(cmdStr, flag) {
			t.Errorf(
				"Expected command NOT to contain Docker-specific flag '%s', but it was present.\nCommand: %s",
				flag,
				cmdStr,
			)
		}
	}

	// Check common flags are present
	commonFlags := []string{
		"--hostname my-hostname",
		"--name my-container",
		"--privileged",
		"--env container=podman",
	}

	for _, flag := range commonFlags {
		if !strings.Contains(cmdStr, flag) {
			t.Errorf("Expected command to contain common flag '%s', but it was missing.\nCommand: %s", flag, cmdStr)
		}
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
	if strings.Contains(cmdStr, "--userns keep-id") {
		t.Errorf("Expected rootful command NOT to contain '--userns keep-id', but it was present.\nCommand: %s", cmdStr)
	}

	// Should still have other Podman-specific flags
	if !strings.Contains(cmdStr, "--annotation run.oci.keep_original_groups=1") {
		t.Errorf("Expected command to contain '--annotation run.oci.keep_original_groups=1'")
	}
	if !strings.Contains(cmdStr, "--ulimit host") {
		t.Errorf("Expected command to contain '--ulimit host'")
	}
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
	if strings.Contains(cmdStr, "--systemd=always") {
		t.Errorf(
			"Expected non-init command NOT to contain '--systemd=always', but it was present.\nCommand: %s",
			cmdStr,
		)
	}

	// Should still have other Podman-specific flags
	if !strings.Contains(cmdStr, "--annotation run.oci.keep_original_groups=1") {
		t.Errorf("Expected command to contain '--annotation run.oci.keep_original_groups=1'")
	}
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
		if !strings.Contains(cmdStr, "--runtime=crun") {
			t.Errorf(
				"Expected command to contain '--runtime=crun' when crun is available, but it was missing.\nCommand: %s",
				cmdStr,
			)
		}
	} else {
		if strings.Contains(cmdStr, "--runtime=crun") {
			t.Errorf(
				"Expected command NOT to contain '--runtime=crun' when crun is not available, but it was present.\nCommand: %s",
				cmdStr,
			)
		}
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

	if !strings.Contains(cmdStr, "--platform=linux/arm64") {
		t.Errorf("Expected command to contain '--platform=linux/arm64', but it was missing.\nCommand: %s", cmdStr)
	}
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
	if !strings.Contains(cmdStr, "--env HOME=/custom/home") {
		t.Errorf("Expected command to contain '--env HOME=/custom/home', but it was missing")
	}

	// Check DISTROBOX_HOST_HOME is set to original home
	if !strings.Contains(cmdStr, "--env DISTROBOX_HOST_HOME=/home/user") {
		t.Errorf("Expected command to contain '--env DISTROBOX_HOST_HOME=/home/user', but it was missing")
	}

	// Check custom home is mounted
	if !strings.Contains(cmdStr, "--volume /custom/home:/custom/home:rslave") {
		t.Errorf("Expected command to contain custom home volume mount, but it was missing")
	}

	// Check entrypoint args use custom home
	if !strings.Contains(cmdStr, "--home /custom/home") {
		t.Errorf("Expected entrypoint args to use custom home, but it was missing")
	}
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

	if !strings.Contains(cmdStr, "--cap-add=SYS_ADMIN") {
		t.Errorf("Expected command to contain additional flag '--cap-add=SYS_ADMIN', but it was missing")
	}
	if !strings.Contains(cmdStr, "--device=/dev/fuse") {
		t.Errorf("Expected command to contain additional flag '--device=/dev/fuse', but it was missing")
	}
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

	if !strings.Contains(cmdStr, "--additional-packages vim git htop") {
		t.Errorf("Expected command to contain additional packages, but it was missing")
	}
}

//nolint:gocognit
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
				if !strings.Contains(cmdStr, "--ipc host") {
					t.Errorf("Expected '--ipc host' in command")
				}
			} else {
				if strings.Contains(cmdStr, "--ipc host") {
					t.Errorf("Did not expect '--ipc host' in command")
				}
			}

			if tt.wantNetwork {
				if !strings.Contains(cmdStr, "--network host") {
					t.Errorf("Expected '--network host' in command")
				}
			} else {
				if strings.Contains(cmdStr, "--network host") {
					t.Errorf("Did not expect '--network host' in command")
				}
			}

			if tt.wantPID {
				if !strings.Contains(cmdStr, "--pid host") {
					t.Errorf("Expected '--pid host' in command")
				}
			} else {
				if strings.Contains(cmdStr, "--pid host") {
					t.Errorf("Did not expect '--pid host' in command")
				}
			}
		})
	}
}

func TestPodman_Name(t *testing.T) {
	podman := NewPodman(false, "sudo", false)
	if podman.Name() != "podman" {
		t.Errorf("Expected Name() to return 'podman', got '%s'", podman.Name())
	}
}

func TestParsePodmanContainerList(t *testing.T) {
	// Test parsing JSON array output from podman ps --format json
	output := `[{"Command":"sleep infinity","CreatedAt":"2024-01-31 10:00:00 +0000 UTC","ID":"abc123def456789012345678901234567890","Image":"fedora:39","Labels":{"manager":"distrobox","distrobox.unshare_groups":"0"},"Mounts":"","Names":["my-container"],"Networks":"","Ports":"","State":"running","Status":"Up 2 hours"},{"Command":"/bin/bash","CreatedAt":"2024-01-31 09:00:00 +0000 UTC","ID":"xyz789abc123456789012345678901234567890","Image":"ubuntu:22.04","Labels":{"manager":"distrobox"},"Mounts":"","Names":["test-box"],"Networks":"","Ports":"","State":"exited","Status":"Exited (0) 1 hour ago"}]`

	containers, err := parsePodmanContainerList(output)
	if err != nil {
		t.Fatalf("parsePodmanContainerList returned error: %v", err)
	}

	if len(containers) != 2 {
		t.Fatalf("Expected 2 containers, got %d", len(containers))
	}

	// Check first container
	if containers[0].ID != "abc123def456" {
		t.Errorf("Expected container ID 'abc123def456', got '%s'", containers[0].ID)
	}
	if containers[0].Name != "my-container" {
		t.Errorf("Expected container name 'my-container', got '%s'", containers[0].Name)
	}
	if containers[0].Image != "fedora:39" {
		t.Errorf("Expected container image 'fedora:39', got '%s'", containers[0].Image)
	}
	if containers[0].Labels["manager"] != "distrobox" {
		t.Errorf("Expected label manager=distrobox")
	}
	if containers[0].Labels["distrobox.unshare_groups"] != "0" {
		t.Errorf("Expected label distrobox.unshare_groups=0")
	}

	// Check second container
	if containers[1].ID != "xyz789abc123" {
		t.Errorf("Expected container ID 'xyz789abc123', got '%s'", containers[1].ID)
	}
	if containers[1].Name != "test-box" {
		t.Errorf("Expected container name 'test-box', got '%s'", containers[1].Name)
	}
}

func TestParsePodmanContainerListEmpty(t *testing.T) {
	output := ""
	containers, err := parsePodmanContainerList(output)
	if err != nil {
		t.Fatalf("parsePodmanContainerList returned error: %v", err)
	}

	if len(containers) != 0 {
		t.Errorf("Expected 0 containers for empty output, got %d", len(containers))
	}
}

func TestParsePodmanContainerListEmptyNames(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		wantID string
	}{
		{
			name:   "empty names array falls back to truncated container ID",
			json:   `[{"ID":"abc123def456789012345678","Image":"fedora:39","Names":[],"Status":"running","Labels":{}}]`,
			wantID: "abc123def456",
		},
		{
			name:   "null names falls back to truncated container ID",
			json:   `[{"ID":"xyz789abc123456789012345","Image":"ubuntu:22.04","Names":null,"Status":"exited","Labels":{}}]`,
			wantID: "xyz789abc123",
		},
		{
			name:   "short ID (under 12 chars) is used as-is",
			json:   `[{"ID":"shortid","Image":"alpine:latest","Names":[],"Status":"running","Labels":{}}]`,
			wantID: "shortid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containers, err := parsePodmanContainerList(tt.json)
			if err != nil {
				t.Fatalf("parsePodmanContainerList returned error: %v", err)
			}
			if len(containers) != 1 {
				t.Fatalf("Expected 1 container, got %d", len(containers))
			}
			if containers[0].Name != tt.wantID {
				t.Errorf("Expected Name to fall back to container ID %q, got %q", tt.wantID, containers[0].Name)
			}
			if containers[0].ID != tt.wantID {
				t.Errorf("Expected ID %q, got %q", tt.wantID, containers[0].ID)
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	// Test with a command that should exist on all systems
	if !commandExists("sh") {
		t.Error("Expected 'sh' command to exist")
	}

	// Test with a command that should not exist
	if commandExists("this-command-definitely-does-not-exist-12345") {
		t.Error("Did not expect non-existent command to be found")
	}
}
