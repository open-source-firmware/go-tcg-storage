// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"

	tcg "github.com/bluecmd/go-tcg-storage"
	"github.com/bluecmd/go-tcg-storage/drive"
	"github.com/davecgh/go-spew/spew"
)

func TestComID(d tcg.DriveIntf) {
	comID, err := tcg.GetComID(d)
	if err != nil {
		log.Fatalf("Unable to allocate ComID: %v", err)
	}
	log.Printf("Allocated ComID 0x%08x", comID)
	valid, err := tcg.IsComIDValid(d, comID)
	if err != nil {
		log.Fatalf("Unable to validate allocated ComID: %v", err)
	}
	if !valid {
		log.Fatalf("Allocated ComID not valid")
	}
	log.Printf("ComID validated successfully")

	if err := tcg.StackReset(d, comID); err != nil {
		log.Fatalf("Unable to reset the synchronous protocol stack: %v", err)
	}
	log.Printf("Synchronous protocol stack reset successfully")
}

func main() {
	spew.Config.Indent = "  "

	d, err := drive.Open(os.Args[1])
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer d.Close()

	fmt.Printf("===> DRIVE SECURITY INFORMATION\n")
	spl, err := drive.SecurityProtocols(d)
	if err != nil {
		log.Fatalf("drive.SecurityProtocols: %v", err)
	}
	log.Printf("SecurityProtocols: %+v", spl)
	crt, err := drive.Certificate(d)
	if err != nil {
		log.Fatalf("drive.Certificate: %v", err)
	}
	log.Printf("Drive certificate:")
	spew.Dump(crt)
	fmt.Printf("\n")

	fmt.Printf("===> TCG ComID SELF-TEST\n")
	TestComID(d)
	fmt.Printf("\n")

	fmt.Printf("===> TCG FEATURE DISCOVERY\n")
	d0, err := tcg.Discovery0(d)
	if err != nil {
		log.Fatalf("tcg.Discovery0: %v", err)
	}
	spew.Dump(d0)
	fmt.Printf("\n")

	fmt.Printf("===> TCG SESSION\n")
	s, err := tcg.NewSession(d, d0.TPer)
	if err != nil {
		log.Fatalf("s.NewSession: %v", err)
	}
	spew.Dump(s)
}
