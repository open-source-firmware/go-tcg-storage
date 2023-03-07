# go-tcg-storage

[![Workflow](https://github.com/open-source-firmware/go-tcg-storage/workflows/Release/badge.svg)](https://github.com/open-source-firmware/go-tcg-storage/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/open-source-firmware/go-tcg-storage)](https://goreportcard.com/report/github.com/open-source-firmware/go-tcg-storage)
[![GoDoc](https://godoc.org/github.com/open-source-firmware/go-tcg-storage?status.svg)](https://pkg.go.dev/github.com/open-source-firmware/go-tcg-storage@main)
[![Slack](https://slack.osfw.dev/badge.svg)](https://slack.osfw.dev)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://github.com/open-source-firmware/go-tcg-storage/blob/master/LICENSE)

Go library for interfacing TCG Storage and Security Subsystem Class (SSC) functions on storage devices.

Supported (or planned) standards:

 * [Core](https://trustedcomputinggroup.org/resource/tcg-storage-architecture-core-specification/)
 * [Opal 2.0](https://trustedcomputinggroup.org/resource/storage-work-group-storage-security-subsystem-class-opal/)
 * [Enterprise](https://trustedcomputinggroup.org/resource/storage-work-group-storage-security-subsystem-class-enterprise-specification/)
 * [Ruby](https://trustedcomputinggroup.org/resource/tcg-storage-security-subsystem-class-ruby-specification/)

Need support for another standard? Let us know by filing a feature request!

## Tools

 * [sedlockctl](cmd/sedlockctl/README.md) is a tool that helps you manage SED/TCG drives.<br>
   Install it: `go install github.com/open-source-firmware/go-tcg-storage/cmd/sedlockctl@main`

 * [tcgsdiag](cmd/tcgsdiag/README.md) lets you list a whole lot of diagnostic information about TCG drives.<br>
   Install it: `go install github.com/open-source-firmware/go-tcg-storage/cmd/tcgsdiag@main`

 * [tcgdiskstat](cmd/tcgdiskstat/README.md) is like `blkid` or `lsscsi` but for TCG drives.<br>
   Install it: `go install github.com/open-source-firmware/go-tcg-storage/cmd/tcgdiskstat@main`


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
The library does not rely on the in-kernel implementation of
TCG Opal[[1](https://github.com/torvalds/linux/commit/455a7b238cd6bc68c4a550cbbd37c1e22b64f71c)].

The most low-level interface is the `drive` interface that implements
the `IF-SEND` and `IF-RECV` functions that the TCG Storage standards
rely on. Likely nobody outside this library will find that library useful.
User of the `core` library usually dont neet to care about `drive` for its functionality
is just to abstract device types from the core library.

One abstraction up is the `core` library that implements the
TCG Storage specifications in a quite verbose manner. The guiding
principle with the `core` library is that you should be able to do
anything with it, but it might require you to know what functions
can be called under what circumstances.
The `core` supplies the user with the `NewCore` function, which opens a
given device and obtains disk information and Level0Discovery from the device.

Finally you have the `locking` library that implements the most
likely reason you are reading this. It allows you to get access
to, and modify, the locking ranges of a TCG Storage compliant
drive without caring much what version of the standards the drive
is implementing.

### Core Library

```go
import (
	"log"

	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/table"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/uid"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

func main() {
	core, err := tcg.NewCore("/dev/sda")
	if err != nil {
		log.Fatalf("tcg.NewCore(/dev/sda) failed: %v",err)
	}
	defer core.Close()

	// This will work if your drive implements GET_COMID,
	// otherwise you will need to figure out the ComID and
	// pass it in with WithComID(x)
	cs, err := tcg.NewControlSession(d, core.)
	if err != nil {
		log.Fatalf("tcg.NewControlSession(d,d0) failed: %v", err)
	}
	defer cs.Close()
	s, err := cs.NewSession(tcg.AdminSP)
	if err != nil {
		log.Fatalf("cs.NewSession(uid.AdminSP) failed: %v", err)
	}
	defer s.Close()

	// This is how you call a method on your SP:
	rand, err := table.ThisSP_Random(s, 8 /* bytes to generate */)

	// You can authenticate using the MSID like this:
	msidPin, err := table.Admin_C_PIN_MSID_GetPIN(s)
	if err := table.ThisSP_Authenticate(s, uid.AuthoritySID, msidPin); err != nil {
	 	log.Fatalf("Authentication as SID failed!")
	}
	// Session is now elevated
}
```

### Locking Library

The most minimal example looks something like this:

```go

import (
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
	"github.com/open-source-firmware/go-tcg-storage/pkg/locking"
)

func main() {
	d, err := drive.Open("/dev/sda")
	defer d.Close()

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
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
	"github.com/open-source-firmware/go-tcg-storage/pkg/locking"
)

func main() {
	d, err := drive.Open("/dev/sda")
	defer d.Close()

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

## Tested drives

These drives have been found to work without issues

| Manufacturer | Model | Transport | Features | Notes |
|--------------|-------|-----------|----------|-------|
| Corsair | Force MP510 | NVMe | Pyrite v1 | |
| Intel | P4510 (SSDPE2KX020T8O) | NVMe | Opal v2 | |
| Intel | P4610 (SSDPE2KE032T8O) | NVMe | Opal v2 | |
| Sabrent | Rocket 4.0 2TB | NVMe | Pyrite v2 | |
| Samsung | PM1735 (MZPLJ12THALA-00007) | NVMe | Opal v2 | Shadow MBR missing |
| Samsung | PM961 (MZVLW512HMJP-000L7) | NVMe | Opal v2 | |
| Samsung | PM981 (MZVLB512HAJQ-000L7) | NVMe | Opal v2 | |
| Samsung | PM983 (MZ1LB1T9HALS-00007) | NVMe | Opal v2 | |
| Samsung | PM9A1 (MZVL2256HCHQ-00B00) | NVMe | Pyrite v2 | |
| Samsung | PM9A3 (MZQL23T8HCLS-00A07) | NVMe | Opal v2 | |
| Samsung | SSD 860 | SATA | Opal v2 | |
| Samsung | SSD 970 EVO Plus | NVMe | Opal v2 | |
| Samsung | SSD 980 Pro (MZVL2250HCHQ) | NVMe | Opal v2 | |
| Seagate | 7E2000 (ST2000NX0343) | SAS3 | Enterprise | |
| Seagate | Exos X14 (ST10000NM0608) | SAS3 | Enterprise | |
| Seagate | Momentus Thin (ST500LT015) | SATA | Opal v2 | |
| SK hynix | PC611 (HFS001TD9TNI-L2B0B) | NVMe | Opal v2 | |
| Toshiba | MG08SCP16TE | SAS3 | Enterprise | |

*Samsung PNs ending in "7" seems to indicate Opal v2 features*
