// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bluecmd/go-tcg-storage/pkg/drive"
	"github.com/bluecmd/go-tcg-storage/pkg/locking"
)

var (
	sidPIN      = flag.String("sid", "", "PIN to authenticate to the AdminSP as SID")
	sidPINMSID  = flag.Bool("try-sid-msid", false, "Try to use the MSID as PIN to authenticate to the AdminSP in addition to other methods")
	sidPINHash  = flag.String("sid-hash", "sedutil-dta", "If set, transform the SID PIN using the specified hash function")
	user        = flag.String("user", "", "Username to authenticate to the LockingSP (admin1 or bandmaster0 is the default)")
	userPIN     = flag.String("password", "", "PIN used to authenticate ot the LockingSP (MSID is the default)")
	userPINHash = flag.String("hash", "sedutil-dta", "If set, transform the PIN using the specified hash function")
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Printf("Usage: %s [-flags ..] device\n", os.Args[0])
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
		}
		if r.WriteLocked {
			strr += "[write locked]"
		}
		if r.ReadLocked {
			strr += "[read locked]"
		}
		fmt.Printf("Range %3d: %s\n", i, strr)
	}
}
