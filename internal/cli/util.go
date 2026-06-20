package cli

// ptr returns a pointer to v. Go has no built-in syntax to take the
// address of a literal, so we need this tiny helper to populate pointer
// fields like urfave/cli's StopOnNthArg (a *int) — the zero value of a
// pointer is nil, and nil keeps the default behaviour, so the call site
// has to be explicit about wanting a non-nil pointer.
func ptr[T any](v T) *T { return &v }
