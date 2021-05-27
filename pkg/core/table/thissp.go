// Copyright (c) 2021 by library authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements TCG Storage Core Table operations

package table

import (
	"fmt"

	"github.com/bluecmd/go-tcg-storage/pkg/core"
	"github.com/bluecmd/go-tcg-storage/pkg/core/stream"
)

func ThisSP_Random(s *core.Session, count uint) ([]byte, error) {
	mc := core.NewMethodCall(core.InvokeIDThisSP, MethodIDRandom)
	mc.UInt(count)
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return nil, err
	}
	res, ok := resp[0].(stream.List)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	rnd, ok := res[0].([]byte)
	if !ok {
		return nil, core.ErrMalformedMethodResponse
	}
	return rnd, nil
}

func ThisSP_Authenticate(s *core.Session, authority core.AuthorityObjectUID, proof []byte) error {
	mc := core.NewMethodCall(core.InvokeIDThisSP, MethodIDAuthenticate)
	mc.Bytes(authority[:])
	mc.StartOptionalParameter(0)
	mc.Bytes(proof)
	mc.EndOptionalParameter()
	resp, err := s.ExecuteMethod(mc)
	if err != nil {
		return err
	}
	res, ok := resp[0].(stream.List)
	if !ok {
		return core.ErrMalformedMethodResponse
	}
	success, okUint := res[0].(uint)
	_, okByte := res[0].([]byte)
	if okByte {
		return fmt.Errorf("got a challenge back, not implemented")
	}
	if !okUint {
		return core.ErrMalformedMethodResponse
	}
	if success == 0 {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
