package cli

import "strings"

// findExecMarkerIndex returns the index of the first -e or --exec in
// args, or -1 if neither is present. It is used by the enter and
// ephemeral commands to recover the marker when the user placed it
// after the container name (or, for ephemeral, after the first
// positional arg) — in those cases urfave/cli leaves the marker in
// the positional tail instead of consuming it as a flag, and we need
// to know where the custom command starts.
func findExecMarkerIndex(args []string) int {
	for i, arg := range args {
		if arg == "-e" || arg == "--exec" {
			return i
		}
	}
	return -1
}

// extractAdditionalFlagsFromPositionals scans args for the first occurrence
// of --additional-flags or -a and returns the cleaned args (with the flag
// tokens removed) along with the extracted flag value. If no such flag is
// found it returns the original args and an empty string.
//
// This is needed because urfave/cli's StopOnNthArg leaves flag-like tokens
// in the positional tail, so --additional-flags placed after the container
// name is not consumed as a flag during normal parsing.
func extractAdditionalFlagsFromPositionals(args []string) ([]string, string) {
	for i := range args {
		switch arg := args[i]; {
		case arg == "--additional-flags" || arg == "-a":
			if i+1 < len(args) {
				// Space-separated value: --additional-flags VALUE
				flags := args[i+1]
				cleaned := make([]string, 0, len(args)-2)
				cleaned = append(cleaned, args[:i]...)
				cleaned = append(cleaned, args[i+2:]...)
				return cleaned, flags
			}
			// Flag at end with no value: remove it silently.
			cleaned := make([]string, 0, len(args)-1)
			cleaned = append(cleaned, args[:i]...)
			return cleaned, ""

		case strings.HasPrefix(arg, "--additional-flags="):
			// --additional-flags=VALUE
			v := strings.TrimPrefix(arg, "--additional-flags=")
			cleaned := make([]string, 0, len(args)-1)
			cleaned = append(cleaned, args[:i]...)
			cleaned = append(cleaned, args[i+1:]...)
			return cleaned, v

		case strings.HasPrefix(arg, "-a="):
			// -a=VALUE
			v := strings.TrimPrefix(arg, "-a=")
			cleaned := make([]string, 0, len(args)-1)
			cleaned = append(cleaned, args[:i]...)
			cleaned = append(cleaned, args[i+1:]...)
			return cleaned, v
		}
	}
	return args, ""
}
