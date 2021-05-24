// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core - Session Manager and Session

package tcgstorage

import (
	"fmt"

	"github.com/bluecmd/go-tcg-storage/drive"
)

type Session struct {
	d        DriveIntf
	c        CommunicationIntf
	ComID    ComID
	TSN, HSN int
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
	InitialHostProperties = HostProperties{
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
)

// Initiate a new session with a Security Provider
//
// TODO: Let's see if this API makes sense...
func NewSession(d DriveIntf, comID ComID) (*Session, error) {
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

	// This means that the first we do should be to set the HostProperties to sane (for us)
	// values, and start a session ASAP to persist those. We assume the current HostProperties
	// are set to the lowest values.
	hp := InitialHostProperties
	tp := InitialTPerProperties
	c := NewPlainCommunication(d, hp, tp)
	s := &Session{
		d:     d,
		c:     c,
		ComID: comID,
		TSN:   0,
		HSN:   1337, /* TODO: Provided by the caller? Allow "WithHSN()" modifier? */
	}

	var err error
	hp, tp, err = s.properties(&hp)
	if err != nil {
		return nil, err
	}

	// TODO: Start session

	return s, nil
}

func (s *Session) properties(hp *HostProperties) (HostProperties, TPerProperties, error) {
	mc := NewMethodCall(InvokeIDSMU, MethodIDProperties)

	// TODO
	// DtaCommand *props = new DtaCommand(OPAL_UID::OPAL_SMUID_UID, OPAL_METHOD::PROPERTIES);
	// props->addToken(OPAL_TOKEN::STARTLIST);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("HostProperties");
	// props->addToken(OPAL_TOKEN::STARTLIST);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("MaxComPacketSize");
	// props->addToken(2048);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("MaxPacketSize");
	// props->addToken(2028);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("MaxIndTokenSize");
	// props->addToken(1992);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("MaxPackets");
	// props->addToken(1);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("MaxSubpackets");
	// props->addToken(1);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::STARTNAME);
	// props->addToken("MaxMethods");
	// props->addToken(1);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::ENDLIST);
	// props->addToken(OPAL_TOKEN::ENDNAME);
	// props->addToken(OPAL_TOKEN::ENDLIST);
	resp, err := mc.Execute(s.c, drive.SecurityProtocolTCGManagement, s)
	if err != nil {
		return HostProperties{}, TPerProperties{}, err
	}

	fmt.Printf("properties resp: %+v\n", resp)
	return HostProperties{}, TPerProperties{}, nil
}
