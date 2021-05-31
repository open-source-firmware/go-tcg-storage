// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"

	"github.com/bluecmd/go-tcg-storage/pkg/drive"
	"github.com/bluecmd/go-tcg-storage/pkg/locking"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	spew.Config.Indent = "  "

	d, err := drive.Open(os.Args[1])
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer d.Close()

	cs, spid, err := locking.Initialize(d)
	if err != nil {
		log.Fatalf("locking.Initalize: %v", err)
	}
	defer cs.Close()

	l, err := locking.NewSession(cs, spid, locking.DefaultAuthority)
	if err != nil {
		log.Fatalf("locking.NewSession: %v", err)
	}
	defer l.Close()
	spew.Dump(l)
}
