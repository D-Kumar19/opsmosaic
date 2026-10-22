GO ?= go
PREFIX ?= $(HOME)/.local
DESTDIR ?=

.PHONY: build install test test-install vet ci release test-release demo test-demo test-signing test-shell

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

ci:
	GO="$(GO)" sh scripts/ci.sh

release:
	GO="$(GO)" sh scripts/release.sh "$(VERSION)"

test-release:
	GO="$(GO)" sh scripts/test-release.sh

demo:
	GO="$(GO)" python3 scripts/demo.py

test-demo:
	python3 scripts/test-demo.py

test-signing:
	sh scripts/test-signing.sh

test-shell:
	GO="$(GO)" python3 scripts/test-shell.py
