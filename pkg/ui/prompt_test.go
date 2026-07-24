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

package ui_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/89luca89/distrobox/pkg/ui"
)

func TestPrompt_YesReturnsTrue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("yes\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", false)

	assert.True(t, result)
}

func TestPrompt_YReturnsTrue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", false)

	assert.True(t, result)
}

func TestPrompt_NoReturnsFalse(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("no\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	assert.False(t, result)
}

func TestPrompt_NReturnsFalse(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("n\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	assert.False(t, result)
}

func TestPrompt_InvalidInputReturnsDefaultTrue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("maybe\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	assert.True(t, result)
}

func TestPrompt_InvalidInputReturnsDefaultFalse(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("maybe\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", false)

	assert.False(t, result)
}

func TestPrompt_EmptyInputReturnsDefault(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	assert.True(t, result)
}

func TestPrompt_WritesPromptToWriter(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	p.Prompt("Continue?", true)

	assert.Equal(t, "Continue? [Y/n] ", writer.String())
}

func TestPrompt_WritesPromptWithDefaultNo(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("n\n"))
	writer := &bytes.Buffer{}
	p := ui.NewPrompter(*reader, writer)

	p.Prompt("Delete file?", false)

	assert.Equal(t, "Delete file? [y/N] ", writer.String())
}
