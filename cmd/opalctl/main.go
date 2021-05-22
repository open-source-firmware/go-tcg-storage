// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"

	opal "github.com/bluecmd/go-opal"
	"github.com/bluecmd/go-opal/drive"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	d, err := drive.Open(os.Args[1])
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer d.Close()

	d0, err := opal.Discovery0(d)
	if err != nil {
		log.Fatalf("opal.Discovery0: %v", err)
	}
	spew.Dump(d0)
}
