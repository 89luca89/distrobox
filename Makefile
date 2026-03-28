GO_BUILD_ENV := CGO_ENABLED=0 GOOS=linux

.PHONY: build
build:
	$(GO_BUILD_ENV) go build -o ./bin/distrobox ./cmd/distrobox

.PHONY: test
test: vet
	$(GO_BUILD_ENV) go test -v ./...

.PHONY: vet
vet:
	$(GO_BUILD_ENV) go vet ./...

.PHONY: fmt
fmt:
	$(GO_BUILD_ENV) go fmt ./...

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

.PHONY: install
install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 ./bin/distrobox $(DESTDIR)$(BINDIR)/distrobox

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/distrobox

.PHONY: clean
clean:
	rm -f ./bin/distrobox

.PHONY: lint
lint:
	$(GO_BUILD_ENV) golangci-lint run --verbose

.PHONY: lint-fix
lint-fix:
	$(GO_BUILD_ENV) golangci-lint run --fix
