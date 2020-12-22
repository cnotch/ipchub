// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"encoding/binary"
	"time"
)

const jan1970 = 0x83aa7e80

// SyncClock .
type SyncClock struct {
	// NTP Timestamp（Network time protocol）SR包发送时的绝对时间值。
	// NTP的作用是同步不同的RTP媒体流。
	// NTP时间戳，它的前32位是从1900 年1 月1 日0 时开始到现在的以秒为单位的整数部，
	// 后32 位是此时间的小数部，因此，它可以肯定的表示了数据发送出去的绝对时间。
	NTPTime int64 // 此处转换成自 January 1, year 1 以来的纳秒数
	// RTP Timestamp：与NTP时间戳对应，
	// 与RTP数据包中的RTP时间戳具有相同的单位和随机初始值。
	RTPTime uint32
}

// LocalTime 本地时间
func (sc *SyncClock) LocalTime() time.Time {
	return time.Unix(0, sc.NTPTime).In(time.Local)
}

// Decode .
func (sc *SyncClock) Decode(data []byte) (ok bool) {
	if data[1] == 200 {
		msw := binary.BigEndian.Uint32(data[8:])
		lsw := binary.BigEndian.Uint32(data[12:])
		sc.RTPTime = binary.BigEndian.Uint32(data[16:])
		sc.NTPTime = int64(msw-jan1970)*int64(time.Second) + (int64(lsw)*1000_000_000)>>32
		ok = true
	}
	return
}

// Rtp2Ntp .
func (sc *SyncClock) Rtp2Ntp(rtptime uint32, unit int) int64 {
	diff := int64(rtptime) - int64(sc.RTPTime)
	duration := int64(float64(diff) / float64(unit) * float64(time.Second))
	return (sc.NTPTime + duration) / int64(time.Millisecond)
}
