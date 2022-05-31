// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core - Session Manager and Session

package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core/method"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/stream"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/uid"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

var (
	ErrTPerSyncNotSupported        = errors.New("synchronous operation not supported by TPer")
	ErrTPerBufferMgmtNotSupported  = errors.New("TPer supports buffer management, but that is not implemented in this library")
	ErrInvalidPropertiesResponse   = errors.New("response was not the expected Properties call format")
	ErrInvalidStartSessionResponse = errors.New("response was not the expected SyncSession format")
	ErrPropertiesCallFailed        = errors.New("the properties call returned non-zero")
	ErrSessionAlreadyClosed        = errors.New("the session has been closed by us")

	sessionRand *rand.Rand
)

const (
	DefaultMaxComPacketSize uint = 1024 * 1024
	DefaultReceiveRetries        = 100
	DefaultReceiveInterval       = 10 * time.Millisecond
)

type ProtocolLevel uint

const (
	ProtocolLevelUnknown    ProtocolLevel = 0
	ProtocolLevelEnterprise ProtocolLevel = 1
	ProtocolLevelCore       ProtocolLevel = 2
)

func (p *ProtocolLevel) String() string {
	switch *p {
	case ProtocolLevelEnterprise:
		return "Enterprise"
	case ProtocolLevelCore:
		return "Core V2.0"
	default:
		return "<Unknown>"
	}
}

type Session struct {
	ControlSession *ControlSession
	MethodFlags    method.MethodFlag
	ProtocolLevel  ProtocolLevel
	d              drive.DriveIntf
	c              CommunicationIntf
	closed         bool
	ComID          ComID
	TSN, HSN       int
	// See "3.2.3.3.1.2 SeqNumber"
	SeqLastXmit     int
	SeqLastAcked    int
	SeqNextExpected int
	ReadOnly        bool // Ignored for Control Sessions
	ReceiveRetries  int
	ReceiveInterval time.Duration
}

type ControlSession struct {
	Session
	HostProperties           HostProperties
	TPerProperties           TPerProperties
	MaxComPacketSizeOverride uint
}

type HostProperties struct {
	MaxMethods               uint
	MaxSubpackets            uint
	MaxPacketSize            uint
	MaxPackets               uint
	MaxComPacketSize         uint
	MaxResponseComPacketSize *uint
	MaxIndTokenSize          uint
	MaxAggTokenSize          uint
	ContinuedTokens          bool
	SequenceNumbers          bool
	AckNak                   bool
	Asynchronous             bool
}
type TPerProperties struct {
	MaxMethods               uint
	MaxSubpackets            uint
	MaxPacketSize            uint
	MaxPackets               uint
	MaxComPacketSize         uint
	MaxResponseComPacketSize *uint
	MaxSessions              *uint
	MaxReadSessions          *uint
	MaxIndTokenSize          uint
	MaxAggTokenSize          uint
	MaxAuthentications       *uint
	MaxTransactionLimit      *uint
	DefSessionTimeout        *uint
	MaxSessionTimeout        *uint
	MinSessionTimeout        *uint
	DefTransTimeout          *uint
	MaxTransTimeout          *uint
	MinTransTimeout          *uint
	MaxComIDTime             *uint
	ContinuedTokens          bool
	SequenceNumbers          bool
	AckNak                   bool
	Asynchronous             bool
}

var (
	// Table 168: "Communications Initial Assumptions"
	InitialTPerProperties = TPerProperties{
		MaxSubpackets:    1,
		MaxPacketSize:    1004,
		MaxPackets:       1,
		MaxComPacketSize: 1024,
		MaxIndTokenSize:  968,
		MaxAggTokenSize:  968,
		MaxMethods:       1,
		ContinuedTokens:  false,
		SequenceNumbers:  false,
		AckNak:           false,
		Asynchronous:     false,
	}
	// Increased to match that one of the highest standard we support
	InitialHostProperties = HostProperties{
		MaxSubpackets:    1,
		MaxPacketSize:    2028,
		MaxPackets:       1,
		MaxComPacketSize: 2048,
		MaxIndTokenSize:  1992,
		MaxAggTokenSize:  1992,
		MaxMethods:       1,
		ContinuedTokens:  false,
		SequenceNumbers:  false,
		AckNak:           false,
		Asynchronous:     false,
	}
)

type SessionOpt func(s *Session)
type ControlSessionOpt func(s *ControlSession)

func WithComID(c ComID) ControlSessionOpt {
	return func(s *ControlSession) {
		s.ComID = c
	}
}

func WithMaxComPacketSize(size uint) ControlSessionOpt {
	return func(s *ControlSession) {
		s.MaxComPacketSizeOverride = size
	}
}

func WithReceiveTimeout(retries int, interval time.Duration) ControlSessionOpt {
	return func(s *ControlSession) {
		s.ReceiveRetries = retries
		s.ReceiveInterval = interval
	}
}

func WithHSN(hsn int) SessionOpt {
	return func(s *Session) {
		s.HSN = hsn
	}
}

func WithReadOnly() SessionOpt {
	return func(s *Session) {
		s.ReadOnly = true
	}
}

// Initiate a new control session with a ComID.
func NewControlSession(d drive.DriveIntf, d0 *Level0Discovery, opts ...ControlSessionOpt) (*ControlSession, error) {
	// --- Control Sessions
	//
	// Every ComID has exactly one control session. This is that session.
	//
	// --- Communication Properties
	//
	// Dyanmic ComIDs seem great from reading the spec, but sadly it seems it is not
	// commonly implemented, which means that we will fight over a single shared ComID.
	// I expect that this can cause issues where session ComPackets are routed to
	// another application on the same ComID - or that another application could
	// simply inject commands in an established session (unless the session has
	// transitioned into a secure session).
	//
	// > "When an IF-RECV is sent to the TPer using a particular ComID, the TPer SHALL respond by putting
	// > packets from the sessions associated with the ComID into the response"
	//
	// TODO: Investigate ComID crosstalk.

	if !d0.TPer.SyncSupported {
		return nil, ErrTPerSyncNotSupported
	}

	if d0.TPer.BufferMgmtSupported {
		return nil, ErrTPerBufferMgmtNotSupported
	}

	hp := InitialHostProperties
	tp := InitialTPerProperties
	c := NewPlainCommunication(d, hp, tp)
	s := &ControlSession{
		Session: Session{
			d:               d,
			c:               c,
			ComID:           ComIDInvalid,
			TSN:             0,
			HSN:             0,
			ReceiveRetries:  DefaultReceiveRetries,
			ReceiveInterval: DefaultReceiveInterval,
		},
		HostProperties:           hp,
		TPerProperties:           tp,
		MaxComPacketSizeOverride: DefaultMaxComPacketSize,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.ComID == ComIDInvalid {
		var err error
		s.ComID, err = GetComID(d)
		if err != nil {
			return nil, fmt.Errorf("unable to auto-allocate ComID: %v", err)
		}
	}

	if d0.Enterprise != nil {
		// The Enterprise SSC implements optional parameters with explicit variable
		// names, while the core spec says to use uintegers instead. This is likely
		// the fact that it is the oldest spec and based on the draft of TCG Core 0.9
		s.MethodFlags |= method.MethodFlagOptionalAsName
		s.ProtocolLevel = ProtocolLevelEnterprise
	} else {
		s.ProtocolLevel = ProtocolLevelCore
	}
	// Try to reset the synchronous protocol stack for the ComID to minimize
	// the dependencies on the implicit state. However, I suspect not all drives
	// implement it so we do it best-effort.
	StackReset(d, s.ComID)

	// Set preferred options
	rhp := InitialHostProperties
	// Technically we should be able to advertise 0 here and the disk should pick
	// for us, but that results in small values being picked in practice.
	rhp.MaxComPacketSize = s.MaxComPacketSizeOverride
	rhp.MaxPacketSize = rhp.MaxComPacketSize - 20
	rhp.MaxIndTokenSize = rhp.MaxComPacketSize - 20 - 24 - 12
	rhp.MaxAggTokenSize = rhp.MaxComPacketSize - 20 - 24 - 12
	rhp.MaxSubpackets = 1024
	rhp.MaxPackets = 1024

	// TODO: These are not fully implemented yet, so let's not advertise them
	//rhp.SequenceNumbers = true
	//rhp.AckNak = true

	var err error
	hp, tp, err = s.properties(&rhp)
	if err != nil {
		return nil, err
	}

	// Update the communication with the active properties
	s.c = NewPlainCommunication(d, hp, tp)
	s.HostProperties = hp
	s.TPerProperties = tp
	return s, nil
}

// Initiate a new session with a Security Provider
//
// The session will be a read-write by default, but can be changed by passing
// a SessionOpt from WithReadOnly() as argument. The session HSN will be random
// unless passed with WithHSN(x).
func (cs *ControlSession) NewSession(spid uid.SPID, opts ...SessionOpt) (*Session, error) {
	// --- What is a Session?
	//
	// Quoting "3.3.7.1 Sessions"
	// "All communications with an SP occurs within sessions. A session SHALL be started by a host and
	// successfully ended by a host."
	//
	// NOTE: This is *not* the same as a Control Session. These are "regular" Sessions.
	//
	// We will generate a Host Session Number (HSN), and we will be provided a TPer Session Number (TSN).
	// The TSN is guaranteed to be unique in the same ComID - thus the session is bound to a ComID it seems.
	//
	// --- Communication Properties
	//
	// Quoting "5.2.2.4.1 Communication Rules Based on TPer Properties and Host Properties"
	// > When communicating on statically allocated ComIDs, it is possible for the TPer’s knowledge of the
	// > HostProperties to be reset without the host’s knowledge (e.g. due to a TCG Hardware reset or a TCG
	// > Power Cycle reset). In this case, the TPer’s knowledge of the host’s communication properties will be
	// > reset to the initial assumed values shown in Table 168. This could adversely affect the performance of
	// > sessions that the host opens on the statically allocated ComID after the reset occurs. To prevent such
	// > performance degredation, it is the host's responsibility to invoke Properties with the HostProperties
	// > parameter prior to each invocation of StartSession on statically allocated ComIDs.
	// >
	// > This problem does not occur when using dynamically allocated ComIDs, because dynamically allocated
	// > ComIDs become inactive when the TPer is reset. The host receives an indication that the ComID is
	// > inactive if it attempts further communication on that ComID. Therefore, the host needs to invoke
	// > Properties with the HostProperties parameter only once per dynamically allocated ComID.

	// Quoting "5.2.2.3 Setting HostProperties"
	// > Subsequent submission of these values (in a subsequent invocation of the Properties method)
	// > SHALL supersede values submitted to previous invocations of the Properties method for that ComID.
	// > Submitted values, if applicable, SHALL only apply to sessions started after the submission of those
	// > values, and not to sessions that are already open on that ComID.
	// > [..]
	// > If the host specifies a value for a property that does not meet the minimum requirement as defined in Table
	// > 168, then the TPer SHALL use the minimum value defined in Table 168 in place of the value supplied
	// > by the host.

	// This is all pretty much impossible to get to work correctly when using
	// shared ComIDs, so let's not try too hard. We set the HostProperties when
	// the ControlSession is created, and if something else changes it between
	// then and the call to NewSession() we would be out of sync. Oh well...

	s := &Session{
		MethodFlags:     cs.MethodFlags,
		ProtocolLevel:   cs.ProtocolLevel,
		d:               cs.d,
		c:               cs.c,
		ControlSession:  cs,
		ComID:           cs.ComID,
		TSN:             0,
		HSN:             -1,
		ReceiveRetries:  cs.ReceiveRetries,
		ReceiveInterval: cs.ReceiveInterval,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.HSN > 0xffffffff {
		return nil, fmt.Errorf("too large HSN provided")
	}

	if s.HSN == -1 {
		s.HSN = int(sessionRand.Int31())
	}

	mc := method.NewMethodCall(uid.InvokeIDSMU, uid.MethodIDSMStartSession, s.MethodFlags)
	mc.UInt(uint(s.HSN))
	mc.Bytes(spid[:])
	mc.Bool(!s.ReadOnly)
	// "5.3.4.1.2.1 Anybody"
	// > The Anybody authority is always considered "authenticated" within a session, even if the Anybody
	// > authority was not specifically called out during session startup.
	// Thus, we do not specify any authority here and let the users call ThisSP_Authenticate
	// to elevate the session.

	basemc := mc.Clone()
	if s.ProtocolLevel == ProtocolLevelEnterprise {
		// sedutil recommends setting a timeout for session on Enterprise protocol
		// level. For normal Core devices I can't get it to work (INVALID_PARAMETER)
		// so only do it for Enterprise drives for now.
		mc.StartOptionalParameter(5, "SessionTimeout")
		mc.UInt(30000 /* 30 sec */)
		mc.EndOptionalParameter()
	}

	// Try with the method call with the optional parameters first,
	// and if that fails fall back to the basic method call (basemc).
	resp, err := cs.ExecuteMethod(mc)
	if err == method.ErrMethodStatusInvalidParameter {
		resp, err = cs.ExecuteMethod(basemc)
	}
	if err != nil {
		return nil, err
	}

	if len(resp) != 4 {
		return nil, ErrInvalidStartSessionResponse
	}
	params, ok := resp[3].(stream.List)

	// See "5.2.2.1.2 Properties Response".
	// The returned response is in the same format as if the method was called.
	if !stream.EqualToken(resp[0], stream.Call) ||
		!stream.EqualBytes(resp[1], uid.InvokeIDSMU[:]) ||
		!stream.EqualBytes(resp[2], uid.MethodIDSMSyncSession[:]) ||
		len(params) < 2 ||
		!ok {
		// This is very serious, but can happen given that we might be using a shared ComID
		return nil, ErrInvalidStartSessionResponse
	}

	// First parameter, required, TPer properties
	hsn, ok1 := params[0].(uint)
	tsn, ok2 := params[1].(uint)
	// TODO: other properties may be returned here
	// TODO: Send InitialCredits if required

	if !ok1 || !ok2 || int(hsn) != s.HSN {
		return nil, ErrInvalidStartSessionResponse
	}

	s.TSN = int(tsn)
	return s, nil
}

// Fetch current Host and TPer properties, optionally changing the Host properties.
func (cs *ControlSession) properties(rhp *HostProperties) (HostProperties, TPerProperties, error) {
	mc := method.NewMethodCall(uid.InvokeIDSMU, uid.MethodIDSMProperties, cs.Session.MethodFlags)

	mc.StartOptionalParameter(0, "HostProperties")
	mc.StartList()
	mc.NamedUInt("MaxMethods", rhp.MaxMethods)
	mc.NamedUInt("MaxSubpackets", rhp.MaxSubpackets)
	mc.NamedUInt("MaxPacketSize", rhp.MaxPacketSize)
	mc.NamedUInt("MaxPackets", rhp.MaxPackets)
	mc.NamedUInt("MaxComPacketSize", rhp.MaxComPacketSize)
	if rhp.MaxResponseComPacketSize != nil {
		mc.NamedUInt("MaxResponseComPacketSize", *rhp.MaxResponseComPacketSize)
	}
	mc.NamedUInt("MaxIndTokenSize", rhp.MaxIndTokenSize)
	mc.NamedUInt("MaxAggTokenSize", rhp.MaxAggTokenSize)
	mc.NamedBool("ContinuedTokens", rhp.ContinuedTokens)
	mc.NamedBool("SequenceNumbers", rhp.SequenceNumbers)
	mc.NamedBool("AckNak", rhp.AckNak)
	mc.NamedBool("Asynchronous", rhp.Asynchronous)
	mc.EndList()
	mc.EndOptionalParameter()

	resp, err := cs.ExecuteMethod(mc)
	if err != nil {
		return HostProperties{}, TPerProperties{}, err
	}

	if len(resp) != 4 {
		return HostProperties{}, TPerProperties{}, ErrInvalidPropertiesResponse
	}
	params, ok := resp[3].(stream.List)

	// See "5.2.2.1.2 Properties Response".
	// The returned response is in the same format as if the method was called.
	if !stream.EqualToken(resp[0], stream.Call) ||
		!stream.EqualBytes(resp[1], uid.InvokeIDSMU[:]) ||
		!stream.EqualBytes(resp[2], uid.MethodIDSMProperties[:]) ||
		!ok ||
		len(params) != 5 {
		// This is very serious, but can happen given that we might be using a shared ComID
		return HostProperties{}, TPerProperties{}, ErrInvalidPropertiesResponse
	}

	hp := InitialHostProperties
	tp := InitialTPerProperties

	// First parameter, required, TPer properties
	tpParams, ok1 := params[0].(stream.List)
	// Second parameter is optional, skip the BeginName + param ID
	hpParams, ok2 := params[3].(stream.List)
	if !ok1 || !ok2 {
		return HostProperties{}, TPerProperties{}, ErrInvalidPropertiesResponse
	}
	if err := parseTPerProperties(tpParams, &tp); err != nil {
		return HostProperties{}, TPerProperties{}, err
	}
	if err := parseHostProperties(hpParams, &hp); err != nil {
		return HostProperties{}, TPerProperties{}, err
	}

	// TODO: Ensure that the returned parameters are not lower than the minimum
	// allowed values.
	return hp, tp, nil
}

func (cs *ControlSession) Close() error {
	// Control sessions cannot be closed
	return nil
}

func (s *Session) Close() error {
	if s.closed {
		return ErrSessionAlreadyClosed
	}
	s.closed = true
	if err := s.c.Send(s, stream.Token(stream.EndOfSession)); err != nil {
		return err
	}

	for i := s.ReceiveRetries; i >= 0; i-- {
		resp, err := s.c.Receive(s)
		if err != nil {
			return err
		}
		if len(resp) > 0 {
			if !stream.EqualToken(resp, stream.EndOfSession) {
				return fmt.Errorf("expected EOS, received other data")
			}
			break
		}
		if i == 0 {
			return method.ErrMethodTimeout
		}
		time.Sleep(s.ReceiveInterval)
	}
	return nil
}

func (s *Session) ExecuteMethod(mc *method.MethodCall) (stream.List, error) {
	if s.closed {
		return nil, ErrSessionAlreadyClosed
	}
	b, err := mc.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Synchronous mode specific: Ensure that there is no pending message
	// before we start.
	resp, err := s.c.Receive(s)
	if err != nil {
		return nil, err
	}
	if len(resp) > 0 {
		return nil, method.ErrReceivedUnexpectedResponse
	}

	if err = s.c.Send(s, b); err != nil {
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

	for i := s.ReceiveRetries; i >= 0; i-- {
		resp, err = s.c.Receive(s)
		if err != nil {
			return nil, err
		}
		if len(resp) > 0 {
			break
		}
		if i == 0 {
			return nil, method.ErrMethodTimeout
		}
		time.Sleep(s.ReceiveInterval)
	}

	reply, err := stream.Decode(resp)
	if err != nil {
		return nil, err
	}

	if len(reply) < 2 {
		return nil, method.ErrEmptyMethodResponse
	}

	// Check for special CloseSession response
	if len(reply) >= 4 {
		tok, ok1 := reply[0].(stream.TokenType)
		iid, ok2 := reply[1].([]byte)
		mid, ok3 := reply[2].([]byte)
		params, ok4 := reply[3].(stream.List)
		if ok1 && ok2 && ok3 && ok4 &&
			tok == stream.Call &&
			bytes.Equal(iid, uid.InvokeIDSMU[:]) &&
			bytes.Equal(mid, uid.MethodIDSMCloseSession[:]) {
			hsn, ok1 := params[0].(uint)
			tsn, ok2 := params[1].(uint)
			if ok1 && ok2 && int(hsn) == s.HSN && int(tsn) == s.TSN {
				return nil, method.ErrTPerClosedSession
			} else {
				return nil, method.ErrReceivedUnexpectedResponse
			}
		}
	}

	// While the normal method result format is known, the Session Manager
	// methods use a different format. What is in common however is that
	// the last element should be the status code list.
	tok, ok1 := reply[len(reply)-2].(stream.TokenType)
	status, ok2 := reply[len(reply)-1].(stream.List)
	if !ok1 || !ok2 || tok != stream.EndOfData {
		return nil, method.ErrMalformedMethodResponse
	}

	sc, ok := status[0].(uint)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
	}
	if sc != method.MethodStatusSuccess {
		err, ok := method.MethodStatusCodeMap[sc]
		if !ok {
			return nil, fmt.Errorf("method returned unknown status code 0x%02x", sc)
		}
		return nil, err
	}

	return reply[:len(reply)-2], nil
}

// Execute a prepared Method call but do not expect anything in return.
func (s *Session) Notify(mc *method.MethodCall) error {
	b, err := mc.MarshalBinary()
	if err != nil {
		return err
	}
	if err = s.c.Send(s, b); err != nil {
		return err
	}
	return nil
}

func parseTPerProperties(params []interface{}, tp *TPerProperties) error {
	for i, p := range params {
		if stream.EqualToken(p, stream.StartName) {
			n, ok1 := params[i+1].([]byte)
			v, ok2 := params[i+2].(uint)
			if !ok1 || !ok2 {
				return fmt.Errorf("tper properties malformed")
			}
			switch string(n) {
			case "MaxMethods":
				tp.MaxMethods = v
			case "MaxSubpackets":
				tp.MaxSubpackets = v
			case "MaxPacketSize":
				tp.MaxPacketSize = v
			case "MaxPackets":
				tp.MaxPackets = v
			case "MaxComPacketSize":
				tp.MaxComPacketSize = v
			case "MaxResponseComPacketSize":
				tp.MaxResponseComPacketSize = &v
			case "MaxSessions":
				tp.MaxSessions = &v
			case "MaxReadSessions":
				tp.MaxReadSessions = &v
			case "MaxIndTokenSize":
				tp.MaxIndTokenSize = v
			case "MaxAggTokenSize":
				tp.MaxAggTokenSize = v
			case "MaxAuthentications":
				tp.MaxAuthentications = &v
			case "MaxTransactionLimit":
				tp.MaxTransactionLimit = &v
			case "DefSessionTimeout":
				tp.DefSessionTimeout = &v
			case "MaxSessionTimeout":
				tp.MaxSessionTimeout = &v
			case "MinSessionTimeout":
				tp.MinSessionTimeout = &v
			case "DefTransTimeout":
				tp.DefTransTimeout = &v
			case "MaxTransTimeout":
				tp.MaxTransTimeout = &v
			case "MinTransTimeout":
				tp.MinTransTimeout = &v
			case "MaxComIDTime":
				tp.MaxComIDTime = &v
			case "ContinuedTokens":
				tp.ContinuedTokens = v > 0
			case "SequenceNumbers":
				tp.SequenceNumbers = v > 0
			case "AckNak":
				tp.AckNak = v > 0
			case "Asynchronous":
				tp.Asynchronous = v > 0
			}
		}
	}
	return nil
}

func parseHostProperties(params []interface{}, hp *HostProperties) error {
	for i, p := range params {
		if stream.EqualToken(p, stream.StartName) {
			n, ok1 := params[i+1].([]byte)
			v, ok2 := params[i+2].(uint)
			if !ok1 || !ok2 {
				return fmt.Errorf("host properties malformed")
			}
			switch string(n) {
			case "MaxMethods":
				hp.MaxMethods = v
			case "MaxSubpackets":
				hp.MaxSubpackets = v
			case "MaxPacketSize":
				hp.MaxPacketSize = v
			case "MaxPackets":
				hp.MaxPackets = v
			case "MaxComPacketSize":
				hp.MaxComPacketSize = v
			case "MaxResponseComPacketSize":
				hp.MaxResponseComPacketSize = &v
			case "MaxIndTokenSize":
				hp.MaxIndTokenSize = v
			case "MaxAggTokenSize":
				hp.MaxAggTokenSize = v
			case "ContinuedTokens":
				hp.ContinuedTokens = v > 0
			case "SequenceNumbers":
				hp.SequenceNumbers = v > 0
			case "AckNak":
				hp.AckNak = v > 0
			case "Asynchronous":
				hp.Asynchronous = v > 0
			}
		}
	}
	return nil
}

func init() {
	sessionRand = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
}
