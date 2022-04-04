// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/table"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

func TestComID(d tcg.DriveIntf) tcg.ComID {
	comID, err := tcg.GetComID(d)
	if err != nil {
		log.Printf("Unable to auto-allocate ComID: %v", err)
		return tcg.ComIDInvalid
	}
	log.Printf("Allocated ComID 0x%08x", comID)
	valid, err := tcg.IsComIDValid(d, comID)
	if err != nil {
		log.Printf("Unable to validate allocated ComID: %v", err)
		return tcg.ComIDInvalid
	}
	if !valid {
		log.Printf("Allocated ComID not valid")
		return tcg.ComIDInvalid
	}
	log.Printf("ComID validated successfully")

	if err := tcg.StackReset(d, comID); err != nil {
		log.Printf("Unable to reset the synchronous protocol stack: %v", err)
		return tcg.ComIDInvalid
	}
	log.Printf("Synchronous protocol stack reset successfully")
	return comID
}

func TestControlSession(d tcg.DriveIntf, d0 *tcg.Level0Discovery, comID tcg.ComID) *tcg.ControlSession {
	if comID == tcg.ComIDInvalid {
		log.Printf("Auto-allocation ComID test failed earlier, selecting first available base ComID")
		if d0.OpalV2 != nil {
			log.Printf("Selecting OpalV2 ComID")
			comID = tcg.ComID(d0.OpalV2.BaseComID)
		} else if d0.PyriteV1 != nil {
			log.Printf("Selecting PyriteV1 ComID")
			comID = tcg.ComID(d0.PyriteV1.BaseComID)
		} else if d0.PyriteV2 != nil {
			log.Printf("Selecting PyriteV2 ComID")
			comID = tcg.ComID(d0.PyriteV2.BaseComID)
		} else if d0.Enterprise != nil {
			log.Printf("Selecting Enterprise ComID")
			comID = tcg.ComID(d0.Enterprise.BaseComID)
		} else {
			log.Printf("No supported feature found, giving up without a ComID ...")
			return nil
		}
	}
	log.Printf("Creating control session with ComID 0x%08x\n", comID)
	cs, err := tcg.NewControlSession(d, d0, tcg.WithComID(comID))
	if err != nil {
		log.Printf("s.NewControlSession failed: %v", err)
		return nil
	}
	log.Printf("Operating using protocol %q", cs.ProtocolLevel.String())
	log.Printf("Negotiated TPerProperties:")
	spew.Dump(cs.TPerProperties)
	log.Printf("Negotiated HostProperties:")
	spew.Dump(cs.HostProperties)
	// TODO: Move this to a test case instead
	if err := cs.Close(); err != nil {
		log.Fatalf("Test of ControlSession Close failed: %v", err)
	}
	return cs
}

func main() {
	spew.Config.Indent = "  "

	d, err := drive.Open(os.Args[1])
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer d.Close()

	fmt.Printf("===> DRIVE SECURITY INFORMATION\n")
	id, err := d.Identify()
	if err != nil {
		log.Fatalf("drive.Identity: %v", err)
	}
	log.Printf("Drive identity: %s", id)
	spl, err := drive.SecurityProtocols(d)
	if err != nil {
		log.Fatalf("drive.SecurityProtocols: %v", err)
	}
	log.Printf("SecurityProtocols: %+v", spl)
	crt, err := drive.Certificate(d)
	if err != nil {
		log.Printf("drive.Certificate: %v", err)
	}
	log.Printf("Drive certificate:")
	spew.Dump(crt)
	fmt.Printf("\n")

	fmt.Printf("===> TCG AUTO ComID SELF-TEST\n")
	comID := TestComID(d)
	fmt.Printf("\n")

	fmt.Printf("===> TCG FEATURE DISCOVERY\n")
	d0, err := tcg.Discovery0(d)
	if err != nil {
		log.Fatalf("tcg.Discovery0: %v", err)
	}
	spew.Dump(d0)
	fmt.Printf("\n")

	fmt.Printf("===> TCG ADMIN SP SESSION\n")

	cs := TestControlSession(d, d0, comID)
	if cs == nil {
		log.Printf("No control session, unable to continue")
		return
	}

	var sessions []*tcg.Session
	// Try to open as many sessions as we can
	maxSessions := 10
	if cs.TPerProperties.MaxSessions != nil {
		maxSessions += int(*cs.TPerProperties.MaxSessions)
	}
	for i := 0; i < maxSessions; i++ {
		var s *tcg.Session
		var err error
		if i == 0 || cs.TPerProperties.MaxReadSessions == nil || *cs.TPerProperties.MaxReadSessions == 0 {
			s, err = cs.NewSession(tcg.AdminSP)
		} else {
			s, err = cs.NewSession(tcg.AdminSP, tcg.WithReadOnly())
		}
		if err == tcg.ErrMethodStatusNoSessionsAvailable || err == tcg.ErrMethodStatusSPBusy {
			break
		}
		if err != nil {
			log.Printf("s.NewSession (#%d) failed: %v", i, err)
			break
		}
		sessions = append(sessions, s)
		log.Printf("Session #%d (HSN=0x%x, TSN=%0x) opened", i, s.HSN, s.TSN)
	}

	if len(sessions) == 0 {
		log.Printf("No session, unable to continue")
		return
	}
	log.Printf("Opened %d sessions", len(sessions))

	defer func() {
		log.Printf("Diagnostics done, cleaning up")
		for i, s := range sessions {
			if s == nil {
				log.Printf("Session #%d already closed", i)
				continue
			}
			if err := s.Close(); err != nil {
				log.Fatalf("Session.Close (#%d) failed: %v", i, err)
			}
			log.Printf("Session #%d closed", i)
		}
	}()

	s := sessions[0]
	_ = s

	msidPin, err := table.Admin_C_PIN_MSID_GetPIN(s)
	if err != nil {
		log.Printf("table.Admin_C_PIN_MSID_GetPIN failed: %v", err)
		msidPin = nil
	} else {
		log.Printf("MSID PIN:\n%s", hex.Dump(msidPin))
	}

	rand, err := table.ThisSP_Random(s, 8)
	if err != nil {
		log.Printf("table.ThisSP_Random failed: %v", err)
	} else {
		log.Printf("Generated random numbers: %v", rand)
	}

	tperInfo, err := table.Admin_TPerInfo(s)
	if err == nil {
		log.Printf("TPerInfo table:")
		spew.Dump(tperInfo)
	}

	llcs, err := table.Admin_SP_GetLifeCycleState(s, tcg.LockingSP)
	if err == nil {
		log.Printf("Life cycle state on Locking SP: %d", llcs)
	} else {
		llcs = -1
	}

	msidOk := false
	if msidPin != nil {
		if err := table.ThisSP_Authenticate(s, tcg.AuthoritySID, msidPin); err != nil {
			log.Printf("table.ThisSP_Authenticate (SID) failed: %v", err)
		} else {
			log.Printf("Successfully authenticated as Admin SID")
			msidOk = true
		}
		if llcs == 8 /* Manufactured-Inactive */ && os.Getenv("TCGSDIAG_ACTIVATE") != "" {
			var MethodIDActivate tcg.MethodID = [8]byte{0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x02, 0x03}
			mc := s.NewMethodCall(tcg.InvokingID(tcg.LockingSP), MethodIDActivate)
			if _, err := s.ExecuteMethod(mc); err != nil {
				log.Printf("LockingSP.Activate failed: %v", err)
			} else {
				log.Printf("Locking SP activated")
				llcs = 9
			}
		}
	}

	psidPin := os.Getenv("TCGSDIAG_PSID")
	if psidPin != "" {
		if err := table.ThisSP_Authenticate(s, tcg.AuthorityPSID, []byte(psidPin)); err != nil {
			log.Printf("table.ThisSP_Authenticate (PSID) failed: %v", err)
		} else {
			log.Printf("Successfully authenticated as PSID SID")
		}
	}

	log.Printf("Admin SP testing done")
	s.Close()
	sessions[0] = nil

	fmt.Printf("\n")
	fmt.Printf("===> TCG LOCKING SP SESSION\n")
	if !msidOk {
		log.Printf("SID is changed from MSID, will not continue")
		return
	}

	if llcs == 8 /* Manufactured-Inactive */ {
		log.Printf("Locking SP not activated")
		return
	}

	auth := [8]byte{}
	username := ""
	if cs.ProtocolLevel == tcg.ProtocolLevelEnterprise {
		s, err = cs.NewSession(tcg.EnterpriseLockingSP)
		copy(auth[:], []byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x80, 0x01}) // BandMaster0
		username = "BandMaster0"
	} else {
		s, err = cs.NewSession(tcg.LockingSP)
		if os.Getenv("TCGSDIAG_AS_USER") == "" {
			copy(auth[:], []byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x01, 0x00, 0x01}) // Admin1
			username = "Admin1"
		} else {
			copy(auth[:], []byte{0x00, 0x00, 0x00, 0x09, 0x00, 0x03, 0x00, 0x01}) // User1
			username = "User1"
		}
	}
	if err != nil {
		log.Printf("Could not open Locking SP session: %v", err)
		return
	}
	sessions[0] = s
	if err := table.ThisSP_Authenticate(s, auth, msidPin); err != nil {
		log.Printf("table.ThisSP_Authenticate (Locking SP, %s) failed: %v", username, err)
		return
	} else {
		log.Printf("Successfully authenticated as %s", username)
		msidOk = true
	}

	log.Printf("Locking SP LockingInfo:")
	spew.Dump(table.LockingInfo(s))

	log.Printf("Locking SP MBRTableInfo:")
	mbi, err := table.MBR_TableInfo(s)
	if err != nil {
		log.Printf("Failed: %v", err)
	} else {
		spew.Dump(mbi)
		mbuf := make([]byte, mbi.SuggestBufferSize(s))
		log.Printf("Reading %d first bytes of MBR", len(mbuf))
		if n, err := table.MBR_Read(s, mbuf, 0); n != len(mbuf) || err != nil {
			log.Printf("Failed: %d, %v", n, err)
		} else {
			log.Printf("MBR start:\n%s", hex.Dump(mbuf[:128]))
		}
	}

	lockList, err := table.Locking_Enumerate(s)
	if err != nil {
		log.Printf("table.Locking_Enumerate failed: %v", err)
	} else {
		log.Printf("Locking regions:")
		for _, luid := range lockList {
			lr, err := table.Locking_Get(s, luid)
			if err != nil {
				spew.Printf("Region %v: <UNKNOWN> (%v)\n", hex.EncodeToString(luid[:]), err)
			} else {
				spew.Printf("Region %v: %+v\n", hex.EncodeToString(luid[:]), lr)
			}
		}
	}
}
