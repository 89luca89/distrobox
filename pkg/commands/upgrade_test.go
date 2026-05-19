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

// failingEnterMock embeds the standard mock but returns an error from Enter
// so we can exercise the error path in UpgradeCommand.Execute.
type failingEnterMock struct {
	testutil.MockContainerManager
}

func (m *failingEnterMock) Enter(_ context.Context, _ containermanager.EnterOptions, _ *ui.Progress, _ *ui.Printer) error {
	return errors.New("enter-failed")
}

func TestUpgradeCommand_Execute_PrintsErrorOnUpgradeFailure(t *testing.T) {
	mock := &failingEnterMock{}

	var buf bytes.Buffer
	printer := ui.NewPrinter(&buf, false)
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)

	cmd := commands.NewUpgradeCommand(&config.Values{}, mock, progress, printer, prompter)

	err := cmd.Execute(context.Background(), &commands.UpgradeOptions{
		ContainerNames: []string{"test-box"},
		NonInteractive: true,
	})
	require.Error(t, err)

	output := buf.String()
	assert.Contains(t, output, "error upgrading test-box")
	assert.Contains(t, output, "enter-failed")
}

func TestUpgradeCommand_Execute_NilPrinterDoesNotPanic(t *testing.T) {
	mock := &failingEnterMock{}

	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)

	cmd := commands.NewUpgradeCommand(&config.Values{}, mock, progress, nil, prompter)

	require.NotPanics(t, func() {
		err := cmd.Execute(context.Background(), &commands.UpgradeOptions{
			ContainerNames: []string{"test-box"},
			NonInteractive: true,
		})
		require.Error(t, err)
	})
}
