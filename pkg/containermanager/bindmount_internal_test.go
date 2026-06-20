package containermanager

import "testing"

// on darwin, rslave/rshared bind propagation is unsupported and dropped;
// elsewhere the standard suffixes are used.
func TestBindPropagation(t *testing.T) {
	orig := bindGOOS
	t.Cleanup(func() { bindGOOS = orig })

	bindGOOS = "linux"
	if got := BindPropagation(); got != ":rslave" {
		t.Errorf("linux BindPropagation() = %q, want %q", got, ":rslave")
	}
	if got := ReadOnlyBindPropagation(); got != ":ro,rslave" {
		t.Errorf("linux ReadOnlyBindPropagation() = %q, want %q", got, ":ro,rslave")
	}

	bindGOOS = "darwin"
	if got := BindPropagation(); got != "" {
		t.Errorf("darwin BindPropagation() = %q, want empty", got)
	}
	if got := ReadOnlyBindPropagation(); got != ":ro" {
		t.Errorf("darwin ReadOnlyBindPropagation() = %q, want %q", got, ":ro")
	}
}
