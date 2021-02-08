
## Copyright (c) 2020-2021 Detlef Stern
##
## This file is part of zettelstore.
##
## Zettelstore is licensed under the latest version of the EUPL (European Union
## Public License). Please see file LICENSE.txt for your rights and obligations
## under this license.

.PHONY: test check validate race build release clean

test:
	go test ./...

check:
	go vet ./...
	~/go/bin/golint ./...

validate: test check

race:
	go test -race ./...

version:
	@echo $(shell go run tools/build.go version)

build:
	go run tools/build.go

release:
	go run tools/build.go release

clean:
	go run tools/build.go clean
