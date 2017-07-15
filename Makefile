export GOPATH = $(CURDIR)/vendor:$(CURDIR)
GO = go
SOURCES = $(shell find src/ -name '*.go')

DESTDIR =
PREFIX = /usr
BIN_DIR = $(DESTDIR)$(PREFIX)/bin
CONF_DIR = $(DESTDIR)/etc/ctbuild
INSTALL = install
VERSION = $(shell git describe --tags --always)
BUILD_DATE = $(shell date -u +%Y-%m-%dT%H:%M:%S+0000)

default: bin/ctbuild

.PHONY: vendorize-dependencies
vendorize-dependencies:
	$(GO) get -d -t ctbuild/...
	python make-vendor-json.py
	find vendor/ -name .git -type d |xargs rm -rf

bin/ctbuild: $(SOURCES)
	$(GO) install -ldflags "-X main.VERSION=$(VERSION) -X main.BUILD_DATE=$(BUILD_DATE)"  ctbuild/...

install: bin/ctbuild
	mkdir -p $(BIN_DIR)

	$(INSTALL) -m 0755 -o root bin/ctbuild $(BIN_DIR)/ctbuild

test:
	$(GO) test ctbuild/...

clean:
	rm -rf bin/ vendor/pkg/ vendor/bin pkg/
