// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of the distrobox project:
//    https://github.com/89luca89/distrobox
//
// Copyright (C) 2021 distrobox contributors
//
// distrobox is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License version 3
// as published by the Free Software Foundation.
//
// distrobox is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with distrobox; if not, see <http://www.gnu.org/licenses/>.

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
	"github.com/89luca89/distrobox/pkg/internal/testutil"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newTestEphemeralCommand(mock *testutil.MockContainerManager) *commands.EphemeralCommand {
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	printer := ui.NewPrinter(io.Discard, false)
	return commands.NewEphemeralCommand(&config.Values{}, mock, progress, printer, prompter)
}

func TestEphemeralCommand_PassesCustomCommandToEnter(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestEphemeralCommand(mock)

	customCommand := []string{"cat", "/etc/os-release"}
	err := cmd.Execute(context.Background(), commands.EphemeralOptions{
		CreateOptions: commands.CreateOptions{
			ContainerName:  "ephemeral-test",
			ContainerImage: "alpine:latest",
			DryRun:         true,
		},
		CustomCommand: customCommand,
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Enter, 1, "expected Enter to be called exactly once")
	opts := getEnterOptions(mock.Spy, 0)
	assert.Equal(t, "ephemeral-test", opts.ContainerName)
	assert.Equal(t, customCommand, opts.CustomCommand)
}

func TestEphemeralCommand_EmptyCustomCommandIsNotForwardedAsArgs(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestEphemeralCommand(mock)

	err := cmd.Execute(context.Background(), commands.EphemeralOptions{
		CreateOptions: commands.CreateOptions{
			ContainerName:  "ephemeral-no-cmd",
			ContainerImage: "alpine:latest",
			DryRun:         true,
		},
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Enter, 1, "expected Enter to be called exactly once")
	opts := getEnterOptions(mock.Spy, 0)
	assert.Equal(t, "ephemeral-no-cmd", opts.ContainerName)
	assert.Empty(t, opts.CustomCommand, "expected no custom command to be forwarded")
}
