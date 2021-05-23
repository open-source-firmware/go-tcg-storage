// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core - Session Manager and Session

package tcgstorage

import (
	"fmt"

	"github.com/bluecmd/go-tcg-storage/drive"
)

type SessionManager struct {
	d DriveIntf
	c CommunicationIntf
}

type Session struct {
	ComID    ComID
	TSN, HSN int
}

func NullSession(comID ComID) *Session {
	return &Session{ComID: comID, TSN: 0, HSN: 1337 /* TODO: Random */}
}

type TPerProperties struct {
	// TODO
}

func NewSessionManager(d DriveIntf) *SessionManager {
	// TODO: The idea here is to allow upgrading a Session like "secses = ses.Secure()"
	return &SessionManager{d: d, c: NewPlainCommunication(d)}
}

func (sm *SessionManager) Properties() (*TPerProperties, error) {
	comID := /* What ID? */ ComID(0)
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
	resp, err := mc.Execute(sm.c, drive.SecurityProtocolTCGManagement, NullSession(comID))
	if err != nil {
		return nil, err
	}

	fmt.Printf("properties resp: %+v\n", resp)
	return &TPerProperties{}, nil
}
