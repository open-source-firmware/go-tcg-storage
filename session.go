// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core - Session Manager and Session

package tcgstorage

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/bluecmd/go-tcg-storage/drive"
)

var (
	ErrTPerSyncNotSupported = errors.New("synchronous operation not supported by TPer")

	InvokeIDSMU InvokingID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}

	// Table 241 - "Session Manager Method UIDs"
	MethodIDSMProperties          MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x01}
	MethodIDSMStartSession        MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x02}
	MethodIDSMSyncSession         MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x03}
	MethodIDSMStartTrustedSession MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x04}
	MethodIDSMSyncTrustedSession  MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x05}
	MethodIDSMCloseSession        MethodID = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x06}
)

type Session struct {
	ControlSession *ControlSession
	d              DriveIntf
	c              CommunicationIntf
	ComID          ComID
	TSN, HSN       int
	// See "3.2.3.3.1.2 SeqNumber"
	SeqLastXmit     int
	SeqLastAcked    int
	SeqNextExpected int
}

type ControlSession struct {
	Session
	HostProperties HostProperties
	TPerProperties TPerProperties
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

func WithHSN(hsn int) SessionOpt {
	return func(s *Session) {
		s.HSN = hsn
	}
}

//func NewOpalV2Session(d DriveIntf, tper *FeatureTPer, opal *FeatureOpalV2) (*Session, error) {
//	return NewSession(d, tper, opal.BaseComID)
//}

// Initiate a new control session with a ComID.
func NewControlSession(d DriveIntf, tper *FeatureTPer, opts ...ControlSessionOpt) (*ControlSession, error) {
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

	if !tper.SyncSupported {
		return nil, ErrTPerSyncNotSupported
	}

	hp := InitialHostProperties
	tp := InitialTPerProperties
	c := NewPlainCommunication(d, hp, tp)
	s := &ControlSession{
		Session: Session{
			d:     d,
			c:     c,
			ComID: ComIDInvalid,
			TSN:   0,
			HSN:   0,
		},
		HostProperties: hp,
		TPerProperties: tp,
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

	// Try to reset the synchronous protocol stack for the ComID to minimize
	// the dependencies on the implicit state. However, I suspect not all drives
	// implement it so we do it best-effort.
	StackReset(d, s.ComID)

	var err error
	hp, tp, err = s.properties(&hp)
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
func (cs *ControlSession) NewSession(opts ...SessionOpt) (*Session, error) {
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
		d:     cs.d,
		c:     cs.c,
		ComID: cs.ComID,
		TSN:   0,
		HSN:   -1,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.HSN > 0xffffffff {
		return nil, fmt.Errorf("too large HSN provided")
	}

	if s.HSN == -1 {
		s.HSN = int(rand.Int31())
	}

	// TODO: Start session

	return s, fmt.Errorf("session start-up not implemented")
}

// Fetch current Host and TPer properties, optionally changing the Host properties.
func (cs *ControlSession) properties(rhp *HostProperties) (HostProperties, TPerProperties, error) {
	mc := NewMethodCall(InvokeIDSMU, MethodIDSMProperties)

	mc.PushToken(StreamStartList)
	// TODO: Include host parameters
	// TOKEN::STARTLIST
	// TOKEN::STARTNAME
	// "HostProperties"
	// TOKEN::STARTLIST
	// TOKEN::STARTNAME
	// "MaxComPacketSize"
	// 2048
	// TOKEN::ENDNAME
	// TOKEN::STARTNAME
	// "MaxPacketSize"
	// 2028
	// TOKEN::ENDNAME
	// TOKEN::STARTNAME
	// "MaxIndTokenSize"
	// 1992
	// TOKEN::ENDNAME
	// TOKEN::STARTNAME
	// "MaxPackets"
	// 1
	// TOKEN::ENDNAME
	// TOKEN::STARTNAME
	// "MaxSubpackets"
	// 1
	// TOKEN::ENDNAME
	// TOKEN::STARTNAME
	// "MaxMethods"
	// 1
	// TOKEN::ENDNAME
	// TOKEN::ENDLIST
	// TOKEN::ENDNAME
	// TOKEN::ENDLIST
	mc.PushToken(StreamEndList)

	resp, err := mc.Execute(cs.c, drive.SecurityProtocolTCGManagement, &cs.Session)
	if err != nil {
		return HostProperties{}, TPerProperties{}, err
	}

	hp := InitialHostProperties
	tp := InitialTPerProperties

	_ = resp
	// TODO: Parse returned tokens into Host and TPer properties
	// TODO: Ensure that the returned parameters are not lower than the minimum
	// allowed values.

	fmt.Printf("hp: %+v\n", hp)
	fmt.Printf("tp: %+v\n", tp)
	return hp, tp, fmt.Errorf("properties parsing not implemented")
}

func (cs *ControlSession) Close() error {
	// Control sessions cannot be closed
	return nil
}

func (s *Session) Close() error {
	// TODO
	return fmt.Errorf("session close not implemented")
}
