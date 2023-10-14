// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"errors"
	"fmt"

	"github.com/matfax/go-tcg-storage/pkg/core"
	"github.com/matfax/go-tcg-storage/pkg/core/method"
	"github.com/matfax/go-tcg-storage/pkg/core/stream"
	"github.com/matfax/go-tcg-storage/pkg/core/uid"
)

var (
	ErrAuthenticationFailed = errors.New("authentication failed")
)

func ThisSP_Random(s *core.Session, count uint) ([]byte, error) {
	mc := method.NewMethodCall(uid.InvokeIDThisSP, uid.OpalRandom, s.MethodFlags)
	mc.UInt(count)
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	res, ok := resp[0].(stream.List)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
	}
	rnd, ok := res[0].([]byte)
	if !ok {
		return nil, method.ErrMalformedMethodResponse
	}
	return rnd, nil
}

func ThisSP_Authenticate(s *core.Session, authority uid.AuthorityObjectUID, proof []byte) error {
	authUID := uid.MethodID{}
	if s.ProtocolLevel == core.ProtocolLevelEnterprise {
		copy(authUID[:], uid.OpalEnterpriseAuthenticate[:])
	} else {
		copy(authUID[:], uid.OpalAuthenticate[:])
	}
	mc := method.NewMethodCall(uid.InvokeIDThisSP, authUID, s.MethodFlags)
	mc.Bytes(authority[:])
	mc.StartOptionalParameter(0, "Challenge")
	mc.Bytes(proof)
	mc.EndOptionalParameter()
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return err
	}
	res, ok := resp[0].(stream.List)
	if !ok {
		return method.ErrMalformedMethodResponse
	}
	success, okUint := res[0].(uint)
	_, okByte := res[0].([]byte)
	if okByte {
		return fmt.Errorf("got a challenge back, not implemented")
	}
	if !okUint {
		return method.ErrMalformedMethodResponse
	}
	if success == 0 {
		return ErrAuthenticationFailed
	}
	return nil
}
