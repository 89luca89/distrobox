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

package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func TestEmitContainerNames(t *testing.T) {
	t.Parallel()

	cm := &listSpy{
		containers: []containermanager.Container{
			{Name: "arch", Labels: map[string]string{"manager": "distrobox"}},
			{Name: "not-ours", Labels: map[string]string{"manager": "toolbox"}},
			{Name: "ubuntu", Labels: map[string]string{"distrobox.unshare_groups": "0"}},
			{Name: "stray"},
		},
	}
	var buf bytes.Buffer
	emitContainerNames(t.Context(), cm, &buf)
	assert.Equal(t, "arch\nubuntu\n", buf.String())
}

func TestEmitContainerNamesSwallowsErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	emitContainerNames(t.Context(), &listSpy{err: errors.New("boom")}, &buf)
	assert.Empty(t, buf.String())
}

func TestLastTokenRequestsImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"empty", nil, false},
		{"no completion flag", []string{"distrobox", "create", "--image", "fedora"}, false},
		{"--image", []string{"distrobox", "create", "--image", "--generate-shell-completion"}, true},
		{"-i", []string{"distrobox", "create", "-i", "--generate-shell-completion"}, true},
		{"--image=foo stays off (matches v1)", []string{"distrobox", "create", "--image=foo", "--generate-shell-completion"}, false},
		{"other flag", []string{"distrobox", "create", "--name", "--generate-shell-completion"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, lastTokenRequestsImage(tc.args))
		})
	}
}

func TestEmitCachedImages(t *testing.T) {
	// XDG_CACHE_HOME is process-global; keep this case serial.
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	cacheDir := filepath.Join(cacheHome, "distrobox")
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))
	// version.Version == "dev" in tests → compatibilityRef → "main".
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "distrobox-compatibility-main"), []byte("fedora:40\nalpine:3.21\n"), 0o644))

	var buf bytes.Buffer
	emitCachedImages(t.Context(), &buf)
	assert.Equal(t, "fedora:40\nalpine:3.21\n", buf.String())
}

// listSpy stubs ContainerManager; only ListContainers is used by the
// completion path. The other methods panic to catch accidental calls.
type listSpy struct {
	containers []containermanager.Container
	err        error
}

func (s *listSpy) ListContainers(_ context.Context) ([]containermanager.Container, error) {
	return s.containers, s.err
}
func (s *listSpy) Name() string                                   { return "listSpy" }
func (s *listSpy) CloneAsRoot() containermanager.ContainerManager { return s }
func (s *listSpy) Enter(_ context.Context, _ containermanager.EnterOptions, _ *ui.Progress, _ *ui.Printer) error {
	panic("not used")
}
func (s *listSpy) Create(_ context.Context, _ containermanager.CreateOptions) error {
	panic("not used")
}
func (s *listSpy) Remove(_ context.Context, _ string, _ containermanager.RmOptions) error {
	panic("not used")
}
func (s *listSpy) Exists(_ context.Context, _ string) bool      { panic("not used") }
func (s *listSpy) ImageExists(_ context.Context, _ string) bool { panic("not used") }
func (s *listSpy) Stop(_ context.Context, _ []string) error     { panic("not used") }
func (s *listSpy) InspectContainer(_ context.Context, _ string) (*containermanager.InspectResult, error) {
	panic("not used")
}
func (s *listSpy) PullImage(_ context.Context, _, _ string, _ bool) error {
	panic("not used")
}
func (s *listSpy) Commit(_ context.Context, _, _ string) error { panic("not used") }
func (s *listSpy) NeedsMigration(_ context.Context, _ string) (bool, error) {
	panic("not used")
}
