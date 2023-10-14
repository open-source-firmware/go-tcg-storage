// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/sha1"
	"crypto/sha512"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

func HashSedutilDTA(password string, serial string) []byte {
	// This needs to match https://github.com/Drive-Trust-Alliance/sedutil/
	salt := fmt.Sprintf("%-20s", serial)
	return pbkdf2.Key([]byte(password), []byte(salt[:20]), 75000, 32, sha1.New)
}

func HashSedutil512(password string, serial string) []byte {
	// This needs to match https://github.com/ChubbyAnt/sedutil/
	salt := fmt.Sprintf("%-20s", serial)
	return pbkdf2.Key([]byte(password), []byte(salt[:20]), 500000, 32, sha512.New)
}
