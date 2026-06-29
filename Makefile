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
