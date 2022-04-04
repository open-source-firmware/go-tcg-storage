// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
	"github.com/open-source-firmware/go-tcg-storage/pkg/locking"

	// TODO: Move to locking API when it has MBR functions
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/table"
)

var (
	sidPIN      = flag.String("sid", "", "PIN to authenticate to the AdminSP as SID")
	sidPINMSID  = flag.Bool("try-sid-msid", false, "Try to use the MSID as PIN to authenticate to the AdminSP in addition to other methods")
	sidPINHash  = flag.String("sid-hash", "sedutil-dta", "If set, transform the SID PIN using the specified hash function")
	user        = flag.String("user", "", "Username to authenticate to the LockingSP (admin1 or bandmaster0 is the default)")
	userPIN     = flag.String("password", "", "PIN used to authenticate ot the LockingSP (MSID is the default)")
	userPINHash = flag.String("hash", "sedutil-dta", "If set, transform the PIN using the specified hash function")
	readMBRSize = flag.Int("read-mbr-size", 0, "If set to >0, specify how many bytes to read from the MBR table (otherwise read the whole table).")
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Printf("Usage: %s [-flags ..] device [verb...]\n", os.Args[0])
		fmt.Printf("\nVerbs:\n")
		fmt.Printf("  list               List all ranges (default)\n")
		fmt.Printf("  unlock-all         Unlocks all ranges completely\n")
		fmt.Printf("  lock-all           Lock all ranges completely\n")
		fmt.Printf("  mbr-done on|off    Sets the MBRDone property (hide/show Shadow MBR)\n")
		fmt.Printf("  read-mbr           Prints the binary data in the MBR area\n")
		return
	}
	d, err := drive.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer d.Close()
	snRaw, err := d.SerialNumber()
	if err != nil {
		log.Fatalf("drive.SerialNumber: %v", err)
	}
	sn := string(snRaw)

	spin := []byte{}
	if *sidPIN != "" {
		switch *sidPINHash {
		case "sedutil-dta":
			spin = HashSedutilDTA(*sidPIN, sn)
		default:
			log.Fatalf("Unknown hash method %q", *sidPINHash)
		}
	}

	initOps := []locking.InitializeOpt{}
	if len(spin) > 0 {
		initOps = append(initOps, locking.WithAuth(locking.DefaultAdminAuthority(spin)))
	}
	if *sidPINMSID {
		initOps = append(initOps, locking.WithAuth(locking.DefaultAuthorityWithMSID))
	}

	cs, lmeta, err := locking.Initialize(d, initOps...)
	if err != nil {
		log.Fatalf("locking.Initalize: %v", err)
	}
	defer cs.Close()

	var auth locking.LockingSPAuthenticator
	pin := []byte{}
	if *userPIN != "" {
		switch *userPINHash {
		case "sedutil-dta":
			pin = HashSedutilDTA(*userPIN, sn)
		default:
			log.Fatalf("Unknown hash method %q", *userPINHash)
		}
	}
	if *user != "" {
		var ok bool
		auth, ok = locking.AuthorityFromName(*user, pin)
		if !ok {
			log.Fatalf("Authority %q is not known for this device", *user)
		}
	} else {
		if len(pin) == 0 {
			auth = locking.DefaultAuthorityWithMSID
		} else {
			auth = locking.DefaultAuthority(pin)
		}
	}

	l, err := locking.NewSession(cs, lmeta, auth)
	if err != nil {
		log.Fatalf("locking.NewSession: %v", err)
	}
	defer l.Close()

	args := flag.Args()[1:]
	verb := "list"
	if len(args) > 0 {
		verb = args[0]
	}
	switch verb {
	case "list":
		list(l)
	case "unlock-all":
		unlockAll(l)
	case "lock-all":
		lockAll(l)
	case "mbr-done":
		if len(args) < 2 {
			log.Fatalf("Missing argument to mbr-done verb")
		}
		var v bool
		if args[1] == "on" {
			v = true
		} else if args[1] == "off" {
			v = false
		} else {
			log.Fatalf("Argument %q is not 'on' or 'off'", args[1])
		}
		setMBRDone(l, v)
	case "read-mbr":
		mbi, err := table.MBR_TableInfo(l.Session)
		if err != nil {
			log.Fatalf("table.MBR_TableInfo failed: %v", err)
		}
		mbuf := make([]byte, mbi.SuggestBufferSize(l.Session))
		sz := mbi.Size
		if *readMBRSize > 0 && uint32(*readMBRSize) < sz {
			sz = uint32(*readMBRSize)
		}
		pos := uint32(0)
		chk := uint32(len(mbuf))
		for i := sz; i != 0; i -= chk {
			if n, err := table.MBR_Read(l.Session, mbuf, pos); n != len(mbuf) || err != nil {
				log.Fatalf("table.MBR_Read failed: %v (read: %d)", err, n)
			}
			pos += chk
			if i < chk {
				os.Stdout.Write(mbuf[:i])
				break
			} else {
				os.Stdout.Write(mbuf)
			}
		}
	default:
		log.Fatalf("Unknown verb %q", verb)
	}
}

func list(l *locking.LockingSP) {
	if len(l.Ranges) == 0 {
		log.Fatalf("No available locking ranges as this user\n")
	}
	for i, r := range l.Ranges {
		strr := "whole disk"
		if r.End > 0 {
			strr = fmt.Sprintf("%d to %d", r.Start, r.End)
		}
		if !r.WriteLockEnabled && !r.ReadLockEnabled {
			strr = "disabled"
		} else {
			if r.WriteLocked {
				strr += " [write locked]"
			}
			if r.ReadLocked {
				strr += " [read locked]"
			}
		}
		if r == l.GlobalRange {
			strr += " [global]"
		}
		if r.Name != nil {
			strr += fmt.Sprintf(" [name=%q]", *r.Name)
		}
		fmt.Printf("Range %3d: %s\n", i, strr)
	}
}

func unlockAll(l *locking.LockingSP) {
	for i, r := range l.Ranges {
		if err := r.UnlockRead(); err != nil {
			log.Printf("Read unlock range %d failed: %v", i, err)
		}
		if err := r.UnlockWrite(); err != nil {
			log.Printf("Write unlock range %d failed: %v", i, err)
		}
	}
}

func lockAll(l *locking.LockingSP) {
	for i, r := range l.Ranges {
		if err := r.LockRead(); err != nil {
			log.Printf("Read lock range %d failed: %v", i, err)
		}
		if err := r.LockWrite(); err != nil {
			log.Printf("Write lock range %d failed: %v", i, err)
		}
	}
}

func setMBRDone(l *locking.LockingSP, v bool) {
	if err := l.SetMBRDone(v); err != nil {
		log.Fatalf("SetMBRDone failed: %v", err)
	}
}
