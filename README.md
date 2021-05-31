# go-tcg-storage

[![Workflow](https://github.com/bluecmd/go-tcg-storage/workflows/Release/badge.svg)](https://github.com/bluecmd/go-tcg-storage/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bluecmd/go-tcg-storage)](https://goreportcard.com/report/github.com/bluecmd/go-tcg-storage)
[![GoDoc](https://godoc.org/github.com/bluecmd/go-tcg-storage?status.svg)](https://pkg.go.dev/github.com/bluecmd/go-tcg-storage@main)
[![Slack](https://slack.osfw.dev/badge.svg)](https://slack.osfw.dev)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://github.com/bluecmd/go-tcg-storage/blob/master/LICENSE)

Go library for interfacing TCG Storage and Security Subsystem Class (SSC) functions on storage devices.

Supported (or planned) standards:

 * [Core](https://trustedcomputinggroup.org/resource/tcg-storage-architecture-core-specification/)
 * [Opal 2.0](https://trustedcomputinggroup.org/resource/storage-work-group-storage-security-subsystem-class-opal/)
 * [Enterprise](https://trustedcomputinggroup.org/resource/storage-work-group-storage-security-subsystem-class-enterprise-specification/)
 * [Ruby](https://trustedcomputinggroup.org/resource/tcg-storage-security-subsystem-class-ruby-specification/)

Need support for another standard? Let us know by filing a feature request!

## Tools

 * [sedlockctl](cmd/sedlockctl/README.md) is a tool that helps you manage SED/TCG drives.<br>
   Install it: `go install github.com/bluecmd/go-tcg-storage/cmd/sedlockctl@main`

 * [tcgsdiag](cmd/tcgsdiag/README.md) lets you list a whole lot of diagnostic information about TCG drives.<br>
   Install it: `go install github.com/bluecmd/go-tcg-storage/cmd/tcgsdiag@main`

## Supported Transports

The following transports are supported by the library:

 * NVMe
 * SATA
 * SAS

Need another transport? You can do one of two things:

 1. You can implement the `drive` interface yourself to talk to your device.
 2. You can file a feature request describing your setup and we can discuss implementing it

## Usage

The library consists of multiple libraries in order to abstract
away the functionallity the library user does not need to care about.

The most low-level interface is the `drive` interface that implements
the `IF-SEND` and `IF-RECV` functions that the TCG Storage standards
rely on. Likely nobody outside this library will find that library useful.

One abstraction up is the `core` library that implements the 
TCG Storage specifications in a quite verbose manner. The guiding
principle with the `core` library is that you should be able to do
anything with it, but it might require you to know what functions
can be called under what circumstances.

Finally you have the `locking` library that implements the most
likely reason you are reading this. It allows you to get access
to, and modify, the locking ranges of a TCG Storage compliant
drive without caring much what version of the standards the drive
is implementing.

### Core Library

```go
import (
	tcg "github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/core/table"
	"github.com/bluecmd/go-tcg-storage/pkg/drive"
)

func main() {
	d, err := drive.Open("/dev/sda")
	defer d.Close()

	d0, err := tcg.Discovery0(d)
        // This will work if your drive implements GET_COMID,
        // otherwise you will need to figure out the ComID and
        // pass it in with WithComID(x)
        cs, err := tcg.NewControlSession(d, d0)
        defer cs.Close()
        s, err := cs.NewSession(tcg.AdminSP)
        defer s.Close()

        // This is how you call a method on your SP:
        rand, err := table.ThisSP_Random(s, 8 /* bytes to generate */)
        
        // You can authenticate using the MSID like this:
        msidPin, err := table.Admin_C_PIN_MSID_GetPIN(s)
        if err := table.ThisSP_Authenticate(s, tcg.AuthoritySID, msidPin); err != nil {
        	log.Fatalf("Authentication as SID failed!")
        }
        // Session is now elevated
}
```

### Locking Library

The most minimal example looks something like this:

```go

import (
	"github.com/bluecmd/go-tcg-storage/pkg/drive"
	"github.com/bluecmd/go-tcg-storage/pkg/locking"
)

func main() {
	cs, lmeta, err := locking.Initialize(d)
	defer cs.Close()
	l, err := locking.NewSession(cs, lmeta, locking.DefaultAuthorityWithMSID)
	defer l.Close()
	fmt.Printf("Authenticated user has %d locking ranges", len(l.Ranges))
}
```

A slightly more realistic example looks like this:
```go

import (
	"github.com/bluecmd/go-tcg-storage/pkg/drive"
	"github.com/bluecmd/go-tcg-storage/pkg/locking"
)

func main() {
        password := []byte{} /* Password for Admin1 or BandMaster0 */
	cs, lmeta, err := locking.Initialize(d,
		locking.WithAuth(locking.DefaultAuthorityWithMSID)
		locking.WithTakeOwnership(password),
		locking.WithHardening())
	defer cs.Close()
	l, err := locking.NewSession(cs, lmeta, locking.DefaultAuthority(password))
	defer l.Close()
	fmt.Printf("Authenticated user has %d locking ranges", len(l.Ranges))
}
```
