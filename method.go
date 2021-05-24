// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package tcgstorage

import (
	"bytes"
	"fmt"

	"github.com/bluecmd/go-tcg-storage/drive"
)

type InvokingID [8]byte
type MethodID [8]byte

var (
	InvokeIDNull   InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	InvokeIDThisSP InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
)

type MethodCall struct {
	buf bytes.Buffer
}

func NewMethodCall(iid InvokingID, mid MethodID) *MethodCall {
	m := &MethodCall{bytes.Buffer{}}
	m.PushToken(StreamCall)
	m.PushRaw([]byte{0xa8})
	m.PushRaw(iid[:])
	m.PushRaw([]byte{0xa8})
	m.PushRaw(mid[:])
	return m
}

func (m *MethodCall) PushToken(tok StreamToken) {
	m.buf.Write([]byte(tok))
}

func (m *MethodCall) PushRaw(b []byte) {
	m.buf.Write(b)
}

func (m *MethodCall) PushBytes(b []byte) {
	if len(b) == 0 {
		m.buf.Write([]byte{0xa0}) // Short atom with length of 0 ("3.2.2.3.1.2 Short atoms")
	} else if len(b) == 1 && b[0] < 64 {
		m.buf.Write(b) // Tiny atom
	} else {
		panic("atom not implemented")
		// Large atom
		// ...
	}
}

func (m *MethodCall) MarshalBinary() ([]byte, error) {
	m.PushToken(StreamEndOfData) // Finish method call
	m.PushToken(StreamStartList) // Status code list
	m.PushBytes([]byte{0})       // Expected status code
	m.PushBytes([]byte{0})       // Reserved
	m.PushBytes([]byte{0})       // Reserved
	m.PushToken(StreamEndList)   // Status code list
	return m.buf.Bytes(), nil
}

// Execute a prepared Method call, returns a list of tokens returned from call.
func (m *MethodCall) Execute(c CommunicationIntf, proto drive.SecurityProtocol, ses *Session) ([][]byte, error) {
	b, err := m.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if err = c.Send(proto, ses, b); err != nil {
		return nil, err
	}

	resp, err := c.Receive(proto, ses)
	if err != nil {
		return nil, err
	}

	fmt.Printf("method response: %+v\n", resp)
	// TODO: Decode into atom arrays
	return [][]byte{resp}, nil
}
