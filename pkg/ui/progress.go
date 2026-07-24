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

package ui

import (
	"fmt"
	"io"
)

type Progress struct {
	pending bool
	writer  io.Writer
}

func NewProgress(writer io.Writer) *Progress {
	return &Progress{
		pending: false,
		writer:  writer,
	}
}

func NewDevNullProgress() *Progress {
	return &Progress{
		pending: false,
		writer:  io.Discard,
	}
}

func (p *Progress) Next(message string, a ...any) {
	if p.pending {
		p.Done()
	}

	p.pending = true
	msg := fmt.Sprintf(message, a...)
	fmt.Fprintf(p.writer, "%-40s\t", msg)
}

func (p *Progress) Finalize(message string, a ...any) {
	p.Done()
	p.Next(message, a...)
	fmt.Fprintf(p.writer, "\n")
	p.pending = false
}

func (p *Progress) Done() {
	if !p.pending {
		return
	}

	p.pending = false
	fmt.Fprintf(p.writer, "%s\n", Green("[ OK ] "))
}

func (p *Progress) Fail() {
	if !p.pending {
		return
	}

	p.pending = false
	fmt.Fprintf(p.writer, "%s\n", Red("[ ERR ]"))
}
