// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import (
	"encoding/hex"
	"errors"
)

// RawSPS .
type RawSPS struct {
	Profile
	SampleRate
	ChannelConfig
}

// DecodeString 从 hex 字串解码 sps
func (sps *RawSPS) DecodeString(config string) error {
	data, err := hex.DecodeString(config)
	if err != nil {
		return err
	}
	return sps.Decode(data)
}

// Decode 从字节序列中解码 sps
func (sps *RawSPS) Decode(config []byte) error {
	if len(config) < 2 {
		return errors.New("config miss data")
	}
	// 5 bits
	sps.Profile = Profile(config[0] >> 3)
	// 4 bits
	sps.SampleRate = SampleRate((config[0]&0x07)<<1 | (config[1] >> 7))
	// 4 bits
	sps.ChannelConfig = ChannelConfig((config[1] >> 3) & 0x0f)
	return nil
}
