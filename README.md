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

## Supported Transports

The following transports are supported by the library:

 * NVMe
 * SATA
 * SAS

Need another transport? You can do one of two things:

 1. You can implement the `drive` interface yourself to talk to your device.
 2. You can file a feature request describing your setup and we can discuss implementing it

## Usage

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

