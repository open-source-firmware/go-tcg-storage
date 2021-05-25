// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Data Stream

package stream

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type TokenType uint8

var (
	StartList        TokenType = 0xF0
	EndList          TokenType = 0xF1
	StartName        TokenType = 0xF2
	EndName          TokenType = 0xF3
	Call             TokenType = 0xF8
	EndOfData        TokenType = 0xF9
	EndOfSession     TokenType = 0xFA
	StartTransaction TokenType = 0xFB
	EndTransaction   TokenType = 0xFC
	EmptyAtom        TokenType = 0xFF

	ErrUnbalancedList = errors.New("message contained unbalanced list structures")
)

type BytesData struct {
	Data []byte
}

type UIntData struct {
	Value uint
}

type TokenData struct {
	Token TokenType
}

func Token(tok TokenType) []byte {
	return []byte{byte(tok)}
}

func UInt(val uint) []byte {
	if val < 64 {
		return []byte{uint8(val)}
	}
	if val < 65536 {
		x := make([]byte, 3)
		x[0] = 0x82
		binary.BigEndian.PutUint16(x[1:], uint16(val))
		return x
	}
	x := make([]byte, 5)
	x[0] = 0x84
	binary.BigEndian.PutUint32(x[1:], uint32(val))
	return x
}

func Bytes(b []byte) []byte {
	// Tiny atom are not used for binary ("3.2.2.3.1 Simple Tokens – Atoms Overview")
	if len(b) < 16 {
		// Short Atom and 0-Length Atom
		return append([]byte{0xa0 | uint8(len(b))}, b...)
	} else if len(b) < 2048 {
		// Medium atom
		return append([]byte{0xd0 | uint8((len(b)>>8)&0x7), uint8(len(b) & 0xff)}, b...)
	} else {
		// TODO: Long atom
		// Really though, when would this be used?
		panic("long atom not implemented")
	}
}

func Decode(b []byte) ([]interface{}, error) {
	res, rest, err := internalDecode(b, 0)
	if len(rest) > 0 {
		return nil, ErrUnbalancedList
	}
	return res, err
}

func internalDecode(b []byte, depth int) ([]interface{}, []byte, error) {
	res := []interface{}{}
	for len(b) > 0 {
		s := 1
		var x interface{}
		if b[0]&0x80 == 0 {
			// Tiny atom
			x = UIntData{uint(b[0])}
		} else if b[0]&0xC0 == 0x80 {
			isbyte := b[0]&0x20 > 0
			// Short atom
			s = int(b[0] & 0xf)
			if isbyte {
				bc := make([]byte, s)
				copy(bc, b[1:1+s])
				x = BytesData{bc}
			} else {
				var v uint
				for _, i := range b[1 : 1+s] {
					v = v<<8 | uint(i)
				}
				x = UIntData{v}
			}
			s += 1
		} else if b[0]&0xE0 == 0xC0 { // Medium atom
			isbyte := b[0]&0x10 > 0
			s = int(b[0]&0x7)<<8 | int(b[1])
			if isbyte {
				bc := make([]byte, s)
				copy(bc, b[2:2+s])
				x = BytesData{bc}
				s += 2
			} else {
				return nil, nil, fmt.Errorf("medium integer not implemented")
			}
		} else if b[0] == byte(StartList) {
			list, rest, err := internalDecode(b[1:], depth+1)
			if err != nil {
				return nil, nil, err
			}
			s = (len(b) - len(rest))
			x = list
		} else if b[0] == byte(EndList) {
			if depth == 0 {
				return nil, nil, ErrUnbalancedList
			}
			b = b[1:]
			break
		} else if b[0]&0xF0 == 0xF0 {
			// Token
			x = TokenData{TokenType(uint8(b[0]))}
		} else {
			return nil, nil, fmt.Errorf("unknown atom 0x%02x", b[0])
		}
		res = append(res, x)
		b = b[s:]
	}
	return res, b, nil
}

func EqualBytes(obj interface{}, b []byte) bool {
	bd, ok := obj.(BytesData)
	if !ok {
		return false
	}
	// Special nil case
	if len(b) == 0 && len(bd.Data) == 0 {
		return true
	}
	return bytes.Equal(b, bd.Data)
}

func EqualToken(obj interface{}, b TokenType) bool {
	bd, ok := obj.(TokenData)
	if !ok {
		return false
	}
	return bd.Token == b
}

func EqualUInt(obj interface{}, b uint) bool {
	bd, ok := obj.(UIntData)
	if !ok {
		return false
	}
	return bd.Value == b
}
