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

package nspawn

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// minRootlessSystemdVersion is the first systemd release where unprivileged
// directory-based nspawn (via systemd-nsresourced and systemd-mountfsd) is
// usable.
const minRootlessSystemdVersion = 257

// ErrRootlessUnsupported is returned when the host cannot run unprivileged
// nspawn machines.
var ErrRootlessUnsupported = errors.New("rootless systemd-nspawn is not supported on this host")

// prober collects the host facts needed to decide whether rootless nspawn
// can work. The seams exist for tests.
type prober struct {
	lookPath func(file string) (string, error)
	// runner executes a command and returns its combined trimmed output.
	runner func(ctx context.Context, argv ...string) (string, error)
}

func defaultProber() prober {
	return prober{
		lookPath: exec.LookPath,
		runner: func(ctx context.Context, argv ...string) (string, error) {
			//nolint:gosec // argv is a fixed set of systemctl invocations
			out, err := exec.CommandContext(ctx, argv[0], argv[1:]...).CombinedOutput()
			return strings.TrimSpace(string(out)), err
		},
	}
}

// probeRootless verifies the host supports unprivileged nspawn machines,
// memoized for the process lifetime. Only mutating operations call it;
// metadata reads (list/inspect/exists) must keep working everywhere.
func (n *Nspawn) probeRootless(ctx context.Context) error {
	n.probeOnce.Do(func() {
		n.probeResult = runProbe(ctx, defaultProber())
	})
	return n.probeResult
}

func runProbe(ctx context.Context, p prober) error {
	var problems []string

	if _, err := p.lookPath("systemd-nspawn"); err != nil {
		problems = append(problems, "systemd-nspawn not found in PATH (install systemd-container)")
	}

	// Unprivileged machines always live in a private network namespace;
	// pasta provides their user-mode networking, like rootless podman.
	if _, err := p.lookPath("pasta"); err != nil {
		problems = append(problems, "pasta not found in PATH (install passt)")
	}

	if version, err := systemdVersion(ctx, p); err != nil {
		problems = append(problems, fmt.Sprintf("cannot determine systemd version: %v", err))
	} else if version < minRootlessSystemdVersion {
		problems = append(problems,
			fmt.Sprintf("systemd %d found, need >= %d", version, minRootlessSystemdVersion))
	}

	for _, socket := range []string{"systemd-mountfsd.socket", "systemd-nsresourced.socket"} {
		if !socketAvailable(ctx, p, socket) {
			problems = append(problems, socket+": not enabled")
		}
	}

	if out, err := p.runner(ctx, "systemctl", "--user", "is-system-running"); err != nil && out == "" {
		problems = append(problems, "systemd user manager is not reachable")
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%w:\n  - %s\n"+
		"Unprivileged nspawn machines need systemd-nsresourced and systemd-mountfsd (systemd >= %d).\n"+
		"Enable them with: sudo systemctl enable --now systemd-mountfsd.socket systemd-nsresourced.socket\n"+
		"or re-run with --root to use rootful nspawn",
		ErrRootlessUnsupported, strings.Join(problems, "\n  - "), minRootlessSystemdVersion)
}

func systemdVersion(ctx context.Context, p prober) (int, error) {
	out, err := p.runner(ctx, "systemctl", "--version")
	if err != nil {
		return 0, err
	}
	// First line looks like: "systemd 257 (257.1-arch1)".
	fields := strings.Fields(strings.SplitN(out, "\n", 2)[0])
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected systemctl --version output: %q", out)
	}
	version, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, fmt.Errorf("unexpected systemctl --version output: %q", out)
	}
	return version, nil
}

// socketAvailable reports whether the given socket unit can serve requests:
// it must be enabled (so it comes up on boot) or currently active. A merely
// installed but disabled socket does not count — socket activation cannot
// reach a service whose socket is not listening.
func socketAvailable(ctx context.Context, p prober, socket string) bool {
	out, _ := p.runner(ctx, "systemctl", "is-enabled", socket)
	switch strings.TrimSpace(out) {
	case "enabled", "static", "alias", "indirect", "generated":
		return true
	}
	out, _ = p.runner(ctx, "systemctl", "is-active", socket)
	return strings.TrimSpace(out) == "active"
}
