// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// High-level locking API for TCG Storage devices

package locking

import (
	"fmt"

	"github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/core/table"
)

var (
	LifeCycleStateManufacturedInactive table.LifeCycleState = 8
	LifeCycleStateManufactured         table.LifeCycleState = 9

	LockingAuthorityBandMaster0 core.AuthorityObjectUID = [8]byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x80, 0x01}
	LockingAuthorityAdmin1      core.AuthorityObjectUID = [8]byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x01}
)

type Range struct {
}

type Locking struct {
	Session *core.Session
}

func (l *Locking) Close() error {
	return l.Session.Close()
}

type Authenticator interface {
	Authenticate(s *core.Session) error
}

var (
	DefaultAuthority = &defLockingAuthority{}
)

type defLockingAuthority struct {
}

func (a *defLockingAuthority) Authenticate(s *core.Session) error {
	var auth core.AuthorityObjectUID
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		// BandMaster0
		copy(auth[:], LockingAuthorityBandMaster0[:])
	} else {
		// Admin1
		copy(auth[:], LockingAuthorityAdmin1[:])
	}
	msidPin, err := msidWithSanity(s)
	if err != nil {
		return err
	}
	return table.ThisSP_Authenticate(s, auth, msidPin)
}

func msidWithSanity(s *core.Session) ([]byte, error) {
	// TODO: Fail if C_PIN behavior is not MSID
	// TODO: Fail if Block SID is enabled
	return table.Admin_C_PIN_MSID_GetPIN(s)
}

func NewSession(cs *core.ControlSession, spid core.SPID, auth Authenticator, opts ...core.SessionOpt) (*Locking, error) {
	s, err := cs.NewSession(spid, opts...)
	if err != nil {
		return nil, fmt.Errorf("session creation failed: %v", err)
	}

	if err := auth.Authenticate(s); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	l := &Locking{Session: s}

	lockList, err := table.Locking_Enumerate(s)
	if err != nil {
		return nil, fmt.Errorf("enumerate ranges failed: %v", err)
	}
	fmt.Printf("Locking regions:")
	for _, luid := range lockList {
		lr, err := table.Locking_Get(s, luid)
		if err != nil {
			fmt.Printf("Region %v: <UNKNOWN> (%v)\n", luid[:], err)
		} else {
			fmt.Printf("Region %v: %+v\n", luid[:], lr)
		}
	}

	return l, nil
}

type initializeConfig struct {
	auths    []Authenticator
	activate bool
}

type InitializeOpts func(ic *initializeConfig)

func WithAuth(auth Authenticator) InitializeOpts {
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
	if err == nil {
		comID = autoComID
	}

	valid, err := core.IsComIDValid(d, comID)
	if err != nil {
		return core.ComIDInvalid, core.ProtocolLevelUnknown, fmt.Errorf("comID validation failed: %v", err)
	}

	if !valid {
		return core.ComIDInvalid, core.ProtocolLevelUnknown, fmt.Errorf("allocated comID was not valid")
	}

	return comID, proto, nil
}

func Initialize(d core.DriveIntf, opts ...InitializeOpts) (*core.ControlSession, core.SPID, error) {
	var ic initializeConfig
	for _, o := range opts {
		o(&ic)
	}

	var spid core.SPID
	d0, err := core.Discovery0(d)
	if err != nil {
		return nil, spid, fmt.Errorf("discovery feiled: %v", err)
	}

	comID, proto, err := findComID(d, d0)
	cs, err := core.NewControlSession(d, d0, core.WithComID(comID))
	if err != nil {
		return nil, spid, fmt.Errorf("failed to create control session: %v", err)
	}

	as, err := cs.NewSession(core.AdminSP)
	if err != nil {
		return nil, spid, fmt.Errorf("admin session creation failed: %v", err)
	}
	defer as.Close()

	// TODO:
	//if err := auth.Authenticate(s); err != nil {
	//	return nil, fmt.Errorf("admin authentication failed: %v", err)
	//}

	if proto == core.ProtocolLevelEnterprise {
		copy(spid[:], core.EnterpriseLockingSP[:])
		if err := initializeEnterprise(as, d0, &ic); err != nil {
			return nil, spid, err
		}
	} else {
		copy(spid[:], core.LockingSP[:])
		if err := initializeOpalFamily(as, d0, &ic); err != nil {
			return nil, spid, err
		}
	}

	// TODO: Take ownership

	return cs, spid, nil
}

func initializeEnterprise(s *core.Session, d0 *core.Level0Discovery, ic *initializeConfig) error {
	// TODO: lockdown
	return nil
}

func initializeOpalFamily(s *core.Session, d0 *core.Level0Discovery, ic *initializeConfig) error {
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

	// TODO: lockdown
	return nil
}
