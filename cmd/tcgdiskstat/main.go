package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

var (
	outputFmt = flag.String("output", "table", "Output format; one of [table, json, openmetrics]")
	noHeader  = flag.Bool("no-header", false, "Supress the header in table format output")
)

type DeviceState struct {
	Device   string
	Identity *drive.Identity
	Level0   *tcg.Level0Discovery
}

type Devices []DeviceState

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("The following state flags might be shown:")
		fmt.Println("  L/l - Locking is supported and is enabled (L) or disabled (l)")
		fmt.Println("  M/m - MBR is enabled and is active (M) or hidden (m)")
		fmt.Println("  E   - The device has media encryption")
		fmt.Println("  P   - The Admin SP SID PIN is set to MSID [Block SID feature specific]")
		fmt.Println("  !   - Authentication to Admin SP is blocked [Block SID feature specific]")
		fmt.Println()
	}
	flag.Parse()

	sysblk, err := ioutil.ReadDir("/sys/class/block/")
	if err != nil {
		log.Printf("Failed to enumerate block devices: %v", err)
		return
	}

	var state Devices

	for _, fi := range sysblk {
		devname := fi.Name()
		if _, err := os.Stat(filepath.Join("/sys/class/block", devname, "device")); os.IsNotExist(err) {
			continue
		}
		devpath := filepath.Join("/dev", devname)
		if _, err := os.Stat(devpath); os.IsNotExist(err) {
			log.Printf("Failed to find device node %s", devpath)
			continue
		}

		d, err := drive.Open(devpath)
		if err != nil {
			log.Printf("drive.Open(%s): %v", devpath, err)
			continue
		}
		defer d.Close()
		identity, err := d.Identify()
		if err != nil {
			log.Printf("drive.Identify(%s): %v", devpath, err)
		}
		d0, err := tcg.Discovery0(d)
		if err != nil {
			if err != tcg.ErrNotSupported {
				log.Printf("tcg.Discovery0(%s): %v", devpath, err)
				continue
			}
			d0 = nil
		}
		state = append(state, DeviceState{
			Device:   devpath,
			Identity: identity,
			Level0:   d0,
		})
	}

	if *outputFmt == "json" {
		outputJSON(state)
	} else if *outputFmt == "openmetrics" {
		outputMetrics(state)
	} else if *outputFmt == "table" {
		outputTable(state)
	} else {
		fmt.Printf("Unsupported output format %q\n", *outputFmt)
		flag.Usage()
		os.Exit(2)
	}
}

func outputJSON(state Devices) {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	os.Stdout.Write(b)
}

func sscFeatures(l0 *tcg.Level0Discovery) []string {
	feat := []string{}
	if l0.Enterprise != nil {
		feat = append(feat, "Enterprise")
	}
	if l0.OpalV1 != nil {
		feat = append(feat, "Opal 1")
	}
	if l0.OpalV2 != nil {
		feat = append(feat, "Opal 2")
	}
	if l0.Opalite != nil {
		feat = append(feat, "Opalite")
	}
	if l0.PyriteV1 != nil {
		feat = append(feat, "Pyrite 1")
	}
	if l0.PyriteV2 != nil {
		feat = append(feat, "Pyrite 2")
	}
	if l0.RubyV1 != nil {
		feat = append(feat, "Ruby 1")
	}
	return feat
}

func outputTable(state Devices) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if *noHeader == false {
		fmt.Fprintf(w, "DEVICE\tMODEL\tSERIAL\tFIRMWARE\tPROTOCOL\tSSC\tSTATE\n")
	}
	for _, s := range state {
		feat := []string{}
		state := ""
		if s.Level0 != nil {
			feat = sscFeatures(s.Level0)
			if l := s.Level0.Locking; l != nil {
				if l.LockingEnabled {
					state += "L"
				} else if l.LockingSupported {
					state += "l"
				}

				if l.MBREnabled {
					if l.MBRDone {
						state += "m"
					} else {
						state += "M"
					}
				}
				if l.MediaEncryption {
					state += "E"
				}
			}
			if b := s.Level0.BlockSID; b != nil {
				if !b.SIDValueState {
					state += "P"
				}
				if b.SIDAuthenticationBlockedState {
					state += "!"
				}
			}
		} else {
			state = "-"
			feat = []string{"-"}
		}

		fmt.Fprint(w,
			s.Device, "\t",
			s.Identity.Model, "\t",
			s.Identity.SerialNumber, "\t",
			s.Identity.Firmware, "\t",
			s.Identity.Protocol, "\t",
			strings.Join(feat, ","), "\t",
			state, "\t",
			"\n")
	}
	w.Flush()
}
