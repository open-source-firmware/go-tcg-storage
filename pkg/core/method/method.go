// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package method

import (
	"bytes"
	"errors"

	"github.com/matfax/go-tcg-storage/pkg/core/stream"
	"github.com/matfax/go-tcg-storage/pkg/core/uid"
)

type MethodFlag int

const (
	MethodFlagOptionalAsName MethodFlag = 1
)

var (
	ErrMalformedMethodResponse    = errors.New("method response was malformed")
	ErrEmptyMethodResponse        = errors.New("method response was empty")
	ErrMethodListUnbalanced       = errors.New("method argument list is unbalanced")
	ErrTPerClosedSession          = errors.New("TPer forcefully closed our session")
	ErrReceivedUnexpectedResponse = errors.New("method response was unexpected")
	ErrMethodTimeout              = errors.New("method call timed out waiting for a response")

	MethodStatusSuccess uint = 0x00
	MethodStatusCodeMap      = map[uint]error{
		0x00: errors.New("method returned status SUCCESS"),
		0x01: errors.New("method returned status NOT_AUTHORIZED"),
		0x02: errors.New("method returned status OBSOLETE"),
		0x03: errors.New("method returned status SP_BUSY"),
		0x04: errors.New("method returned status SP_FAILED"),
		0x05: errors.New("method returned status SP_DISABLED"),
		0x06: errors.New("method returned status SP_FROZEN"),
		0x07: errors.New("method returned status NO_SESSIONS_AVAILABLE"),
		0x08: errors.New("method returned status UNIQUENESS_CONFLICT"),
		0x09: errors.New("method returned status INSUFFICIENT_SPACE"),
		0x0A: errors.New("method returned status INSUFFICIENT_ROWS"),
		0x0B: errors.New("method returned status INVALID_COMMAND"), /* from Core Revision 0.9 Draft */
		0x0C: errors.New("method returned status INVALID_PARAMETER"),
		0x0D: errors.New("method returned status INVALID_REFERENCE"),         /* from Core Revision 0.9 Draft */
		0x0E: errors.New("method returned status INVALID_SECMSG_PROPERTIES"), /* from Core Revision 0.9 Draft */
		0x0F: errors.New("method returned status TPER_MALFUNCTION"),
		0x10: errors.New("method returned status TRANSACTION_FAILURE"),
		0x11: errors.New("method returned status RESPONSE_OVERFLOW"),
		0x12: errors.New("method returned status AUTHORITY_LOCKED_OUT"),
		0x3F: errors.New("method returned status FAIL"),
	}

	ErrMethodStatusNotAuthorized       = MethodStatusCodeMap[0x01]
	ErrMethodStatusSPBusy              = MethodStatusCodeMap[0x03]
	ErrMethodStatusNoSessionsAvailable = MethodStatusCodeMap[0x07]
	ErrMethodStatusInvalidParameter    = MethodStatusCodeMap[0x0C]
	ErrMethodStatusAuthorityLockedOut  = MethodStatusCodeMap[0x12]
)

type Call interface {
	MarshalBinary() ([]byte, error)
	IsEOS() bool
}

type MethodCall struct {
	buf bytes.Buffer
	// Used to verify detect programming errors
	depth int
	flags MethodFlag
}

// Prepare a new method call
func NewMethodCall(iid uid.InvokingID, mid uid.MethodID, flags MethodFlag) *MethodCall {
	m := &MethodCall{bytes.Buffer{}, 0, flags}
	m.buf.Write(stream.Token(stream.Call))
	m.Bytes(iid[:])
	m.Bytes(mid[:])
	// Start argument list
	m.StartList()
	return m
}

// Copy the current state of a method call into a new independent copy
func (m *MethodCall) Clone() *MethodCall {
	mn := &MethodCall{bytes.Buffer{}, m.depth, m.flags}
	mn.buf.Write(m.buf.Bytes())
	return mn
}

func (m *MethodCall) IsEOS() bool {
	return false
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
// > Optional parameters are submitted to the method invocation as Named value pairs.
// > The Name portion of the Named value pair SHALL be a uinteger. Starting at zero,
// > these uinteger values are assigned based on the ordering of the optional parameters
// > as defined in this document.
// The above is true for Core 2.0 things like OpalV2 but not for e.g. Enterprise.
// Thus, we provide a way for the code to switch between using uint or string.
func (m *MethodCall) StartOptionalParameter(id uint, name string) {
	m.depth++
	m.buf.Write(stream.Token(stream.StartName))
	if m.flags&MethodFlagOptionalAsName > 0 {
		m.buf.Write(stream.Bytes([]byte(name)))
	} else {
		m.buf.Write(stream.UInt(id))
	}
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

// Token adds a specific token to the MethodCall buffer.
func (m *MethodCall) Token(t stream.TokenType) {
	m.buf.Write(stream.Token(t))
}

// EndOptionalParameter ends the current optional parameter group
func (m *MethodCall) EndOptionalParameter() {
	m.depth--
	m.buf.Write(stream.Token(stream.EndName))
}

// Bytes adds a bytes atom
func (m *MethodCall) Bytes(b []byte) {
	m.buf.Write(stream.Bytes(b))
}

// UInt adds an uint atom
func (m *MethodCall) UInt(v uint) {
	m.buf.Write(stream.UInt(v))
}

// Bool adds a bool atom (as uint)
func (m *MethodCall) Bool(v bool) {
	if v {
		m.UInt(1)
	} else {
		m.UInt(0)
	}
}

func (m *MethodCall) RawByte(b []byte) {
	m.buf.Write(b)
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

type EOSMethodCall struct {
}

func (m *EOSMethodCall) MarshalBinary() ([]byte, error) {
	return stream.Token(stream.EndOfSession), nil
}

func (m *EOSMethodCall) IsEOS() bool {
	return true
}
