package commands_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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

// failingInspectMock embeds the standard mock but returns an error from
// InspectContainer so we can exercise the error path in RmCommand.Execute.
type failingInspectMock struct {
	testutil.MockContainerManager

	containers []containermanager.Container
}

func (m *failingInspectMock) ListContainers(_ context.Context) ([]containermanager.Container, error) {
	return m.containers, nil
}

func (m *failingInspectMock) InspectContainer(_ context.Context, _ string) (*containermanager.InspectResult, error) {
	return nil, errors.New("boom")
}

func TestRmCommand_Execute_PrintsErrorOnRemovalFailure(t *testing.T) {
	mock := &failingInspectMock{
		containers: []containermanager.Container{
			{
				Name:   "test-box",
				Status: "Exited",
				Labels: map[string]string{"manager": "distrobox"},
			},
		},
	}

	var buf bytes.Buffer
	printer := ui.NewPrinter(&buf, false)
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)

	cmd := commands.NewRmCommand(&config.Values{}, mock, printer, prompter)

	_, err := cmd.Execute(context.Background(), commands.RmOptions{
		ContainerNames: []string{"test-box"},
		NoTTY:          true,
		Force:          true,
	})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "error deleting test-box")
	assert.Contains(t, output, "boom")
}

func TestRmCommand_Execute_NilPrinterDoesNotPanic(t *testing.T) {
	mock := &failingInspectMock{
		containers: []containermanager.Container{
			{
				Name:   "test-box",
				Status: "Exited",
				Labels: map[string]string{"manager": "distrobox"},
			},
		},
	}

	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	cmd := commands.NewRmCommand(&config.Values{}, mock, nil, prompter)

	require.NotPanics(t, func() {
		_, err := cmd.Execute(context.Background(), commands.RmOptions{
			ContainerNames: []string{"test-box"},
			NoTTY:          true,
			Force:          true,
		})
		require.NoError(t, err)
	})
}
