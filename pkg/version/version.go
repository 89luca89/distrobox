package version

// Version is set at build time via -ldflags.
var Version = "dev" //nolint:gochecknoglobals // default value for development builds
