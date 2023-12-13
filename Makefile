
## Copyright (c) 2020-present Detlef Stern
##
## This file is part of Zettelstore.
##
## Zettelstore is licensed under the latest version of the EUPL (European Union
## Public License). Please see file LICENSE.txt for your rights and obligations
## under this license.

.PHONY:  check relcheck api version build release clean

check:
	go run tools/check/check.go

relcheck:
	go run tools/check/check.go -r

api:
	go run tools/testapi/testapi.go

version:
	@echo $(shell go run tools/build/build.go version)

build:
	go run tools/build/build.go build

release:
	go run tools/build/build.go release

clean:
	go run tools/clean/clean.go
