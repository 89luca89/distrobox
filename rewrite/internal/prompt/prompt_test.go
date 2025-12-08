package prompt_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/89luca89/distrobox/internal/prompt"
)

func TestPrompt_YesReturnsTrue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("yes\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", false)

	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestPrompt_YReturnsTrue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", false)

	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestPrompt_NoReturnsFalse(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("no\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestPrompt_NReturnsFalse(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("n\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestPrompt_InvalidInputReturnsDefaultTrue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("maybe\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	if result != true {
		t.Errorf("expected true (default), got %v", result)
	}
}

func TestPrompt_InvalidInputReturnsDefaultFalse(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("maybe\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", false)

	if result != false {
		t.Errorf("expected false (default), got %v", result)
	}
}

func TestPrompt_EmptyInputReturnsDefault(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	result := p.Prompt("Continue?", true)

	if result != true {
		t.Errorf("expected true (default), got %v", result)
	}
}

func TestPrompt_WritesPromptToWriter(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	p.Prompt("Continue?", true)

	expected := "Continue? [Y/n] "
	if writer.String() != expected {
		t.Errorf("expected %q, got %q", expected, writer.String())
	}
}

func TestPrompt_WritesPromptWithDefaultNo(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("n\n"))
	writer := &bytes.Buffer{}
	p := prompt.NewPrompter(*reader, writer)

	p.Prompt("Delete file?", false)

	expected := "Delete file? [y/N] "
	if writer.String() != expected {
		t.Errorf("expected %q, got %q", expected, writer.String())
	}
}
