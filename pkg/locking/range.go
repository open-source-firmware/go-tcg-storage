// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Functions and structures for dealing with lock ranges

package locking

import (
	"bytes"
	"fmt"

	"github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/core/table"
)

var (
	GlobalRangeRowUID table.RowUID = [8]byte{0x00, 0x00, 0x08, 0x02, 0x00, 0x00, 0x00, 0x01}
)

type LockRange int

var (
	LockRangeUnspecified LockRange = -1
)

type Range struct {
	l        *LockingSP
	isGlobal bool

	UID table.RowUID
	// All known authoritiers that have access to lock/unlock on this range
	// Only populated with other users if authenticated as an Admin
	// For enterprise this will always be just one user, the band-dedicated BandMasterN for RangeN
	Users map[string]core.AuthorityObjectUID

	Start LockRange
	End   LockRange

	ReadLockEnabled  bool
	WriteLockEnabled bool

	ReadLocked  bool
	WriteLocked bool

	//LockOnReset SomeType TODO
}

func fillRanges(s *core.Session, l *LockingSP) error {
	lockList, err := table.Locking_Enumerate(s)
	if err != nil {
		return fmt.Errorf("enumerate ranges failed: %v", err)
	}

	for _, luid := range lockList {
		lr, err := table.Locking_Get(s, luid)
		if err != nil {
			continue
		}
		r := &Range{}
		copy(r.UID[:], lr.UID[:])
		if bytes.Equal(r.UID[:], GlobalRangeRowUID[:]) {
			l.GlobalRange = r
			r.isGlobal = true
		}
		if lr.RangeStart != nil && lr.RangeLength != nil {
			r.Start = LockRange(*lr.RangeStart)
			r.End = r.Start + LockRange(*lr.RangeLength)
		}
		if lr.ReadLockEnabled != nil && lr.WriteLockEnabled != nil {
			r.ReadLockEnabled = *lr.ReadLockEnabled
			r.WriteLockEnabled = *lr.WriteLockEnabled
		}
		if lr.ReadLocked != nil && lr.WriteLocked != nil {
			r.ReadLocked = *lr.ReadLocked
			r.WriteLocked = *lr.WriteLocked
		}
		// TODO: Users
		// TODO: LockOnReset
		l.Ranges = append(l.Ranges, r)
	}
	return nil
}
