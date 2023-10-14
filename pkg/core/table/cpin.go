// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations on Locking SP tables

package table

import (
	"github.com/matfax/go-tcg-storage/pkg/core"
	"github.com/matfax/go-tcg-storage/pkg/core/method"
	"github.com/matfax/go-tcg-storage/pkg/core/uid"
)

// ref: 5.3.2.12 Credential Table Group - C_PIN (Object Table)
// https://trustedcomputinggroup.org/wp-content/uploads/TCG_Storage_Architecture_Core_Spec_v2.01_r1.00.pdf
type CPINInfoRow struct {
	UID         uid.RowUID
	Name        *string
	CommonName  *string
	PIN         []byte
	CharSet     []byte
	TryLimit    *uint32
	Tries       *uint32
	Persistence *bool
}

func CPINInfo(s *core.Session) (*CPINInfoRow, error) {
	rowUID := uid.RowUID{}
	copy(rowUID[:], uid.Admin_C_PIN_SIDRow[:])

	val, err := GetFullRow(s, rowUID)
	if err != nil {
		return nil, err
	}
	row := CPINInfoRow{}
	for col, val := range val {
		switch col {
		case "0", "UID":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			copy(row.UID[:], v[:8])
		case "1", "Name":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := string(v)
			row.Name = &vv
		case "2", "CommonName":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := string(v)
			row.CommonName = &vv
		case "3", "PIN":
			v, ok := val.([]byte)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := v
			row.PIN = vv
		case "4", "CharSet":
			v, ok := val.([]uint8)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := v
			row.CharSet = vv
		case "5", "TryLimit":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.TryLimit = &vv
		case "6", "Tries":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			vv := uint32(v)
			row.Tries = &vv
		case "7", "Persistence":
			v, ok := val.(uint)
			if !ok {
				return nil, method.ErrMalformedMethodResponse
			}
			var vv bool
			if v > 0 {
				vv = true
			}
			row.Persistence = &vv
		}
	}
	return &row, nil
}
