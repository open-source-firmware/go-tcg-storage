# Copyright (c) 2021 by library authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY: build
build:
	go build ${LDFLAGS} -v -o target/tcgsdiag $(CURDIR)/cmd/tcgsdiag
	go build ${LDFLAGS} -v -o target/tcgdiskstat $(CURDIR)/cmd/tcgdiskstat
	go build ${LDFLAGS} -v -o target/sedlockctl $(CURDIR)/cmd/sedlockctl
	go build ${LDFLAGS} -v -o target/gosedctl $(CURDIR)/cmd/gosedctl

.PHONY: build-release
build-release: build-release-amd64 build-release-arm64

.PHONY: build-release-amd64
build-release-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o=tcgsdiag.linux.amd64 $(CURDIR)/cmd/tcgsdiag
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o=tcgdiskstat.linux.amd64 $(CURDIR)/cmd/tcgdiskstat
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o=sedlockctl.linux.amd64 $(CURDIR)/cmd/sedlockctl
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o=gosedctl.linux.amd64 $(CURDIR)/cmd/gosedctl

.PHONY: build-release-arm64
build-release-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o=tcgsdiag.linux.arm64 $(CURDIR)/cmd/tcgsdiag
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o=tcgdiskstat.linux.arm64 $(CURDIR)/cmd/tcgdiskstat
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o=sedlockctl.linux.arm64 $(CURDIR)/cmd/sedlockctl
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o=gosedctl.linux.arm64 $(CURDIR)/cmd/gosedctl

.PHONY: test
test:
	go test -v ./...

.PHONY: get-dependencies
get-dependencies:
	go get -v -t -d ./...

.PHONY: vet
vet:
	go vet ./...
