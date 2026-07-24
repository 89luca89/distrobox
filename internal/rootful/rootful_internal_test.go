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

package rootful

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetState() {
	validateOnce = sync.Once{}
	errValidate = nil
	cachedSudoCommand = ""
}

func TestValidate_Success(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	err := Validate(context.Background(), "true")
	require.NoError(t, err)
}

func TestValidate_Failure(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	err := Validate(context.Background(), "false")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"false"`)
}

func TestValidate_CachesResult(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	require.NoError(t, Validate(context.Background(), "true"))
	require.NoError(t, Validate(context.Background(), "true"))
}

func TestValidate_MismatchedCommand(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	require.NoError(t, Validate(context.Background(), "true"))

	err := Validate(context.Background(), "false")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
	assert.Contains(t, err.Error(), `"true"`)
	assert.Contains(t, err.Error(), `"false"`)
}
