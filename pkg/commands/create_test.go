package commands_test

import (
	"bufio"
	"context"
	"io"
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

func newTestCreateCommand(mock *testutil.MockContainerManager) *commands.CreateCommand {
	return newTestCreateCommandWithConfig(mock, &config.Values{})
}

func newTestCreateCommandWithConfig(mock *testutil.MockContainerManager, cfg *config.Values) *commands.CreateCommand {
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	return commands.NewCreateCommand(cfg, mock, progress, prompter)
}

func TestCreateCommand_Execute_DefaultNameOnDefaultImage(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cfg := &config.Values{
		DefaultContainerImage: "registry.fedoraproject.org/fedora-toolbox:latest",
		DefaultContainerName:  "my-distrobox",
	}
	cmd := newTestCreateCommandWithConfig(mock, cfg)

	// No name given and the (flag-default) image is the default image: the name
	// must fall back to DefaultContainerName, not basename(image).
	_, err := cmd.Execute(context.Background(), commands.CreateOptions{
		ContainerImage: cfg.DefaultContainerImage,
		DryRun:         true,
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Create, 1)
	opts := mock.Spy.Create[0][0].(containermanager.CreateOptions)
	assert.Equal(t, "my-distrobox", opts.ContainerName)
}

func TestCreateCommand_Execute_AdditionalFlagsAreWordSplit(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestCreateCommand(mock)

	_, err := cmd.Execute(context.Background(), commands.CreateOptions{
		ContainerImage:  "alpine:latest",
		ContainerName:   "test",
		AdditionalFlags: []string{"--label=name=test --label=stack=alpine"},
		DryRun:          true,
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Create, 1)
	opts := mock.Spy.Create[0][0].(containermanager.CreateOptions)
	assert.Equal(t, []string{"--label=name=test", "--label=stack=alpine"}, opts.AdditionalFlags)
}

func TestCreateCommand_Execute_AdditionalFlagsAcrossMultipleEntries(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestCreateCommand(mock)

	_, err := cmd.Execute(context.Background(), commands.CreateOptions{
		ContainerImage:  "alpine:latest",
		ContainerName:   "test",
		AdditionalFlags: []string{"--a --b", "--c=1", "--d --e"},
		DryRun:          true,
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Create, 1)
	opts := mock.Spy.Create[0][0].(containermanager.CreateOptions)
	assert.Equal(t, []string{"--a", "--b", "--c=1", "--d", "--e"}, opts.AdditionalFlags)
}
