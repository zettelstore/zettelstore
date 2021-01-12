
## Copyright (c) 2020 Detlef Stern
##
## This file is part of zettelstore.
##
## Zettelstore is licensed under the latest version of the EUPL (European Union
## Public License). Please see file LICENSE.txt for your rights and obligations
## under this license.

.PHONY: test check validate race run build build-dev release clean

PACKAGE := zettelstore.de/z/cmd/zettelstore

GO_LDFLAG_VERSION := -X main.buildVersion=$(shell go run tools/version.go || echo unknown)
GOFLAGS_DEVELOP := -ldflags "$(GO_LDFLAG_VERSION)" -tags osusergo,netgo
GOFLAGS_RELEASE := -ldflags "$(GO_LDFLAG_VERSION) -w" -tags osusergo,netgo

test:
	go test ./...

check:
	go vet ./...
	~/go/bin/golint ./...

validate: test check

race:
	go test -race ./...

build-dev:
	mkdir -p bin
	go build $(GOFLAGS_DEVELOP) -o bin/zettelstore $(PACKAGE)

build:
	mkdir -p bin
	go build $(GOFLAGS_RELEASE) -o bin/zettelstore $(PACKAGE)

release:
	mkdir -p releases
	GOARCH=amd64 GOOS=linux go build $(GOFLAGS_RELEASE) -o releases/zettelstore $(PACKAGE)
	GOARCH=arm GOARM=6 GOOS=linux go build $(GOFLAGS_RELEASE) -o releases/zettelstore-arm6 $(PACKAGE)
	GOARCH=amd64 GOOS=darwin go build $(GOFLAGS_RELEASE) -o releases/iZettelstore $(PACKAGE)
	GOARCH=amd64 GOOS=windows go build $(GOFLAGS_RELEASE) -o releases/zettelstore.exe $(PACKAGE)

clean:
	rm -rf bin releases
