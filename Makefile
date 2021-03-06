
## Copyright (c) 2020-2021 Detlef Stern
##
## This file is part of zettelstore.
##
## Zettelstore is licensed under the latest version of the EUPL (European Union
## Public License). Please see file LICENSE.txt for your rights and obligations
## under this license.

.PHONY:  check api build release clean

check:
	go run tools/build.go check

api:
	go run tools/build.go testapi

version:
	@echo $(shell go run tools/build.go version)

build:
	go run tools/build.go build

release:
	go run tools/build.go release

clean:
	go run tools/build.go clean
