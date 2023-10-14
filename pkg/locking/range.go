// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Functions and structures for dealing with lock ranges

package locking

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/matfax/go-tcg-storage/pkg/core"
	"github.com/matfax/go-tcg-storage/pkg/core/table"
	"github.com/matfax/go-tcg-storage/pkg/core/uid"
)

type LockRange int

var (
	LockRangeUnspecified LockRange = -1
)

type Range struct {
	l        *LockingSP
	isGlobal bool

	UID  uid.RowUID
	Name *string
	// All known authoritiers that have access to lock/unlock on this range
	// Only populated with other users if authenticated as an Admin
	// For enterprise this will always be just one user, the band-dedicated BandMasterN for RangeN
	Users map[string]uid.AuthorityObjectUID

	Start LockRange
	End   LockRange

	ReadLockEnabled  bool
	WriteLockEnabled bool

	ReadLocked  bool
	WriteLocked bool

	//LockOnReset SomeType TODO: Create this type from spec
}

func fillRanges(s *core.Session, l *LockingSP) error {
	lockList, err := table.Locking_Enumerate(s)
	if err != nil {
		return fmt.Errorf("enumerate ranges failed: %v", err)
	}

	sort.Slice(lockList, func(i, j int) bool {
		return bytes.Compare(lockList[i][:], lockList[j][:]) < 0
	})

	for _, luid := range lockList {
		lr, err := table.Locking_Get(s, luid)
		if err != nil {
			continue
		}
		r := &Range{
			l: l,
		}
		copy(r.UID[:], lr.UID[:])
		if bytes.Equal(r.UID[:], uid.GlobalRangeRowUID[:]) {
			l.GlobalRange = r
			r.isGlobal = true
		}
		if lr.Name != nil && len(*lr.Name) > 0 {
			r.Name = lr.Name
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
		// TODO: Enumerate users with permissions on this range
		// TODO: Fill the LockOnReset property
		l.Ranges = append(l.Ranges, r)
	}
	return nil
}

func (r *Range) UnlockRead() error {
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	v := false
	lr.ReadLocked = &v
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.ReadLocked = v
	return nil
}

func (r *Range) LockRead() error {
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	v := true
	lr.ReadLocked = &v
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.ReadLocked = v
	return nil
}

func (r *Range) UnlockWrite() error {
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	v := false
	lr.WriteLocked = &v
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.WriteLocked = v
	return nil
}

func (r *Range) LockWrite() error {
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	v := true
	lr.WriteLocked = &v
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.WriteLocked = v
	return nil
}

func (r *Range) SetReadLockEnabled(v bool) error {
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	lr.ReadLockEnabled = &v
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.ReadLockEnabled = v
	return nil

}

func (r *Range) SetWriteLockEnabled(v bool) error {
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	lr.WriteLockEnabled = &v
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.WriteLockEnabled = v
	return nil

}

func (r *Range) SetRange(from LockRange, to LockRange) error {
	if r.isGlobal {
		return fmt.Errorf("cannot modify the global range")
	}
	lr := &table.LockingRow{}
	copy(lr.UID[:], r.UID[:])
	from64 := uint64(from)
	lr.RangeStart = &from64
	to64 := uint64(to)
	lr.RangeLength = &to64
	if err := table.Locking_Set(r.l.Session, lr); err != nil {
		return err
	}
	r.Start = from
	r.End = to
	return nil
}

func (r *Range) Erase() error {
	return fmt.Errorf("not implemented")
}
