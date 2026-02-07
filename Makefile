# Makefile for distrobox
# SPDX-License-Identifier: GPL-3.0-only

.PHONY: help test test-integration clean

# Verbose mode: make test VERBOSE=1
VERBOSE ?= 0
ifeq ($(VERBOSE),1)
    Q =
else
    Q = @
endif

# Default target
help:
	@echo "Distrobox Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  help             - Show this help message"
	@echo "  test             - Run integration tests"
	@echo "  test-integration - Run integration tests (same as test)"
	@echo "  clean            - Clean up test artifacts"
	@echo ""
	@echo "Options:"
	@echo "  VERBOSE=1        - Show all commands and output (e.g., make test VERBOSE=1)"
	@echo ""

# Run integration tests
test: test-integration

# Run integration tests (works on Linux and macOS)
test-integration:
	$(Q)echo "Running distrobox integration tests..."
	$(Q)VERBOSE=$(VERBOSE) bash tests/macos/test-integration.sh

# Clean up test artifacts (temporary test directories only)
clean:
	$(Q)echo "Cleaning up test artifacts..."
	$(Q)find . -maxdepth 1 -type d -name '.test-tmp-*' -exec rm -rf {} +
	$(Q)echo "Clean complete"
