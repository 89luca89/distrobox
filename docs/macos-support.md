# macOS Support

Distrobox now supports macOS as a host platform when using Podman or Docker Desktop.

## How It Works

On macOS, Podman and Docker run containers inside a Linux VM. This means:

- **Host-side scripts** run on macOS and need macOS-compatible commands
- **Container-side code** (distrobox-init) runs inside the Linux VM and works unchanged
- Home directory and other mounts are passed through the VM layer automatically

## Implementation Details

### OS Detection

All host-side scripts now detect the operating system:

```bash
host_os="$(uname -s)"
```

This returns `"Darwin"` on macOS and `"Linux"` on Linux systems.

### User Information Retrieval

On Linux, distrobox uses `getent` to retrieve user information. On macOS, `getent` is not available, so we use `dscl` (Directory Service Command Line utility) instead:

**Linux (getent):**
```bash
HOME="$(getent passwd "${USER}" | cut -d':' -f6)"
SHELL="$(getent passwd "${USER}" | cut -d':' -f7)"
```

**macOS (dscl):**
```bash
HOME="$(dscl . -read "/Users/${USER}" NFSHomeDirectory 2>/dev/null | awk '{print $2}')"
SHELL="$(dscl . -read "/Users/${USER}" UserShell 2>/dev/null | awk '{print $2}')"
```

With fallbacks:
```bash
[ -z "${HOME}" ] && HOME="$(eval echo ~"${USER}")"
[ -z "${SHELL}" ] && SHELL="/bin/bash"
```

### Mount Point Handling

The `findmnt` command is Linux-specific and not available on macOS. In `distrobox-create`, the mount logic now skips `findmnt` on macOS:

```bash
if echo "${container_manager}" | grep -q "podman" &&
   ${container_manager} info 2> /dev/null | grep -q runc > /dev/null 2>&1 &&
   [ "${host_os}" != "Darwin" ]; then
    # Use findmnt to detect read-only mounts on Linux
    ...
else
    # Simple mount for macOS/crun/docker
    result_command="${result_command}
        --volume /:/run/host/:rslave"
fi
```

### /tmp Directory Handling

On macOS, `/tmp` is a symlink to `/private/tmp`. To avoid mount issues, `distrobox-create` mounts the real directory on macOS:

```bash
# On macOS, /tmp is a symlink to /private/tmp, so mount the real directory
if [ "${host_os}" = "Darwin" ]; then
    tmp_mount="--volume /private/tmp:/tmp:rslave"
else
    tmp_mount="--volume /tmp:/tmp:rslave"
fi
```

This ensures `/tmp` is fully functional for bidirectional file access between host and container.

## Modified Files

The following host-side scripts were updated for macOS compatibility:

1. `distrobox-create` - OS detection, getent replacement, findmnt skip, /tmp mount fix
2. `distrobox-enter` - OS detection, getent replacement
3. `distrobox-list` - OS detection, getent replacement
4. `distrobox-rm` - OS detection, getent replacement
5. `distrobox-stop` - OS detection, getent replacement
6. `distrobox-assemble` - OS detection, getent replacement
7. `distrobox-upgrade` - OS detection, getent replacement
8. `distrobox-ephemeral` - OS detection, getent replacement
9. `distrobox-generate-entry` - OS detection, getent replacement
10. `distrobox-export` - OS detection, getent replacement
11. `distrobox-host-exec` - OS detection, getent replacement

## Requirements

- macOS 10.15 or later
- Podman Desktop or Docker Desktop installed and running
- Bash shell

## Known Limitations

### Container Creation Error Message

When creating containers on macOS, you may see this error message:

```
Error: 2 errors occurred:
	* copying to host: copier: put: error resolving "/tmp": open /tmp: too many levels of symbolic links
	* copying from container: io: read/write on closed pipe
```

This is a harmless error from Podman's internal operations dealing with the `/tmp` symlink on macOS. **The container is created successfully and `/tmp` is fully functional** - you can create and access files in `/tmp` from both the host and container. This error can be safely ignored.

### Container Manager

Podman on macOS uses a Linux VM under the hood. This adds a small performance overhead compared to running on Linux directly.

## Testing

Integration tests validate that distrobox works correctly on macOS. Run tests with:

```bash
make test
```

See [tests/macos/README.md](../tests/macos/README.md) for more information about the test suite.

## Compatibility Matrix

| Feature | Linux | macOS |
|---------|-------|-------|
| Create containers | ✅ | ✅ |
| Enter containers | ✅ | ✅ |
| Home directory mount | ✅ | ✅ |
| Command execution | ✅ | ✅ |
| List containers | ✅ | ✅ |
| Stop containers | ✅ | ✅ |
| Remove containers | ✅ | ✅ |
| Export apps | ✅ | ✅ |
| Export binaries | ✅ | ✅ |
| Host exec | ✅ | ✅ |
| Init system (systemd) | ✅ | ⚠️ (Limited) |
| GPU passthrough (NVIDIA) | ✅ | ❌ |

## Troubleshooting

### Command Not Found: podman/docker

Make sure Podman Desktop or Docker Desktop is installed and the command-line tools are in your PATH:

```bash
# Check if podman is available
which podman

# Or check for docker
which docker
```

### HOME or SHELL Not Detected

If distrobox fails to detect your HOME or SHELL:

1. Verify dscl works:
   ```bash
   dscl . -read "/Users/${USER}" NFSHomeDirectory
   dscl . -read "/Users/${USER}" UserShell
   ```

2. Check environment variables:
   ```bash
   echo $HOME
   echo $SHELL
   ```

The fallback logic should handle most cases, but if issues persist, set these variables explicitly before running distrobox commands.

### Container Fails to Start

Check the Podman/Docker VM status:

```bash
# For Podman
podman machine list
podman machine start

# For Docker Desktop
# Ensure Docker Desktop app is running
```

## Development Notes

### Code Pattern

When adding new features to distrobox, follow this pattern for cross-platform compatibility:

```bash
# Early in the script, detect OS
host_os="$(uname -s)"

# When retrieving user info, branch on OS
if [ -z "${HOME}" ]; then
    if [ "${host_os}" = "Darwin" ]; then
        HOME="$(dscl . -read "/Users/${USER}" NFSHomeDirectory 2>/dev/null | awk '{print $2}')"
        [ -z "${HOME}" ] && HOME="$(eval echo ~"${USER}")"
    else
        HOME="$(getent passwd "${USER}" | cut -d':' -f6)"
    fi
fi
```

### Testing on Multiple Platforms

When modifying host-side scripts:

1. Test on Linux to ensure no regressions
2. Test on macOS to verify compatibility
3. Run the integration test suite: `make test`

### Container-Side Code

Code running inside containers (like `distrobox-init`) doesn't need macOS-specific changes since it always runs in a Linux environment.
