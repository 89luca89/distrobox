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
