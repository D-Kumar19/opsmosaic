GO ?= go
PREFIX ?= $(HOME)/.local
DESTDIR ?=

.PHONY: build install test test-install vet

build:
	$(GO) build -o bin/opsmosaic ./cmd/opsmosaic

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 bin/opsmosaic "$(DESTDIR)$(PREFIX)/bin/opsmosaic"
	@printf 'Installed: %s\n' "$(DESTDIR)$(PREFIX)/bin/opsmosaic"
	@printf 'Ensure %s is on PATH, then run: opsmosaic\n' "$(PREFIX)/bin"

test:
	$(GO) test -race ./...

test-install:
	sh scripts/test-install.sh

vet:
	$(GO) vet ./...
