// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// High-level locking API for TCG Storage devices

package locking

import (
	"fmt"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/table"
)

var (
	LifeCycleStateManufacturedInactive table.LifeCycleState = 8
	LifeCycleStateManufactured         table.LifeCycleState = 9

	LockingAuthorityBandMaster0 core.AuthorityObjectUID = [8]byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x80, 0x01}
	LockingAuthorityAdmin1      core.AuthorityObjectUID = [8]byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x01, 0x00, 0x01}
)

type LockingSP struct {
	Session *core.Session
	// All authorities that have been discovered on the SP.
	// This will likely be only the authenticated UID unless authorized as an Admin
	Authorities map[string]core.AuthorityObjectUID
	// The full range of Ranges (heh!) that the current session has access to see and possibly modify
	GlobalRange *Range
	Ranges      []*Range // Ranges[0] == GlobalRange

	// These are always false on SSC Enterprise
	MBREnabled     bool
	MBRDone        bool
	MBRDoneOnReset []table.ResetType
}

func (l *LockingSP) Close() error {
	return l.Session.Close()
}

type AdminSPAuthenticator interface {
	AuthenticateAdminSP(s *core.Session) error
}
type LockingSPAuthenticator interface {
	AuthenticateLockingSP(s *core.Session, lmeta *LockingSPMeta) error
}

var (
	DefaultAuthorityWithMSID = &authority{}
)

type authority struct {
	auth  []byte
	proof []byte
}

func (a *authority) AuthenticateAdminSP(s *core.Session) error {
	var auth core.AuthorityObjectUID
	if len(a.auth) == 0 {
		copy(auth[:], core.AuthoritySID[:])
	} else {
		copy(auth[:], a.auth)
	}
	if len(a.proof) == 0 {
		// TODO: Verify with C_PIN behavior and Block SID
		msidPin, err := table.Admin_C_PIN_MSID_GetPIN(s)
		if err != nil {
			return err
		}
		return table.ThisSP_Authenticate(s, auth, msidPin)
	} else {
		return table.ThisSP_Authenticate(s, auth, a.proof)
	}
}

func (a *authority) AuthenticateLockingSP(s *core.Session, lmeta *LockingSPMeta) error {
	var auth core.AuthorityObjectUID
	if len(a.auth) == 0 {
		if s.ProtocolLevel == core.ProtocolLevelEnterprise {
			copy(auth[:], LockingAuthorityBandMaster0[:])
		} else {
			copy(auth[:], LockingAuthorityAdmin1[:])
		}
	} else {
		copy(auth[:], a.auth)
	}
	if len(a.proof) == 0 {
		if len(lmeta.MSID) == 0 {
			return fmt.Errorf("authentication via MSID disabled")
		}
		return table.ThisSP_Authenticate(s, auth, lmeta.MSID)
	} else {
		return table.ThisSP_Authenticate(s, auth, a.proof)
	}
}

func DefaultAuthority(proof []byte) *authority {
	return &authority{proof: proof}
}

func DefaultAdminAuthority(proof []byte) *authority {
	return &authority{proof: proof}
}

func AuthorityFromName(user string, proof []byte) (*authority, bool) {
	return nil, false
}

func NewSession(cs *core.ControlSession, lmeta *LockingSPMeta, auth LockingSPAuthenticator, opts ...core.SessionOpt) (*LockingSP, error) {
	if lmeta.D0.Locking == nil {
		return nil, fmt.Errorf("device does not have the Locking feature")
	}
	s, err := cs.NewSession(lmeta.SPID, opts...)
	if err != nil {
		return nil, fmt.Errorf("session creation failed: %v", err)
	}

	if err := auth.AuthenticateLockingSP(s, lmeta); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	l := &LockingSP{Session: s}

	// TODO: These can be read from the LockingSP instead, it would be cleaner
	// to not have to drag D0 in the SPMeta.
	l.MBRDone = lmeta.D0.Locking.MBRDone
	l.MBREnabled = lmeta.D0.Locking.MBREnabled
	// TODO: Set MBRDoneOnReset to real value
	l.MBRDoneOnReset = []table.ResetType{table.ResetPowerOff}

	if err := fillRanges(s, l); err != nil {
		return nil, err
	}

	// TODO: Fill l.Authorities with known users for admin actions
	return l, nil
}

type initializeConfig struct {
	auths    []AdminSPAuthenticator
	activate bool
}

type InitializeOpt func(ic *initializeConfig)

func WithAuth(auth AdminSPAuthenticator) InitializeOpt {
	return func(ic *initializeConfig) {
		ic.auths = append(ic.auths, auth)
	}
}

func findComID(d core.DriveIntf, d0 *core.Level0Discovery) (core.ComID, core.ProtocolLevel, error) {
	proto := core.ProtocolLevelUnknown
	comID := core.ComIDInvalid
	if d0.OpalV2 != nil {
		comID = core.ComID(d0.OpalV2.BaseComID)
		proto = core.ProtocolLevelCore
	} else if d0.PyriteV1 != nil {
		comID = core.ComID(d0.PyriteV1.BaseComID)
		proto = core.ProtocolLevelCore
	} else if d0.PyriteV2 != nil {
		comID = core.ComID(d0.PyriteV2.BaseComID)
		proto = core.ProtocolLevelCore
	} else if d0.Enterprise != nil {
		comID = core.ComID(d0.Enterprise.BaseComID)
		proto = core.ProtocolLevelEnterprise
	}

	autoComID, err := core.GetComID(d)
	if err == nil && autoComID > 0 {
		comID = autoComID
	}

	return comID, proto, nil
}

type LockingSPMeta struct {
	SPID core.SPID
	MSID []byte
	D0   *core.Level0Discovery
}

func Initialize(d core.DriveIntf, opts ...InitializeOpt) (*core.ControlSession, *LockingSPMeta, error) {
	var ic initializeConfig
	for _, o := range opts {
		o(&ic)
	}

	lmeta := &LockingSPMeta{}
	d0, err := core.Discovery0(d)
	if err != nil {
		return nil, nil, fmt.Errorf("discovery feiled: %v", err)
	}
	lmeta.D0 = d0

	comID, proto, err := findComID(d, d0)
	if err != nil {
		return nil, nil, err
	}
	cs, err := core.NewControlSession(d, d0, core.WithComID(comID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create control session (comID 0x%04x): %v", comID, err)
	}

	as, err := cs.NewSession(core.AdminSP)
	if err != nil {
		return nil, nil, fmt.Errorf("admin session creation failed: %v", err)
	}
	defer as.Close()

	err = nil
	for _, x := range ic.auths {
		if err = x.AuthenticateAdminSP(as); err == table.ErrAuthenticationFailed {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		break
	}
	if err != nil {
		return nil, nil, fmt.Errorf("all authentications failed")
	}

	if proto == core.ProtocolLevelEnterprise {
		copy(lmeta.SPID[:], core.EnterpriseLockingSP[:])
		if err := initializeEnterprise(as, d0, &ic, lmeta); err != nil {
			return nil, nil, err
		}
	} else {
		copy(lmeta.SPID[:], core.LockingSP[:])
		if err := initializeOpalFamily(as, d0, &ic, lmeta); err != nil {
			return nil, nil, err
		}
	}
	return cs, lmeta, nil
}

func initializeEnterprise(s *core.Session, d0 *core.Level0Discovery, ic *initializeConfig, lmeta *LockingSPMeta) error {
	msidPin, err := table.Admin_C_PIN_MSID_GetPIN(s)
	if err == nil {
		lmeta.MSID = msidPin
	}
	// TODO: Implement take ownership for enterprise if activated in initializeConfig.
	// The spec should explain what is needed.
	// TODO: If initializeConfig wants WithHardended, implement relevant
	// FIPS recommendations.
	return nil
}

func initializeOpalFamily(s *core.Session, d0 *core.Level0Discovery, ic *initializeConfig, lmeta *LockingSPMeta) error {
	// TODO: Verify with C_PIN behavior and Block SID - no need to burn PIN tries
	// if we can say that MSID will not work.
	msidPin, err := table.Admin_C_PIN_MSID_GetPIN(s)
	if err == nil {
		lmeta.MSID = msidPin
	}
	// TODO: Take ownership (*before* Activate to ensure that the PINs are copied)
	// This is explained in the spec.
	lcs, err := table.Admin_SP_GetLifeCycleState(s, core.LockingSP)
	if err != nil {
		return err
	}
	if lcs == LifeCycleStateManufactured {
		// The Locking SP is already activated
		return nil
	} else if lcs == LifeCycleStateManufacturedInactive {
		if !ic.activate {
			return fmt.Errorf("locking SP not active, but activation not requested")
		}
		mc := s.NewMethodCall(core.InvokingID(core.LockingSP), table.MethodIDAdmin_Activate)
		if _, err := s.ExecuteMethod(mc); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported life cycle state on locking SP: %v", lcs)
	}

	// TODO: If initializeConfig wants WithHardended, implement relevant
	// FIPS recommendations.
	return nil
}

func (l *LockingSP) SetMBRDone(v bool) error {
	mbr := &table.MBRControl{Done: &v}
	return table.MBRControl_Set(l.Session, mbr)
}
