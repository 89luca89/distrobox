package containermanager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

func TestContainer_IsDistrobox_StandardManagerLabel(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{"manager": "distrobox", "distrobox.unshare_groups": "0"},
	}
	assert.True(t, c.IsDistrobox())
}

// Regression: when the user overrides the manager label via
// `--additional-flags --label=manager=apx`, the container is still a
// distrobox container — the `distrobox.unshare_groups` label is always set on
// creation and is enough to identify it.
func TestContainer_IsDistrobox_ManagerLabelOverridden(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{"manager": "apx", "distrobox.unshare_groups": "0"},
	}
	assert.True(t, c.IsDistrobox())
}

func TestContainer_IsDistrobox_NoDistroboxLabels(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{"manager": "toolbox"},
	}
	assert.False(t, c.IsDistrobox())
}

func TestContainer_IsDistrobox_NilLabels(t *testing.T) {
	c := containermanager.Container{Labels: nil}
	assert.False(t, c.IsDistrobox())
}
