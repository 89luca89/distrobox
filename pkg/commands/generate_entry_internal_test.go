package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDesktopIcon_AutoDetectsFedora(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "fedora-39")

	want := "https://raw.githubusercontent.com/89luca89/distrobox/main/docs/assets/png/distros/fedora-distrobox.png"
	assert.Equal(t, want, got)
}

func TestGetDesktopIcon_AutoDetectsUbuntu(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "ubuntu-22")

	want := "https://raw.githubusercontent.com/89luca89/distrobox/main/docs/assets/png/distros/ubuntu-distrobox.png"
	assert.Equal(t, want, got)
}

func TestGetDesktopIcon_AutoFallsBackToTerminalOnUnknownDistro(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "unknown-distro")

	assert.Equal(t, defaultEntryIcon, got)
}

func TestGetDesktopIcon_CustomIconIsPassedThroughUnchanged(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("custom-url", "fedora")

	assert.Equal(t, "custom-url", got)
}

func TestGetDesktopIcon_AutoDetectsOpenSUSETumbleweedBeforeGenericOpenSUSE(t *testing.T) {
	// Regression: opensuse-tumbleweed must be matched as a specific variant,
	// not collapsed to a non-existent generic "opensuse" entry.
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "registry.opensuse.org/opensuse/tumbleweed:latest")

	want := "https://raw.githubusercontent.com/89luca89/distrobox/main/docs/assets/png/distros/opensuse-distrobox.png"
	assert.Equal(t, want, got)
}

func TestGetDesktopIcon_AutoDetectsArchFromImageRegistry(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "docker.io/library/archlinux:latest")

	want := "https://raw.githubusercontent.com/89luca89/distrobox/main/docs/assets/png/distros/arch-distrobox.png"
	assert.Equal(t, want, got)
}

func TestGetDesktopIcon_AutoIsCaseInsensitive(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "Fedora-Toolbox")

	want := "https://raw.githubusercontent.com/89luca89/distrobox/main/docs/assets/png/distros/fedora-distrobox.png"
	assert.Equal(t, want, got)
}

func TestGetDesktopIcon_AutoWithEmptyHintFallsBackToDefault(t *testing.T) {
	c := &GenerateEntryCommand{}

	got := c.getDesktopIcon("auto", "")

	assert.Equal(t, defaultEntryIcon, got)
}
