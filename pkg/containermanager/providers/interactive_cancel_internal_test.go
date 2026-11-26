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

package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInteractiveRunSurvivesContextCancel(t *testing.T) {
	// A shell stands in for the container engine. Docker runs "docker" from PATH.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte("#!/bin/sh\nexec /bin/sh \"$@\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	run := map[string]func(context.Context, []string, runOptions) (string, error){
		"podman": (&Podman{command: "sh"}).run,
		"docker": (&Docker{}).run,
	}

	for name, engine := range run {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			// What Ctrl+C does while a session is open.
			time.AfterFunc(200*time.Millisecond, cancel)

			if _, err := engine(ctx, []string{"-c", "sleep 1"}, runOptions{Interactive: true}); err != nil {
				t.Fatalf("interactive session was ended by the cancel: %v", err)
			}
		})
	}
}
