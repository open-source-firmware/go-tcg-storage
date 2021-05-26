// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package tcgstorage

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/bluecmd/go-tcg-storage/pkg/core/stream"
	"github.com/bluecmd/go-tcg-storage/pkg/drive"
)

type InvokingID [8]byte
type MethodID [8]byte

var (
	InvokeIDNull   InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	InvokeIDThisSP InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	ErrMalformedMethodResponse = errors.New("method response was malformed")
	ErrEmptyMethodResponse     = errors.New("method response was empty")
	ErrMethodListUnbalanced    = errors.New("method argument list is unbalanced")

	MethodStatusSuccess uint = 0x00
	MethodStatusCodeMap      = map[uint]string{
		0x00: "SUCCESS",
		0x01: "NOT_AUTHORIZED",
		0x02: "OBSOLETE",
		0x03: "SP_BUSY",
		0x04: "SP_FAILED",
		0x05: "SP_DISABLED",
		0x06: "SP_FROZEN",
		0x07: "NO_SESSIONS_AVAILABLE",
		0x08: "UNIQUENESS_CONFLICT",
		0x09: "INSUFFICIENT_SPACE",
		0x0A: "INSUFFICIENT_ROWS",
		0x0C: "INVALID_PARAMETER",
		0x0D: "OBSOLETE",
		0x0E: "OBSOLETE",
		0x0F: "TPER_MALFUNCTION",
		0x10: "TRANSACTION_FAILURE",
		0x11: "RESPONSE_OVERFLOW",
		0x12: "AUTHORITY_LOCKED_OUT",
		0x3F: "FAIL",
	}
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
	m.Bytes(iid[:])
	m.Bytes(mid[:])
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

// Start an optional parameters group
//
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

// Add a named value (uint) pair
func (m *MethodCall) NamedUInt(name string, val uint) {
	m.buf.Write(stream.Token(stream.StartName))
	m.buf.Write(stream.Bytes([]byte(name)))
	m.buf.Write(stream.UInt(val))
	m.buf.Write(stream.Token(stream.EndName))
}

// Add a named value (bool) pair
func (m *MethodCall) NamedBool(name string, val bool) {
	if val {
		m.NamedUInt(name, 1)
	} else {
		m.NamedUInt(name, 0)
	}
}

// End the current optional parameter group
func (m *MethodCall) EndOptionalParameter() {
	m.depth--
	m.buf.Write(stream.Token(stream.EndName))
}

// Add a bytes atom
func (m *MethodCall) Bytes(b []byte) {
	m.buf.Write(stream.Bytes(b))
}

// Add an uint atom
func (m *MethodCall) UInt(v uint) {
	m.buf.Write(stream.UInt(v))
}

// Add a bool atom (as uint)
func (m *MethodCall) Bool(v bool) {
	if v {
		m.UInt(1)
	} else {
		m.UInt(0)
	}
}

// Marshal the complete method call to the data stream representation
func (m *MethodCall) MarshalBinary() ([]byte, error) {
	mn := *m
	mn.EndList() // End argument list
	// Finish method call
	mn.buf.Write(stream.Token(stream.EndOfData))
	mn.StartList() // Status code list
	mn.buf.Write(stream.UInt(MethodStatusSuccess))
	mn.buf.Write(stream.UInt(0)) // Reserved
	mn.buf.Write(stream.UInt(0)) // Reserved
	mn.EndList()
	if mn.depth != 0 {
		return nil, ErrMethodListUnbalanced
	}
	return mn.buf.Bytes(), nil
}

// Execute a prepared Method call but do not expect anything in return.
func (m *MethodCall) Notify(c CommunicationIntf, proto drive.SecurityProtocol, ses *Session) error {
	b, err := m.MarshalBinary()
	if err != nil {
		return err
	}
	if err = c.Send(proto, ses, b); err != nil {
		return err
	}
	return nil
}

// Execute a prepared Method call, returns a list of tokens returned from call.
func (m *MethodCall) Execute(c CommunicationIntf, proto drive.SecurityProtocol, ses *Session) (stream.List, error) {
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

	if len(resp) < 2 {
		return nil, ErrEmptyMethodResponse
	}

	reply, err := stream.Decode(resp)
	if err != nil {
		return nil, err
	}
	// While the normal method result format is known, the Session Manager
	// methods use a different format. What is in common however is that
	// the last element should be the status code list.
	tok, ok1 := reply[len(reply)-2].(stream.TokenType)
	status, ok2 := reply[len(reply)-1].(stream.List)
	if !ok1 || !ok2 || tok != stream.EndOfData {
		return nil, ErrMalformedMethodResponse
	}

	sc, ok := status[0].(uint)
	if !ok {
		return nil, ErrMalformedMethodResponse
	}
	if sc != MethodStatusSuccess {
		str, ok := MethodStatusCodeMap[sc]
		if !ok {
			return nil, fmt.Errorf("method returned unknown status code 0x%02x", sc)
		}
		return nil, fmt.Errorf("method returned status 0x%02x (%s)", sc, str)
	}

	return reply[:len(reply)-1], nil
}
