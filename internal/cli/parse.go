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

package cli

import (
	"strings"

	"github.com/urfave/cli/v3"
)

// PrepareArgs lets `enter` and `ephemeral` accept distrobox flags after the
// container name, like the bash distrobox-enter (which only starts the command
// at -e/--exec/--). urfave's StopOnNthArg can't express this: it stops at the
// first positional and can't tell the name from the command. So we find the
// command ourselves and splice a bare "--" in front of it, then let urfave
// parse the rest. It runs at the entrypoint because urfave has no pre-parse hook.
func PrepareArgs(root *cli.Command, args []string) []string {
	if len(args) < 2 {
		return args
	}

	// Completion appends this flag last; splicing a "--" ahead of it would
	// hide it from urfave.
	if args[len(args)-1] == "--generate-shell-completion" {
		return args
	}

	// The sub-command can be preceded by global flags (e.g. --verbose).
	globalArity := flagArity(root.Flags)
	i := 1
	for i < len(args) && args[i] != "--" && isFlagToken(args[i]) {
		base, hasValue := flagBaseName(args[i])
		i++
		if globalArity[base] && !hasValue {
			i++ // also skip the flag's value
		}
	}
	if i >= len(args) {
		return args
	}

	sub := root.Command(args[i])
	if sub == nil || (sub.Name != "enter" && sub.Name != "ephemeral") {
		return args
	}

	// Scan with the sub-command's flags plus the inherited globals (--verbose).
	flags := append(append([]cli.Flag{}, sub.Flags...), root.Flags...)

	// enter takes the first bare token as the name, unless one is already
	// supplied via --name's default (DBX_CONTAINER_NAME/config). ephemeral has
	// no positional name, so its first bare token is already the command.
	nameSeen := true
	if sub.Name == "enter" {
		nameSeen = nameFlagDefault(sub.Flags) != ""
	}

	tail, help := splitExecCommand(flags, args[i+1:], nameSeen)

	out := append([]string{}, args[:i+1]...)
	if help {
		// urfave would treat the container name as a help topic ("No help
		// topic for '<name>'"), so drop it and just show the sub-command help.
		return append(out, "--help")
	}
	return append(out, tail...)
}

// splitExecCommand splices a bare "--" in front of the custom command so urfave
// hands it through untouched, and reports whether help was requested. Args are
// returned unchanged when there is no command — including an unrecognized flag,
// which is left for urfave to reject.
//
// nameSeen is true when a name is already set without a positional (ephemeral,
// or enter with DBX_CONTAINER_NAME), so the first bare token is the command.
func splitExecCommand(flags []cli.Flag, args []string, nameSeen bool) ([]string, bool) {
	arity := flagArity(flags)
	for i := 0; i < len(args); {
		arg := args[i]
		switch {
		case arg == "--": // urfave already isolates the command here
			return args, false

		case isExecMarker(arg):
			// -e/--exec open the command, but before the name they are a
			// harmless no-op flag that urfave consumes.
			if nameSeen {
				return spliceSeparator(args, i, true), false
			}
			i++

		case isHelpToken(arg):
			return nil, true

		case isFlagToken(arg):
			base, hasValue := flagBaseName(arg)
			takesValue, known := arity[base]
			if !known {
				return args, false
			}
			if base == "name" || base == "n" {
				nameSeen = true
			}
			i++
			if takesValue && !hasValue {
				i++ // skip the flag's value
			}

		case !nameSeen:
			nameSeen = true // the first bare token is the container name
			i++

		default:
			return spliceSeparator(args, i, false), false // command starts here
		}
	}
	return args, false
}

// spliceSeparator inserts a "--" at index i so the args from there on are the
// verbatim command. dropMarker replaces the token at i (an -e/--exec marker)
// rather than keeping it, since the marker itself is not part of the command.
func spliceSeparator(args []string, i int, dropMarker bool) []string {
	out := append([]string{}, args[:i]...)
	out = append(out, "--")
	if dropMarker {
		return append(out, args[i+1:]...)
	}
	return append(out, args[i:]...)
}

// flagArity maps each flag name (long and short) to whether it consumes a
// following value, so the scanner can skip that value instead of mistaking it
// for the container name or command.
func flagArity(flags []cli.Flag) map[string]bool {
	arity := make(map[string]bool)
	for _, f := range flags {
		takesValue := false
		if dg, ok := f.(cli.DocGenerationFlag); ok {
			takesValue = dg.TakesValue()
		}
		for _, name := range f.Names() {
			arity[name] = takesValue
		}
	}
	return arity
}

// nameFlagDefault returns the --name flag's default value, which config/env
// (DBX_CONTAINER_NAME) populate, or "" if unset.
func nameFlagDefault(flags []cli.Flag) string {
	for _, f := range flags {
		sf, ok := f.(*cli.StringFlag)
		if ok && sf.Name == "name" {
			return sf.Value
		}
	}
	return ""
}

// flagBaseName strips leading dashes and any "=value" suffix, returning the
// bare flag name and whether an inline value was present.
func flagBaseName(tok string) (string, bool) {
	base := strings.TrimLeft(tok, "-")
	if name, _, found := strings.Cut(base, "="); found {
		return name, true
	}
	return base, false
}

func isExecMarker(tok string) bool { return tok == "-e" || tok == "--exec" }

func isHelpToken(tok string) bool {
	base, _ := flagBaseName(tok)
	return isFlagToken(tok) && (base == "help" || base == "h")
}

// isFlagToken reports whether tok is a flag: a dash followed by another dash or
// a letter. "-", "--" and negative numbers ("-9") are not flags.
func isFlagToken(tok string) bool {
	return len(tok) > 1 && tok[0] == '-' && (tok[1] == '-' || isASCIILetter(tok[1]))
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
