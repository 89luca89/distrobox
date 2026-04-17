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
