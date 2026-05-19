package providers

import "time"

// Sleep durations used while waiting for a container's setup to complete.
const (
	// logsRetryInterval is the delay before retrying when fetching container
	// logs fails transiently during setup waiting.
	logsRetryInterval = 100 * time.Millisecond
	// setupPollInterval is the cadence at which container setup logs are
	// polled while waiting for the setup-done marker.
	setupPollInterval = 500 * time.Millisecond
)
