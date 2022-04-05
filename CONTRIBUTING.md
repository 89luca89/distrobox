# Contributing to Distrobox

We greatly appreciate your input! We want to make contributing to this project
as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features

## Creating a Pull Requests

Pull requests are the best way to propose changes to the codebase
We actively welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Any contributions you make will be under the GPLv3 Software License

In short, when you submit code changes, your submissions are understood to be
under the same [GPLv3 License](https://choosealicense.com/licenses/gpl-3.0/) that
covers the project.
Feel free to contact the maintainers if that's a concern.

## Suggestions

Suggestions are welcome, be sure:

- it is not already being discussed in the [issue tracker](https://github.com/89luca89/distrobox/issues)
  - If it has and is marked as OPEN, go ahead and share your own
    thoughts about the topic!
  - If it has and is marked as CLOSED, please read the ticket and depending on
    whether the suggestion was accepted or not consider if it is worth opening
    a new issue or not.
- Consider if the suggestion is not too out of scope of the project.
- Mark them with a [Suggestion] in the title

## Report bugs using Github's [issues](https://github.com/89luca89/distrobox/issues)

We use GitHub issues to track public bugs.
Report a bug by
[opening a new issue](https://github.com/89luca89/distrobox/issues); it's that easy!

### Write bug reports with detail, background, and sample code

**A good bug report** should have:

- Check that the bug is not already discussed in the [issue tracker](https://github.com/89luca89/distrobox/issues)
- See our [documentation](https://github.com/89luca89/distrobox/tree/main/docs)
  if there are some steps that could help you solve your issue
- Mark them with an [Error] in the title
- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Provide logs (terminal output, runs with verbose mode)
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you
  tried that didn't work)

## Use a Consistent Coding Style

- use `shellcheck` to check for posix compliance and bashisms using:
  - `shellcheck -s sh -o all -Cnever -Sstyle -a -f gcc -x`
  - install from: [HERE](https://github.com/koalaman/shellcheck)
    following [this](https://github.com/koalaman/shellcheck#installing)
- use `shfmt` to style the code using:
  - `shfmt -s`
  - install from [HERE](https://github.com/mvdan/sh) using `go install mvdan.cc/sh/v3/cmd/shfmt@latest`
- Legibility of the code is more important than code golfing, try to be
  expressive in the code
- Error checking is important! Ensure to LBYL (Look Before You Leap), check for
  variables and for code success exit codes
- Don't hesitate to comment your code! We're placing high importance on this to
  maintain the code readable and understandeable
- Update documentation to reflect your changes - Manual pages can be found in
  directory `docs`

If you are using Visual Studio Code, there are [plugins](https://marketplace.visualstudio.com/items?itemName=timonwong.shellcheck)
that include all this functionality and throw a warning if you're doing
something wrong.
If you are using Vim or Emacs there are plenty of linters and checkers that will
integrate with the 2 tools listed above.

## License

By contributing, you agree that your contributions will be licensed under
its GPLv3 License.

## References

This document was adapted from the open-source contribution guidelines
for [Facebook's Draft](https://github.com/facebook/draft-js/blob/a9316a723f9e918afde44dea68b5f9f39b7d9b00/CONTRIBUTING.md)
