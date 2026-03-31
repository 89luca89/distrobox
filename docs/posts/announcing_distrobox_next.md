- [Distrobox](../README.md)
  - [Announcing the next generation of Distrobox](#announcing-the-next-generation-of-distrobox)
    - [Try it now](#try-it-now)
    - [Why we rewrote Distrobox](#why-we-rewrote-distrobox)
    - [Compatibility](#compatibility)
    - [During the transition](#during-the-transition)
    - [Contributing](#contributing)
    - [The many thanks we have to say](#the-many-thanks-we-have-to-say)

---

# Announcing the next generation of Distrobox

We're releasing Distrobox v2 to the public as a release candidate. This is a complete rewrite in Go. Distrobox v1
remains the stable version and we recommend using it in production for now.

The first objective is to reach feature parity between v2 and v1, at which point we can declare v2 stable. The source
code is available now on [the `next` branch](https://github.com/89luca89/distrobox/tree/next).

## Try it now

v2.0.0-rc releases are available on [GitHub](https://github.com/89luca89/distrobox/releases/tag/2.0.0-rc.1).

You can also build from source on the `next` branch:

```sh
git clone https://github.com/89luca89/distrobox.git
cd distrobox
git checkout next
make build
sudo make install
```

Please test it with your usual workflows and report any issues you find. Your feedback is essential to reach stability
quickly.

## Why we rewrote Distrobox

Shell's immediate feedback loop was critical to Distrobox's early success. But as the project matured, we hit its
limits: no proper module system for code reuse, no handy test engine, and patterns that are hard to maintain. We also
want to extend Distrobox to new use cases, which would have required a significant refactor of the existing codebase.

We chose Go because the core team is confident in it. It's popular with a short learning curve, so the community can
jump in and contribute. It has a solid toolchain and standard library that lets us keep external dependencies to a
minimum. And it's straightforward to build for multiple architectures—important for Distrobox's diverse user base.

We didn't start this effort to improve performance. But first benchmarks show a sensible performance increase on common
usage scenarios. More data to come.

## Compatibility

v2 maintains the same interface for CLI command arguments, manifest files, and configuration files. Your scripts and
`.distrobox` folders will work with v2.

Existing v1 containers work with v2, except for exported bins and apps—those containers must be recreated. v2 ships as
a single binary, so command-specific executables like `distrobox-enter` and `distrobox-create` no longer exist. Use
`distrobox enter`, `distrobox create`, etc. instead.

## During the transition

While v2 reaches feature parity and stability, we're making focused choices.

We do not accept new features on v1 nor v2 until v2 reaches feature parity with v1 and is declared stable. New features
would slow down that milestone. Bugfixes must be submitted against the `next` branch. We'll decide on backports to v1
case-by-case.

Before reporting a bug, check whether it's already fixed in v2. For already open PRs on v1, we'll decide case-by-case
with the authors. For open issues on v1, we ask that you verify whether the issue is present on v2 as well. We
prioritize fixing issues on v2 first. We'll consider backporting critical fixes to v1 if the issue makes Distrobox v1
unusable or insecure.

We're releasing v2.0.0-rc versions as we progress. Releases are available on GitHub and are published as needed, with
no fixed cadence. v2 will be declared stable when we can assert it covers all the use cases of v1 without relevant
regressions.

## Contributing

All contributions must be sent against the `next` branch. Please read the
[architecture document](distrobox_next_architecture.md) before contributing.

A working Go installation is required to build and test the project. Refer to the
[official Go documentation](https://go.dev/doc/install) to set up your local environment.

## The many thanks we have to say

A project like Distrobox would have gone nowhere without the support of its community. Over the years, we received
contributions from more than 200 developers; these people are first of all enthusiastic Distrobox users, and we cannot
be more grateful for that.

Some of them are now seeing their code disappear to make room for the rewrite. We want to emphasize that the rewrite
itself wouldn't have been possible without their contributions. Please take a moment to acknowledge the
[Distrobox contributors list](https://github.com/89luca89/distrobox/graphs/contributors) — to them go our warmest
thanks.

We're excited to see where v2 takes Distrobox, and we hope you are excited, too. Try it out, report bugs, and join the
discussion on [Matrix](https://matrix.to/#/%23distrobox:matrix.org) and [Telegram](https://t.me/distrobox_chat_new)
