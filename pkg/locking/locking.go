// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// High-level locking API for TCG Storage devices

package locking

import (
	"fmt"
	"time"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/method"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/table"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/uid"
)

var (
	LifeCycleStateManufacturedInactive table.LifeCycleState = 8
	LifeCycleStateManufactured         table.LifeCycleState = 9
)

type LockingSP struct {
	Session *core.Session
	// All authorities that have been discovered on the SP.
	// This will likely be only the authenticated UID unless authorized as an Admin
	Authorities map[string]uid.AuthorityObjectUID
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
	var auth uid.AuthorityObjectUID
	if len(a.auth) == 0 {
		copy(auth[:], uid.AuthoritySID[:])
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
	var auth uid.AuthorityObjectUID
	if len(a.auth) == 0 {
		if s.ProtocolLevel == core.ProtocolLevelEnterprise {
			copy(auth[:], uid.LockingAuthorityBandMaster0[:])
		} else {
			copy(auth[:], uid.LockingAuthorityAdmin1[:])
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
	auths                    []AdminSPAuthenticator
	activate                 bool
	MaxComPacketSizeOverride uint
	ReceiveRetries           int
	ReceiveInterval          time.Duration
}

type InitializeOpt func(ic *initializeConfig)

func WithAuth(auth AdminSPAuthenticator) InitializeOpt {
	return func(ic *initializeConfig) {
		ic.auths = append(ic.auths, auth)
	}
}

func WithMaxComPacketSize(size uint) InitializeOpt {
	return func(s *initializeConfig) {
		s.MaxComPacketSizeOverride = size
	}
}

func WithReceiveTimeout(retries int, interval time.Duration) InitializeOpt {
	return func(ic *initializeConfig) {
		ic.ReceiveRetries = retries
		ic.ReceiveInterval = interval
	}
}

type LockingSPMeta struct {
	SPID uid.SPID
	MSID []byte
	D0   *core.Level0Discovery
}

// Initialize WHAT?
func Initialize(coreObj *core.Core, opts ...InitializeOpt) (*core.ControlSession, *LockingSPMeta, error) {
	ic := initializeConfig{
		MaxComPacketSizeOverride: core.DefaultMaxComPacketSize,
		ReceiveRetries:           core.DefaultReceiveRetries,
		ReceiveInterval:          core.DefaultReceiveInterval,
	}
	for _, o := range opts {
		o(&ic)
	}

	lmeta := &LockingSPMeta{}
	lmeta.D0 = coreObj.DiskInfo.Level0Discovery

	comID, proto, err := core.FindComID(coreObj.DriveIntf, coreObj.DiskInfo.Level0Discovery)
	if err != nil {
		return nil, nil, err
	}
	controlSessionOpts := []core.ControlSessionOpt{
		core.WithComID(comID),
		core.WithMaxComPacketSize(ic.MaxComPacketSizeOverride),
		core.WithReceiveTimeout(ic.ReceiveRetries, ic.ReceiveInterval),
	}

	cs, err := core.NewControlSession(coreObj.DriveIntf, coreObj.DiskInfo.Level0Discovery, controlSessionOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create control session (comID 0x%04x): %v", comID, err)
	}

	as, err := cs.NewSession(uid.AdminSP)
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
		copy(lmeta.SPID[:], uid.EnterpriseLockingSP[:])
		if err := initializeEnterprise(as, coreObj.DiskInfo.Level0Discovery, &ic, lmeta); err != nil {
			return nil, nil, err
		}
	} else {
		copy(lmeta.SPID[:], uid.LockingSP[:])
		if err := initializeOpalFamily(as, coreObj.DiskInfo.Level0Discovery, &ic, lmeta); err != nil {
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
	lcs, err := table.Admin_SP_GetLifeCycleState(s, uid.LockingSP)
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
		mc := method.NewMethodCall(uid.InvokingID(uid.LockingSP), uid.MethodIDAdmin_Activate, s.MethodFlags)
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
