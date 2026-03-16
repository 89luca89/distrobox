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

.PHONY: clean
clean:
	rm -f ./bin/distrobox

.PHONY: lint
lint:
	$(GO_BUILD_ENV) golangci-lint run --verbose

.PHONY: lint-fix
lint-fix:
	$(GO_BUILD_ENV) golangci-lint run --fix
