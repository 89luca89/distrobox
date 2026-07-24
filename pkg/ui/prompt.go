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
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
)

type Prompter struct {
	reader bufio.Reader
	writer io.Writer
}

func NewPrompter(reader bufio.Reader, writer io.Writer) *Prompter {
	return &Prompter{
		reader: reader,
		writer: writer,
	}
}

func (p *Prompter) Prompt(label string, defaultChoice bool) bool {
	choices := getChoices(defaultChoice)

	yes := []string{"y", "yes"}
	no := []string{"n", "no"}

	for {
		fmt.Fprintf(p.writer, "%s [%s] ", label, choices)
		s, err := p.reader.ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))

		switch {
		case s == "":
			return defaultChoice
		case slices.Contains(yes, s):
			return true
		case slices.Contains(no, s):
			return false
		}

		// Unrecognized answer: re-prompt instead of silently taking the
		// default (the shell rejects invalid input). Give up on a closed/EOF
		// stream to avoid an infinite loop in non-interactive contexts.
		if err != nil {
			return defaultChoice
		}
		fmt.Fprintln(p.writer, "Invalid input. Please answer yes or no.")
	}
}

func getChoices(defaultChoice bool) string {
	if defaultChoice {
		return "Y/n"
	}
	return "y/N"
}
