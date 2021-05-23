// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"

	tcg "github.com/bluecmd/go-tcg-storage"
	"github.com/bluecmd/go-tcg-storage/drive"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	d, err := drive.Open(os.Args[1])
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer d.Close()

	spl, err := drive.SecurityProtocols(d)
	if err != nil {
		log.Fatalf("drive.SecurityProtocols: %v", err)
	}
	spew.Dump(spl)
	crt, err := drive.Certificate(d)
	if err != nil {
		log.Fatalf("drive.Certificate: %v", err)
	}
	spew.Dump(crt)

	d0, err := tcg.Discovery0(d)
	if err != nil {
		log.Fatalf("tcg.Discovery0: %v", err)
	}
	spew.Dump(d0)
}
