# Copyright (c) 2021 by library authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY: build
build:
	go build ${LDFLAGS} -v -o target/tcgstorage $(CURDIR)/cmd/tcgstorage

.PHONY: build-release
build-release: build-release-amd64 build-release-arm64

.PHONY: build-release-amd64
build-release-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o=tcgstorage.linux.amd64 $(CURDIR)/cmd/tcgstorage

.PHONY: build-release-arm64
build-release-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o=tcgstorage.linux.arm64 $(CURDIR)/cmd/tcgstorage

.PHONY: test
test:
	go test -v ./...

.PHONY: get-dependencies
get-dependencies:
	go get -v -t -d ./...

.PHONY: vet
vet:
	go vet ./...
