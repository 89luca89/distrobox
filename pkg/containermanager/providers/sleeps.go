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

import "time"

// Sleep durations used while waiting for a container's setup to complete.
const (
	// logsRetryInterval is the delay before retrying when fetching container
	// logs fails transiently during setup waiting.
	logsRetryInterval = 100 * time.Millisecond
	// setupPollInterval is the cadence at which container setup logs are
	// polled while waiting for the setup-done marker.
	setupPollInterval = 500 * time.Millisecond
)
