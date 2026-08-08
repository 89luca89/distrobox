GOOS ?= $(shell go env GOOS)
GO_BUILD_ENV := CGO_ENABLED=0 GOOS=$(GOOS)
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/89luca89/distrobox/pkg/version.Version=$(VERSION)"

.PHONY: build
build:
	$(GO_BUILD_ENV) go build $(LDFLAGS) -o ./bin/distrobox ./cmd/distrobox

# Regenerate man pages from docs/usage/*.md via pandoc.
# Requires pandoc on $PATH; run after touching docs/usage/.
.PHONY: man
man:
	./man/gen-man

.PHONY: test
test: vet
	$(GO_BUILD_ENV) go test -v ./...

.PHONY: vet
vet:
	$(GO_BUILD_ENV) go vet ./...

.PHONY: fmt
fmt:
	$(GO_BUILD_ENV) go fmt ./...

PREFIX      ?= /usr/local
BINDIR      ?= $(PREFIX)/bin
MANDIR      ?= $(PREFIX)/share/man/man1
BASHCOMPDIR ?= $(PREFIX)/share/bash-completion/completions
ZSHCOMPDIR  ?= $(PREFIX)/share/zsh/site-functions
ICONDIR     ?= $(PREFIX)/share/icons/hicolor

ICON_SIZES := 16 22 24 32 36 48 64 72 96 128 256

V1_SUBCOMMANDS := assemble create enter ephemeral generate-entry ls list rm stop upgrade

.PHONY: install
install: build
	install -d $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR) $(DESTDIR)$(BASHCOMPDIR) $(DESTDIR)$(ZSHCOMPDIR)
	install -m 0755 ./bin/distrobox $(DESTDIR)$(BINDIR)/distrobox
	install -m 0644 man/man1/*.1 $(DESTDIR)$(MANDIR)/
	install -m 0644 completions/bash/distrobox $(DESTDIR)$(BASHCOMPDIR)/distrobox
	install -m 0644 completions/zsh/_distrobox $(DESTDIR)$(ZSHCOMPDIR)/_distrobox
	for sub in $(V1_SUBCOMMANDS); do \
		ln -sf distrobox $(DESTDIR)$(BINDIR)/distrobox-$${sub}; \
	done
	install -m 0755 internal/inside-distrobox/assets/distrobox-init      $(DESTDIR)$(BINDIR)/distrobox-init
	install -m 0755 internal/inside-distrobox/assets/distrobox-export    $(DESTDIR)$(BINDIR)/distrobox-export
	install -m 0755 internal/inside-distrobox/assets/distrobox-host-exec $(DESTDIR)$(BINDIR)/distrobox-host-exec
	install -d $(DESTDIR)$(ICONDIR)/scalable/apps
	install -m 0644 icons/terminal-distrobox-icon.svg $(DESTDIR)$(ICONDIR)/scalable/apps/
	for sz in $(ICON_SIZES); do \
		install -d $(DESTDIR)$(ICONDIR)/$${sz}x$${sz}/apps; \
		install -m 0644 icons/hicolor/$${sz}x$${sz}/apps/terminal-distrobox-icon.png \
			$(DESTDIR)$(ICONDIR)/$${sz}x$${sz}/apps/; \
	done

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/distrobox $(DESTDIR)$(BINDIR)/distrobox-*
	rm -f $(DESTDIR)$(MANDIR)/distrobox.1 $(DESTDIR)$(MANDIR)/distrobox-*.1
	rm -f $(DESTDIR)$(BASHCOMPDIR)/distrobox
	rm -f $(DESTDIR)$(ZSHCOMPDIR)/_distrobox
	rm -f $(DESTDIR)$(ICONDIR)/scalable/apps/terminal-distrobox-icon.svg
	for sz in $(ICON_SIZES); do \
		rm -f $(DESTDIR)$(ICONDIR)/$${sz}x$${sz}/apps/terminal-distrobox-icon.png; \
	done

.PHONY: clean
clean:
	rm -f ./bin/distrobox

.PHONY: lint
lint:
	$(GO_BUILD_ENV) golangci-lint run --verbose

.PHONY: lint-fix
lint-fix:
	$(GO_BUILD_ENV) golangci-lint run --fix

# ---------------------------------------------------------------------------
# Local e2e via the hack/ci/*.sh scripts, on the freshly-built binary. Pulled
# images are kept so repeated runs do not re-download. Targets:
#   e2e          mirrors the per-PR gate (.github/workflows/e2e.yml): the gating
#                image subset + commands.sh, on both backends (podman, docker).
#   e2e-full     the whole compatibility matrix (docs/compatibility.md) + commands.
#   e2e-one      one e2e.sh run for a single image + backend; override IMAGE=/CM=.
#   e2e-commands just commands.sh on one image/backend; override IMAGE=/CM=.
# ---------------------------------------------------------------------------
IMAGE ?= docker.io/library/alpine:latest
CM    ?= podman
E2E_ENV := DBX="$(CURDIR)/bin/distrobox" DBX_E2E_KEEP_IMAGE=1

E2E_IMAGES := \
	docker.io/library/alpine:latest \
	docker.io/library/debian:stable \
	docker.io/library/ubuntu:24.04 \
	docker.io/library/fedora:latest \
	registry.opensuse.org/opensuse/distrobox:latest \
	docker.io/library/archlinux:latest \
	registry.access.redhat.com/ubi9/ubi-init

# Full compatibility matrix, derived from docs/compatibility.md like compatibility.yml
# (no drift). Lazy '=' so the pipeline runs only when e2e-full is invoked.
E2E_FULL_SKIP := bazzite|chimera|slackware|stream8|ublue|neon|steamos
E2E_FULL_IMAGES = $(shell sed -n -e '/| Alma/,/| Void/ p' docs/compatibility.md | cut -d'|' -f 4 | sed 's/<br>/\n/g' | tr -d ' ' | sed '/^[[:space:]]*$$/d' | sort -u | grep -Ev '$(E2E_FULL_SKIP)')

# print-<VAR> — echo a make variable; the CI matrices build from these lists.
.PHONY: print-%
print-%:
	@echo '$($*)'

.PHONY: e2e-one
e2e-one: build
	echo ">>> e2e: $(IMAGE) / $(CM)"; \
	$(E2E_ENV) hack/ci/e2e.sh "$(IMAGE)" "$(CM)"

.PHONY: e2e-commands
e2e-commands: build
	echo ">>> commands: $(IMAGE) / $(CM)"; \
	$(E2E_ENV) hack/ci/commands.sh "$(IMAGE)" "$(CM)"

.PHONY: e2e
e2e: build
	@for img in $(E2E_IMAGES); do \
		for cm in podman docker; do \
			$(MAKE) --no-print-directory e2e-one IMAGE="$$img" CM="$$cm" || exit 1; \
		done; \
	done
	@for cm in podman docker; do \
		$(MAKE) --no-print-directory e2e-commands CM="$$cm" || exit 1; \
	done

# Full compatibility sweep (100+ images x 2 backends); local twin of compatibility.yml.
.PHONY: e2e-full
e2e-full:
	$(MAKE) --no-print-directory e2e E2E_IMAGES="$(E2E_FULL_IMAGES)"
