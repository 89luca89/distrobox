GOOS ?= $(shell go env GOOS)
GO_BUILD_ENV := CGO_ENABLED=0 GOOS=$(GOOS)
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/89luca89/distrobox/pkg/version.Version=$(VERSION)"

BINDIR := ./bin

# Go source files — make will rebuild when any .go changes
GO_SRC := $(shell find . -type f -name '*.go' -not -path './vendor/*')

# Asset files copied from internal/inside-distrobox/assets
ASSET_SRCDIR := internal/inside-distrobox/assets
ASSETS := $(notdir $(wildcard $(ASSET_SRCDIR)/*))
ASSET_TARGETS := $(addprefix $(BINDIR)/,$(ASSETS))

# Commands that need POSIX shim wrappers (map to "distrobox <subcommand>")
SHIM_COMMANDS := assemble create enter ephemeral generate-entry list rm stop upgrade
SHIM_TARGETS := $(addprefix $(BINDIR)/distrobox-,$(SHIM_COMMANDS))

# All generated files in $(BINDIR)
BIN_FILES := $(BINDIR)/distrobox $(ASSET_TARGETS) $(SHIM_TARGETS)

.PHONY: build
build: $(BIN_FILES)

$(BINDIR):
	mkdir -p $(BINDIR)

$(BINDIR)/distrobox: go.mod go.sum $(GO_SRC) $(wildcard $(ASSET_SRCDIR)/*) | $(BINDIR)
	$(GO_BUILD_ENV) go build $(LDFLAGS) -o $@ ./cmd/distrobox

# Copy asset files from internal/inside-distrobox/assets
$(ASSET_TARGETS): $(BINDIR)/%: $(ASSET_SRCDIR)/% | $(BINDIR)
	@cp -f $< $@; \
	chmod 755 $@; \
	printf '  COPY    %s\n' "$@"

# Generate shim scripts for each subcommand.
# Regenerate when the binary changes (new subcommands may have been added).
$(SHIM_TARGETS): $(BINDIR)/distrobox-%: | $(BINDIR)/distrobox $(BINDIR)
	@cmd=$$(echo $@ | sed 's|.*/distrobox-||'); \
	printf '  SHIM    %s\n' "$@"; \
	{ \
		echo '#!/bin/sh'; \
		echo '# SPDX-License-Identifier: GPL-3.0-only'; \
		echo '#'; \
		echo '# This file is part of the distrobox project:'; \
		echo '#    https://github.com/89luca89/distrobox'; \
		echo '#'; \
		echo '# Copyright (C) 2021 distrobox contributors'; \
		echo '#'; \
		echo '# distrobox is free software; you can redistribute it and/or modify it'; \
		echo '# under the terms of the GNU General Public License version 3'; \
		echo '# as published by the Free Software Foundation.'; \
		echo '#'; \
		echo '# distrobox is distributed in the hope that it will be useful, but'; \
		echo '# WITHOUT ANY WARRANTY; without even the implied warranty of'; \
		echo '# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU'; \
		echo '# General Public License for more details.'; \
		echo '#'; \
		echo '# You should have received a copy of the GNU General Public License'; \
		echo '# along with distrobox; if not, see <http://www.gnu.org/licenses/>.'; \
		echo ''; \
		echo 'exec "$$(dirname "$$0")/distrobox" "'$${cmd}'" "$$@"'; \
	} > $@; \
	chmod 755 $@ 

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

.PHONY: install
install: build
	install -d $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR) $(DESTDIR)$(BASHCOMPDIR) $(DESTDIR)$(ZSHCOMPDIR)
	install -m 0755 $(BINDIR)/distrobox $(DESTDIR)$(BINDIR)/distrobox
	for f in $(ASSET_TARGETS) $(SHIM_TARGETS); do \
		install -m 0755 "$$f" $(DESTDIR)$(BINDIR)/; \
	done
	install -m 0644 man/man1/*.1 $(DESTDIR)$(MANDIR)/
	install -m 0644 completions/bash/distrobox $(DESTDIR)$(BASHCOMPDIR)/distrobox
	install -m 0644 completions/zsh/_distrobox $(DESTDIR)$(ZSHCOMPDIR)/_distrobox
	install -d $(DESTDIR)$(ICONDIR)/scalable/apps
	install -m 0644 icons/terminal-distrobox-icon.svg $(DESTDIR)$(ICONDIR)/scalable/apps/
	for sz in $(ICON_SIZES); do \
		install -d $(DESTDIR)$(ICONDIR)/$${sz}x$${sz}/apps; \
		install -m 0644 icons/hicolor/$${sz}x$${sz}/apps/terminal-distrobox-icon.png \
			$(DESTDIR)$(ICONDIR)/$${sz}x$${sz}/apps/; \
	done

.PHONY: uninstall
uninstall:
	for f in $(BIN_FILES); do \
		rm -f $(DESTDIR)$(BINDIR)/"$$(basename "$$f")"; \
	done
	rm -f $(DESTDIR)$(MANDIR)/distrobox.1 $(DESTDIR)$(MANDIR)/distrobox-*.1
	rm -f $(DESTDIR)$(BASHCOMPDIR)/distrobox
	rm -f $(DESTDIR)$(ZSHCOMPDIR)/_distrobox
	rm -f $(DESTDIR)$(ICONDIR)/scalable/apps/terminal-distrobox-icon.svg
	for sz in $(ICON_SIZES); do \
		rm -f $(DESTDIR)$(ICONDIR)/$${sz}x$${sz}/apps/terminal-distrobox-icon.png; \
	done

.PHONY: clean
clean:
	rm -f $(BIN_FILES)

.PHONY: lint
lint:
	$(GO_BUILD_ENV) golangci-lint run --verbose

.PHONY: lint-fix
lint-fix:
	$(GO_BUILD_ENV) golangci-lint run --fix
