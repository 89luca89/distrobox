package commands_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
	"github.com/89luca89/distrobox/pkg/ui"
)

// upgrade prints a per-container banner before each box. it runs
// without any prompt, entering each container to run the upgrade script.
func TestUpgradeCommand_PrintsBannerNoPrompt(t *testing.T) {
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: "box1", Status: "Up", Labels: map[string]string{"manager": "distrobox"}},
			{Name: "box2", Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
	}
	var buf strings.Builder
	cmd := commands.NewUpgradeCommand(&config.Values{}, mock, ui.NewDevNullProgress(), ui.NewPrinter(&buf, false))

	err := cmd.Execute(context.Background(), &commands.UpgradeOptions{All: true})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Upgrading box1...")
	assert.Contains(t, out, "Upgrading box2...")
	assert.Len(t, mock.Spy.Enter, 2, "each container is entered to run the upgrade")
}
