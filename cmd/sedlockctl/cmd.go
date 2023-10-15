// Copyright (c) 2022 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/matfax/go-tcg-storage/pkg/core/table"
	"github.com/matfax/go-tcg-storage/pkg/locking"
)

type context struct {
	session *locking.LockingSP
}

type listCmd struct{}

type lockAllCmd struct{}

type unlockAllCmd struct{}

type mbrDoneCmd struct {
	Stat bool `required:"" help:"Status to set the MBRDone"`
}

type readMBRCmd struct {
	ReadMbrSize int `flag:"" default:"0"`
}

var cli struct {
	Device     string       `required:"" short:"d" type:"existingfile" help:"Path to SED device (e.g. /dev/nvme0)"`
	Sidpin     string       `optional:""`
	Sidpinmsid bool         `optional:""`
	Sidhash    string       `optional:""`
	User       string       `optional:"" short:"u"`
	Password   string       `optional:"" short:"p" help:"SID Password"`
	Hash       string       `optional:"" short:"h" default:"sedutil-dta" enum:"sedutil-dta,sedutil-sha512" help:"Either use sedutil-dta (sha1) or sedutil-sha512 for hashing"`
	List       listCmd      `cmd:"" help:"List all ranges (default)"`
	LockAll    lockAllCmd   `cmd:"" help:"Locks all ranges completely"`
	UnlockAll  unlockAllCmd `cmd:"" help:"Unlocks all ranges completely"`
	Mbrdone    mbrDoneCmd   `cmd:"" help:"Sets the MBRDone property (hide/show Shadow MBR)"`
	ReadMbr    readMBRCmd   `cmd:"" help:"Prints the binary data in the MBR area"`
}

func (l listCmd) Run(ctx *context) error {
	if len(ctx.session.Ranges) == 0 {
		return fmt.Errorf("no available locking ranges as this user")
	}
	for i, r := range ctx.session.Ranges {
		strr := "whole disk"
		if r.End > 0 {
			strr = fmt.Sprintf("%d to %d", r.Start, r.End)
		}
		if !r.WriteLockEnabled && !r.ReadLockEnabled {
			strr = "disabled"
		} else {
			if r.WriteLocked {
				strr += " [write locked]"
			}
			if r.ReadLocked {
				strr += " [read locked]"
			}
		}
		if r == ctx.session.GlobalRange {
			strr += " [global]"
		}
		if r.Name != nil {
			strr += fmt.Sprintf(" [name=%q]", *r.Name)
		}
		fmt.Printf("Range %3d: %s\n", i, strr)
	}
	return nil
}

func (u unlockAllCmd) Run(ctx *context) error {
	for i, r := range ctx.session.Ranges {
		if err := r.UnlockRead(); err != nil {
			return fmt.Errorf("read unlock range %d failed: %v", i, err)
		}
		if err := r.UnlockWrite(); err != nil {
			return fmt.Errorf("write unlock range %d failed: %v", i, err)
		}
	}
	return nil
}

func (l lockAllCmd) Run(ctx *context) error {
	for i, r := range ctx.session.Ranges {
		if err := r.LockRead(); err != nil {
			return fmt.Errorf("read lock range %d failed: %v", i, err)
		}
		if err := r.LockWrite(); err != nil {
			return fmt.Errorf("write lock range %d failed: %v", i, err)
		}
	}
	return nil
}

func (m mbrDoneCmd) Run(ctx *context) error {
	if err := ctx.session.SetMBRDone(m.Stat); err != nil {
		return fmt.Errorf("SetMBRDone failed: %v", err)
	}
	return nil
}

func (r readMBRCmd) Run(ctx *context) error {
	mbi, err := table.MBR_TableInfo(ctx.session.Session)
	if err != nil {
		return fmt.Errorf("table.MBR_TableInfo failed: %v", err)
	}
	mbuf := make([]byte, mbi.SuggestBufferSize(ctx.session.Session))
	sz := mbi.Size
	if r.ReadMbrSize > 0 && uint32(r.ReadMbrSize) < sz {
		sz = uint32(r.ReadMbrSize)
	}
	pos := uint32(0)
	chk := uint32(len(mbuf))
	for i := sz; i != 0; i -= chk {
		if n, err := table.MBR_Read(ctx.session.Session, mbuf, pos); n != len(mbuf) || err != nil {
			return fmt.Errorf("table.MBR_Read failed: %v (read: %d)", err, n)
		}
		pos += chk
		if i < chk {
			if _, err := os.Stdout.Write(mbuf[:i]); err != nil {
				return err
			}
			break
		} else {
			if _, err := os.Stdout.Write(mbuf); err != nil {
				return err
			}
		}
	}
	return nil
}
