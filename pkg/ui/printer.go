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

type Printer struct {
	writer   io.Writer
	colorful bool
}

func NewPrinter(writer io.Writer, colorful bool) *Printer {
	return &Printer{
		writer:   writer,
		colorful: colorful,
	}
}

func (p *Printer) Print(msg string, a ...any) {
	fmt.Fprintf(p.writer, msg, a...)
}

func (p *Printer) Println(msg string, a ...any) {
	p.Print(msg+"\n", a...)
}

func (p *Printer) PrintWarning(msg string, a ...any) {
	if p.colorful {
		msg = Yellow(msg)
	}
	p.Print(msg, a...)
}

func (p *Printer) PrintWarningln(msg string, a ...any) {
	if p.colorful {
		msg = Yellow(msg)
	}
	p.Println(msg, a...)
}

func (p *Printer) PrintError(msg string, a ...any) {
	if p.colorful {
		msg = Red(msg)
	}
	p.Print(msg, a...)
}

func (p *Printer) PrintErrorln(msg string, a ...any) {
	if p.colorful {
		msg = Red(msg)
	}
	p.Println(msg, a...)
}
