// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package tcgstorage

import (
	"bytes"

	"github.com/bluecmd/go-tcg-storage/drive"
	"github.com/bluecmd/go-tcg-storage/stream"
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

// Prepare a new method call
func NewMethodCall(iid InvokingID, mid MethodID) *MethodCall {
	m := &MethodCall{bytes.Buffer{}}
	m.PushToken(stream.Call)
	m.PushBytes(iid[:])
	m.PushBytes(mid[:])
	return m
}

// Add a stream-encoded token to the method call
func (m *MethodCall) PushToken(tok stream.TokenType) {
	m.buf.Write(stream.Token(tok))
}

// Add stream-encoded bytes to the method call
func (m *MethodCall) PushBytes(b []byte) {
	m.buf.Write(stream.Bytes(b))
}

// Marshal the complete method call to the data stream representation
func (m *MethodCall) MarshalBinary() ([]byte, error) {
	m.PushToken(stream.EndOfData) // Finish method call
	m.PushToken(stream.StartList) // Status code list
	m.PushBytes([]byte{0})        // Expected status code
	m.PushBytes([]byte{0})        // Reserved
	m.PushBytes([]byte{0})        // Reserved
	m.PushToken(stream.EndList)   // Status code list
	return m.buf.Bytes(), nil
}

// Execute a prepared Method call, returns a list of tokens returned from call.
func (m *MethodCall) Execute(c CommunicationIntf, proto drive.SecurityProtocol, ses *Session) ([]interface{}, error) {
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

	return stream.Decode(resp)
}
