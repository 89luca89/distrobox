# Add --unshare-home flag for isolated containers

## What this does

Adds a new `--unshare-home` flag that creates containers without mounting your host home directory. The container gets its own isolated home instead.

## How it works

- New flag in `distrobox create --unshare-home`
- Container gets labeled so `distrobox enter` knows it's unshared
- Home directory mounting is skipped during creation
- Container creates its own home directory on first entry
- `--unshare-all` now includes this flag automatically

## Why you'd want this

I've been working with AI agents and realized I needed better isolation. These tools can be unpredictable and might access files they shouldn't. With `--unshare-home`, I can run potentially dangerous code knowing it can't touch my personal files, configs, or SSH keys.

It's also useful for testing untrusted software or creating clean development environments.

## Testing

I'll attach detailed testing docs in the next message showing how to verify this works correctly across different distros and use cases.
