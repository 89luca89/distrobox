package cli

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
