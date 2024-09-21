<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="static/assets/brand/png/distrobox-dark.png" />
    <img alt="Distrobox logo" src="static/assets/brand/png/distrobox-light.png" width="400" height="200" />
  </picture>
</p>

<p align="center">
  <strong>Use any linux distribution inside your terminal.</strong>
</p>

---

[![Lint](https://github.com/89luca89/distrobox/actions/workflows/main.yml/badge.svg)](https://github.com/89luca89/distrobox/actions/workflows/main.yml)
[![CI](https://github.com/89luca89/distrobox/actions/workflows/compatibility.yml/badge.svg)](https://github.com/89luca89/distrobox/actions/workflows/compatibility.yml)
[![GitHub](https://img.shields.io/github/license/89luca89/distrobox?color=blue)](https://github.com/89luca89/distrobox/blob/main/COPYING.md)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/89luca89/distrobox)](https://github.com/89luca89/distrobox/releases/latest)
[![Packaging status](https://repology.org/badge/tiny-repos/distrobox.svg)](https://repology.org/project/distrobox/versions)
[![GitHub issues by-label](https://img.shields.io/github/issues-search/89luca89/distrobox?query=is%3Aissue%20is%3Aopen%20label%3Abug%20-label%3Await-on-user%20&label=Open%20Bug%20Reports&color=red)](https://github.com/89luca89/distrobox/issues?q=is%3Aissue+is%3Aopen+label%3Abug+-label%3Await-on-user)

[Matrix Room](https://matrix.to/#/%23distrobox:matrix.org) - [Telegram Group](https://t.me/distrobox)

Use any Linux distribution inside your terminal. Enable both backward and forward
compatibility with software and freedom to use whatever distribution you’re more comfortable with.

Distrobox uses `podman`, `docker` or [`lilipod`](https://github.com/89luca89/lilipod) to create containers
using the Linux distribution of your choice. The created container will be tightly integrated with the
host, allowing sharing of the HOME directory of the user, external storage, external USB devices and
graphical apps (X11/Wayland), and audio.

![overview](https://user-images.githubusercontent.com/598882/144294862-f6684334-ccf4-4e5e-85f8-1d66210a0fff.png)

## Documentation

Documentation for the [latest release](https://github.com/89luca89/distrobox/releases/latest) is available
over at [distrobox.it](https://distrobox.it). Documentation on GitHub strictly refers to the code in the
main branch and is not optimized for being viewed without building it as the website.
