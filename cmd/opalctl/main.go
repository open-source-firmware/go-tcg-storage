// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"

	opal "github.com/bluecmd/go-opal"
	"github.com/bluecmd/go-opal/drive"
)

func main() {
	d, err := drive.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Unable to open drive: %v", err)
	}

	opal.Open(d)
}
