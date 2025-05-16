// Copyright (c) 2022 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/alecthomas/kong"
	"github.com/open-source-firmware/go-tcg-storage/pkg/cmdutil"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/hash"
	"github.com/open-source-firmware/go-tcg-storage/pkg/locking"
)

var (
	programName = "sedlockctl"
	programDesc = "Go SEDlock control (temporary name)"
)

func main() {
	// Parse kong flags and sub-commands
	ctx := kong.Parse(&cli,
		kong.Name(programName),
		kong.Description(programDesc),
		kong.UsageOnError(),
		kong.Resolvers(cmdutil.ResolvePassword()),
		kong.NamedMapper("accessiblefile", cmdutil.AccessibleFileMapper()),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	// Set up connection and initialize session to device.
	coreObj, err := core.NewCore(cli.Device.Device)
	if err != nil {
		log.Fatalf("drive.Open: %v", err)
	}
	defer func() {
		if err := coreObj.Close(); err != nil {
			log.Fatalf("drive.Close: %v", err)
		}
	}()

	snRaw, err := coreObj.SerialNumber()
	if err != nil {
		log.Fatalf("drive.SerialNumber: %v", err)
	}
	sn := string(snRaw)

	spin := []byte{}
	if cli.Sidpin != "" {
		switch cli.Sidhash {
		case "sedutil-dta", "sha1", "dta":
			spin = hash.HashSedutilDTA(cli.Sidpin, sn)
		case "sedutil-sha512", "sha512":
			spin = hash.HashSedutil512(cli.Sidpin, sn)
		default:
			log.Fatalf("Unknown hash method %q", cli.Sidhash)
		}
	}

	initOps := []locking.InitializeOpt{}
	if len(spin) > 0 {
		initOps = append(initOps, locking.WithAuth(locking.DefaultAdminAuthority(spin)))
	}
	if cli.Sidpinmsid {
		initOps = append(initOps, locking.WithAuth(locking.DefaultAuthorityWithMSID))
	}

	cs, lmeta, err := locking.Initialize(coreObj, initOps...)
	if err != nil {
		log.Fatalf("locking.Initalize: %v", err)
	}
	defer func() {
		if err := cs.Close(); err != nil {
			log.Fatalf("locking.Close: %v", err)
		}
	}()

	var auth locking.LockingSPAuthenticator

	var pin []byte
	if cli.Password != "" {
		switch cli.Hash {
		case "sedutil-dta", "sha1", "dta":
			pin = hash.HashSedutilDTA(cli.Password, sn)
		case "sedutil-sha512", "sha512":
			pin = hash.HashSedutil512(cli.Password, sn)
		default:
			log.Fatalf("Unknown hash method %q", cli.Hash)
		}
	}
	if cli.User != "" {
		var ok bool
		auth, ok = locking.AuthorityFromName(cli.User, pin)
		if !ok {
			log.Fatalf("Authority %q is not known for this device", cli.User)
		}
	} else {
		if len(pin) == 0 {
			auth = locking.DefaultAuthorityWithMSID
		} else {
			auth = locking.DefaultAuthority(pin)
		}
	}

	l, err := locking.NewSession(cs, lmeta, auth)
	if err != nil {
		log.Fatalf("locking.NewSession: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Fatalf("locking.Close: %v", err)
		}
	}()

	// Run the command
	err = ctx.Run(&context{session: l})
	ctx.FatalIfErrorf(err)
}
