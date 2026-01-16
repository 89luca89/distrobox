# Distrobox Integration Tests

This directory contains integration tests for distrobox that validate core functionality across platforms.

## What These Tests Do

The tests validate that distrobox works correctly by testing:

1. **Basic commands** - Help flags work for all main commands
2. **Container lifecycle** - Create, start, stop, and remove containers
3. **Container entry** - Enter containers and execute commands
4. **File system mounting** - Home directory is mounted and accessible
5. **Bidirectional file access** - Files created on host are readable in container and vice versa
6. **Container management** - List, stop, and remove operations work correctly

## Why These Tests Exist

These tests were created to validate macOS support for distrobox. On macOS, distrobox host-side scripts needed modifications to work with macOS-specific commands (e.g., `dscl` instead of `getent`). These tests ensure:

- The macOS compatibility changes work correctly on macOS
- The changes don't break existing Linux functionality
- Core distrobox operations work reliably across platforms

## How to Run

### Using Make (Recommended)

```bash
# Run all tests (quiet mode)
make test

# Run tests with verbose output (shows all commands)
make test VERBOSE=1

# Run integration tests explicitly
make test-integration

# Clean up test artifacts
make clean

# Clean with verbose output
make clean VERBOSE=1
```

The `VERBOSE=1` option shows all commands being executed and their full output, which is useful for debugging test failures or understanding what the tests are doing.

### Running Tests Directly

```bash
# From the repository root
bash tests/macos/test-integration.sh

# Or make it executable and run
chmod +x tests/macos/test-integration.sh
./tests/macos/test-integration.sh
```

### Environment Variables

You can customize test behavior with these environment variables:

- `DISTROBOX_TEST_IMAGE` - Container image to use (default: `ubuntu:22.04`)
- `DISTROBOX_PATH` - Path to distrobox scripts (auto-detected by default)

Example:
```bash
DISTROBOX_TEST_IMAGE=fedora:39 make test
```

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed

## Requirements

- Either podman or docker must be installed
- Bash shell
- Internet connection (for pulling container images on first run)

## Test Output

The test script provides colored output:
- **GREEN** - Test passed
- **RED** - Test failed
- **YELLOW** - Informational messages

Failed tests will show the error output to help diagnose issues.

## CI/CD Integration

These tests are designed to run in CI/CD pipelines. The proper exit codes ensure pipeline failures when tests fail:

```yaml
# Example GitHub Actions usage
- name: Run distrobox tests
  run: make test
```

## Troubleshooting

### Container Creation Fails

If container creation fails, check:
- Is podman/docker installed and running?
- Can you pull images? Try: `podman pull ubuntu:22.04`
- Do you have enough disk space?

### Container Entry Fails

If entering the container fails:
- Wait a few seconds and try again (container may be initializing)
- Check container status: `./distrobox-list`
- Look at container logs: `podman logs <container-name>`

### Tests Leave Containers Behind

The tests automatically clean up containers in their cleanup handler. If tests are interrupted (e.g., Ctrl+C), you may need to manually remove test containers:

```bash
# List containers
./distrobox-list

# Remove test containers
./distrobox-rm distrobox-test-XXXXX --force
```

Or run the clean target:
```bash
make clean
```

## Adding New Tests

To add new tests:

1. Add a new `run_test` or `run_test_with_output` call in `test-integration.sh`
2. Follow the existing pattern for test naming and structure
3. Ensure tests clean up after themselves
4. Test on both Linux and macOS if possible

Example:
```bash
# Test that some command works
run_test \
    "Description of what this tests" \
    "'${DISTROBOX_PATH}/distrobox-command' args"

# Test that output matches expected pattern
run_test_with_output \
    "Description of test" \
    "'${DISTROBOX_PATH}/distrobox-command' args" \
    "expected-output-pattern"
```
