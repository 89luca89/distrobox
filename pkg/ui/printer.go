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
