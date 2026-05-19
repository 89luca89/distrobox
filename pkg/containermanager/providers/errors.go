package providers

import "strings"

// isContainerNotFoundError reports whether the given error originates from a
// container manager (docker or podman) complaining about a missing container.
//
// Docker emits stderr like:
//
//	Error: No such object: <name>
//	Error: No such container: <name>
//
// Podman emits stderr like:
//
//	Error: inspecting object: no such container "<name>"
//	Error: no container with name or ID "<name>" found: no such container
//
// The wrapped error from run() carries the captured stderr in its message, so
// we match case-insensitively on the well-known fragments common to both
// runtimes. A missing daemon, permission error, or other failure will not
// match and is bubbled up unchanged.
func isContainerNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such container") ||
		strings.Contains(msg, "no such object")
}
