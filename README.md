# go-tcg-storage

[![Go Report Card](https://goreportcard.com/badge/github.com/bluecmd/go-tcg-storage)](https://goreportcard.com/report/github.com/bluecmd/go-tcg-storage)
[![GoDoc](https://godoc.org/github.com/bluecmd/go-tcg-storage?status.svg)](https://godoc.org/github.com/bluecmd/go-tcg-storage)
[![Slack](https://slack.osfw.dev/badge.svg)](https://slack.osfw.dev)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://github.com/bluecmd/go-tcg-storage/blob/master/LICENSE)

Go library for interfacing TCG Storage Security Subsystem Class (SSC) functions on storage devices.

Supported (or planned) standards:

 * [Core](https://trustedcomputinggroup.org/resource/tcg-storage-architecture-core-specification/)
 * [Opal 2.0](https://trustedcomputinggroup.org/resource/storage-work-group-storage-security-subsystem-class-opal/)
 * [Enterprise](https://trustedcomputinggroup.org/resource/storage-work-group-storage-security-subsystem-class-enterprise-specification/)
 * [Ruby](https://trustedcomputinggroup.org/resource/tcg-storage-security-subsystem-class-ruby-specification/)

Need support for another standard? Let us know by filing a feature request!

## Supported Transports

The following transports are supported by the library:

 * NVMe
 * SATA
 * SAS

Need another transport? You can do one of two things:

 1. You can implement the `drive` interface yourself to talk to your device.
 2. You can file a feature request describing your setup and we can discuss implementing it

## Usage

TODO
