package cli

import "testing"

func TestResolveManifestPath_FlagOnly(t *testing.T) {
	got := resolveManifestPath("/etc/distrobox.ini", nil)
	if got != "/etc/distrobox.ini" {
		t.Fatalf("expected flag value, got %q", got)
	}
}

func TestResolveManifestPath_PositionalOnly(t *testing.T) {
	got := resolveManifestPath("", []string{"/srv/manifest.ini"})
	if got != "/srv/manifest.ini" {
		t.Fatalf("expected positional value, got %q", got)
	}
}

func TestResolveManifestPath_FlagTakesPrecedenceOverPositional(t *testing.T) {
	got := resolveManifestPath("/etc/distrobox.ini", []string{"/srv/manifest.ini"})
	if got != "/etc/distrobox.ini" {
		t.Fatalf("expected flag value to win, got %q", got)
	}
}

func TestResolveManifestPath_NoInputsReturnsDefault(t *testing.T) {
	got := resolveManifestPath("", nil)
	if got != defaultManifestPath {
		t.Fatalf("expected default %q, got %q", defaultManifestPath, got)
	}
}

func TestResolveManifestPath_EmptyPositionalSliceReturnsDefault(t *testing.T) {
	got := resolveManifestPath("", []string{})
	if got != defaultManifestPath {
		t.Fatalf("expected default %q, got %q", defaultManifestPath, got)
	}
}

func TestResolveManifestPath_EmptyFirstPositionalReturnsDefault(t *testing.T) {
	got := resolveManifestPath("", []string{""})
	if got != defaultManifestPath {
		t.Fatalf("expected default %q, got %q", defaultManifestPath, got)
	}
}

func TestResolveManifestPath_FirstPositionalUsedWhenMultiple(t *testing.T) {
	got := resolveManifestPath("", []string{"/first.ini", "/second.ini"})
	if got != "/first.ini" {
		t.Fatalf("expected first positional, got %q", got)
	}
}

func TestResolveManifestPath_URLFlagValue(t *testing.T) {
	got := resolveManifestPath("https://example.com/manifest.ini", nil)
	if got != "https://example.com/manifest.ini" {
		t.Fatalf("expected URL flag value, got %q", got)
	}
}

func TestResolveManifestPath_URLPositionalValue(t *testing.T) {
	got := resolveManifestPath("", []string{"https://example.com/manifest.ini"})
	if got != "https://example.com/manifest.ini" {
		t.Fatalf("expected URL positional value, got %q", got)
	}
}
