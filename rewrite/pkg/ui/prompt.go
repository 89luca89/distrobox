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

	var s string

	fmt.Fprintf(p.writer, "%s [%s] ", label, choices)
	s, _ = p.reader.ReadString('\n')
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	yes := []string{"y", "yes", "Y", "YES"}
	no := []string{"n", "no", "N", "NO"}

	if slices.Contains(yes, s) {
		return true
	}
	if slices.Contains(no, s) {
		return false
	}

	return defaultChoice
}

func getChoices(defaultChoice bool) string {
	if defaultChoice {
		return "Y/n"
	}
	return "y/N"
}
