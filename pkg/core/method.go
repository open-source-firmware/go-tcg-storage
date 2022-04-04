// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Method calling

package core

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core/stream"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

type InvokingID [8]byte
type MethodID [8]byte

type MethodFlag int

const (
	MethodFlagOptionalAsName MethodFlag = 1
)

var (
	InvokeIDNull   InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	InvokeIDThisSP InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

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

type MethodCall struct {
	buf bytes.Buffer
	// Used to verify detect programming errors
	depth int
	flags MethodFlag
}

// Prepare a new method call
func NewMethodCall(iid InvokingID, mid MethodID, flags MethodFlag) *MethodCall {
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

	// Synchronous mode specific: Ensure that there is no pending message
	// before we start.
	resp, err := c.Receive(proto, ses)
	if err != nil {
		return nil, err
	}
	if len(resp) > 0 {
		return nil, ErrReceivedUnexpectedResponse
	}

	if err = c.Send(proto, ses, b); err != nil {
		return nil, err
	}

	// There are a couple of reasons why we might receive empty data from c.Receive.
	//
	// Most relevant is this one:
	// "3.3.10.2.1 Restrictions (3.b)"
	// > If the TPer has not sufficiently processed the command payload and prepared a
	// > response, any IF-RECV command for that ComID SHALL receive a ComPacket with a
	// > Length field value of zero (no payload), an OutstandingData field value of 0x01, and a
	// > MinTransfer field value of zero.
	for i := 100; i >= 0; i-- {
		resp, err = c.Receive(proto, ses)
		if err != nil {
			return nil, err
		}
		if len(resp) > 0 {
			break
		}
		if i == 0 {
			return nil, ErrMethodTimeout
		}
		time.Sleep(10 * time.Millisecond)
	}

	reply, err := stream.Decode(resp)
	if err != nil {
		return nil, err
	}

	if len(reply) < 2 {
		return nil, ErrEmptyMethodResponse
	}

	// Check for special CloseSession response
	if len(reply) >= 4 {
		tok, ok1 := reply[0].(stream.TokenType)
		iid, ok2 := reply[1].([]byte)
		mid, ok3 := reply[2].([]byte)
		params, ok4 := reply[3].(stream.List)
		if ok1 && ok2 && ok3 && ok4 &&
			tok == stream.Call &&
			bytes.Equal(iid, InvokeIDSMU[:]) &&
			bytes.Equal(mid, MethodIDSMCloseSession[:]) {
			hsn, ok1 := params[0].(uint)
			tsn, ok2 := params[1].(uint)
			if ok1 && ok2 && int(hsn) == ses.HSN && int(tsn) == ses.TSN {
				return nil, ErrTPerClosedSession
			} else {
				return nil, ErrReceivedUnexpectedResponse
			}
		}
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
		err, ok := MethodStatusCodeMap[sc]
		if !ok {
			return nil, fmt.Errorf("method returned unknown status code 0x%02x", sc)
		}
		return nil, err
	}

	return reply[:len(reply)-2], nil
}
