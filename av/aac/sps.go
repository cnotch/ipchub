// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import (
	"encoding/hex"
	"errors"
)

// RawSPS .
// RawSPS == flv.AudioSpecificConfig
type RawSPS struct {
	Profile            byte   // 5 bits
	SampleRate         byte   // 4 bits
	ChannelConfig      byte   // 4 bits
	FrameLengthFlag    byte   // 1 bits
	DependsOnCoreCoder byte   // 1 bits
	ExtensionFlag      byte   // 1 bits
	SyncExtensionType  uint16 // 11 bits
	Profile2           byte   // 5 bits
	SbrPresentFlag     byte   // 1 bits
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
	sps.Profile = config[0] >> 3
	// 4 bits
	sps.SampleRate = (config[0]&0x07)<<1 | (config[1] >> 7)
	// 4 bits
	sps.ChannelConfig = (config[1] >> 3) & 0x0f
	sps.FrameLengthFlag = (config[1] >> 2) & 0x01
	sps.DependsOnCoreCoder = (config[1] >> 1) & 0x01
	sps.ExtensionFlag = config[1] & 0x01

	if len(config) > 3 {
		sps.SyncExtensionType = ((uint16(config[2]) << 8) | uint16(config[3])) >> 5
		sps.Profile2 = config[3] & 0x1f
	}
	if len(config) > 4 {
		sps.SbrPresentFlag = config[4] & 0x01
	}

	return nil
}
