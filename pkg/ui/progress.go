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
