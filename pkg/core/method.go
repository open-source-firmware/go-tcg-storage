// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package tcgstorage

import (
	"bytes"
	"fmt"

	"github.com/bluecmd/go-tcg-storage/pkg/drive"
	"github.com/bluecmd/go-tcg-storage/pkg/core/stream"
)

type InvokingID [8]byte
type MethodID [8]byte

var (
	InvokeIDNull   InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	InvokeIDThisSP InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
)

type MethodCall struct {
	buf bytes.Buffer
	// Used to verify detect programming errors
	depth int
}

// Prepare a new method call
func NewMethodCall(iid InvokingID, mid MethodID) *MethodCall {
	m := &MethodCall{bytes.Buffer{}, 0}
	m.buf.Write(stream.Token(stream.Call))
	m.PushBytes(iid[:])
	m.PushBytes(mid[:])
	// Start argument list
	m.StartList()
	return m
}

func (m *MethodCall) StartList() {
	m.depth++
	m.buf.Write(stream.Token(stream.StartList))
}

func (m *MethodCall) EndList() {
	m.depth--
	m.buf.Write(stream.Token(stream.EndList))
}

// From "3.2.1.2 Method Signature Pseudo-code"
// Optional parameters are submitted to the method invocation as Named value pairs.
// The Name portion of the Named value pair SHALL be a uinteger. Starting at zero,
// these uinteger values are assigned based on the ordering of the optional parameters
// as defined in this document.
func (m *MethodCall) StartOptionalParameter(id uint) {
	m.depth++
	m.buf.Write(stream.Token(stream.StartName))
	m.buf.Write(stream.UInt(id))
}

func (m *MethodCall) NamedUInt(name string, val uint) {
	m.buf.Write(stream.Token(stream.StartName))
	m.buf.Write(stream.Bytes([]byte(name)))
	m.buf.Write(stream.UInt(val))
	m.buf.Write(stream.Token(stream.EndName))
}

func (m *MethodCall) NamedBool(name string, val bool) {
	if val {
		m.NamedUInt(name, 1)
	} else {
		m.NamedUInt(name, 0)
	}
}

func (m *MethodCall) EndOptionalParameter() {
	m.depth--
	m.buf.Write(stream.Token(stream.EndName))
}

// Add bytes to the method call
func (m *MethodCall) PushBytes(b []byte) {
	m.buf.Write(stream.Bytes(b))
}

// Marshal the complete method call to the data stream representation
func (mo *MethodCall) MarshalBinary() ([]byte, error) {
	m := *mo
	m.EndList() // End argument list
	// Finish method call
	m.buf.Write(stream.Token(stream.EndOfData))
	m.StartList()          // Status code list
	m.buf.Write([]byte{0}) // Expected status code
	m.buf.Write([]byte{0}) // Reserved
	m.buf.Write([]byte{0}) // Reserved
	m.EndList()
	if m.depth != 0 {
		return nil, fmt.Errorf("method argument list is unbalanced")
	}
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
