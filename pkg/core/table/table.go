// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/core/stream"
)

type RowUID [8]byte
type TableUID [8]byte

func (t *TableUID) Row(uid [4]byte) RowUID {
	return [8]byte{t[0], t[1], t[2], t[3], uid[0], uid[1], uid[2], uid[3]}
}

var (
	CellBlock_StartColumn uint = 3
	CellBlock_EndColumn   uint = 4

	Table_ColumnUID uint = 0

	MethodIDGet          core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x16}
	MethodIDNext         core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x08}
	MethodIDAuthenticate core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x1C}
	MethodIDRandom       core.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x06, 0x01}
)

func parseGetResult(res stream.List) (map[uint]interface{}, error) {
	methodResult, ok := res[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	inner, ok := methodResult[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	return parseRowValues(inner)
}

func parseRowValues(rv stream.List) (map[uint]interface{}, error) {
	res := map[uint]interface{}{}
	for i := range rv {
		if stream.EqualToken(rv[i], stream.StartName) {
			col, ok := rv[i+1].(uint)
			if !ok {
				return nil, core.ErrMalformedMethodResponse
			}
			if !stream.EqualToken(rv[i+2], stream.EndName) {
				res[col] = rv[i+2]
			}
		}
	}
	return res, nil
}
