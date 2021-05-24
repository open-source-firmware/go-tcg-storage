// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Data Stream

package tcgstorage

type StreamToken []byte

var (
	StreamStartList        StreamToken = []byte{0xF0}
	StreamEndList          StreamToken = []byte{0xF1}
	StreamStartName        StreamToken = []byte{0xF2}
	StreamEndName          StreamToken = []byte{0xF3}
	StreamCall             StreamToken = []byte{0xF8}
	StreamEndOfData        StreamToken = []byte{0xF9}
	StreamEndOfSession     StreamToken = []byte{0xFA}
	StreamStartTransaction StreamToken = []byte{0xFB}
	StreamEndTransaction   StreamToken = []byte{0xFC}
	StreamEmptyAtom        StreamToken = []byte{0xFF}
)
