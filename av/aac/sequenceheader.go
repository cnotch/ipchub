// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import "errors"

// SequenceHeader .
type SequenceHeader struct {
	Profile
	SampleRate
	ChannelConfig
}

// Parse .
func (sh *SequenceHeader) Parse(config []byte) error {
	if len(config) < 2 {
		return errors.New("config miss data")
	}
	// 5 bits
	sh.Profile = Profile(config[0] >> 3)
	// 4 bits
	sh.SampleRate = SampleRate((config[0]&0x07)<<1 | (config[1] >> 7))
	// 4 bits
	sh.ChannelConfig = ChannelConfig((config[1] >> 3) & 0x0f)
	return nil
}
