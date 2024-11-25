// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"testing"

	"github.com/pixelbender/go-sdp/sdp"
)

const sdpRaw = `v=0
o=- 0 0 IN IP4 127.0.0.1
s=No Name
c=IN IP4 127.0.0.1
t=0 0
a=tool:libavformat 58.20.100
m=video 0 RTP/AVP 96
b=AS:2500
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAH6zZQFAFuhAAAAMAEAAAAwPI8YMZYA==,aO+8sA==; profile-level-id=64001F
a=control:streamid=0
m=audio 0 RTP/AVP 97
b=AS:160
a=rtpmap:97 MPEG4-GENERIC/44100/2
a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=121056E500
a=control:streamid=1
`

const sdpRaw2 = `v=0
o=- 946684871882903 1 IN IP4 192.168.1.154
s=RTSP/RTP stream from IPNC
i=h264
t=0 0
a=tool:LIVE555 Streaming Media v2008.04.02
a=type:broadcast
a=control:*
a=source-filter: incl IN IP4 * 192.168.1.154
a=rtcp-unicast: reflection
a=range:npt=0-
a=x-qt-text-nam:RTSP/RTP stream from IPNC
a=x-qt-text-inf:h264
m=audio 18888 RTP/AVP 0
c=IN IP4 232.190.161.0/255
a=control:track1
m=video 16666 RTP/AVP 96
c=IN IP4 232.248.88.236/255
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1;profile-level-id=EE3CB0;sprop-parameter-sets=H264
a=control:track2
`

const sdpRaw3 = `v=0
o=- 0 0 IN IP6 ::1
s=No Name
c=IN IP6 ::1
t=0 0
a=tool:libavformat 58.20.100
m=video 0 RTP/AVP 96
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z3oAH7y0AoAt0IAAAAMAgAAAHkeMGVA=,aO8Pyw==; profile-level-id=7A001F
a=control:streamid=0
m=audio 0 RTP/AVP 97
b=AS:128
a=rtpmap:97 MPEG4-GENERIC/44100/2
a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=121056E500`

// 4k mp4
const sdpRaw4 = `v=0
o=- 0 0 IN IP6 ::1
s=No Name
c=IN IP6 ::1
t=0 0
a=tool:libavformat 58.20.100
m=video 0 RTP/AVP 96
b=AS:31998
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAM6wspADwAQ+wFSAgICgAAB9IAAdTBO0LFok=,aOtzUlA=; profile-level-id=640033
a=control:streamid=0
m=audio 0 RTP/AVP 97
b=AS:317
a=rtpmap:97 MPEG4-GENERIC/48000/2
a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=1190
a=control:streamid=1`

const sdpH265Raw = `v=0
o=- 0 0 IN IP6 ::1
s=No Name
c=IN IP6 ::1
t=0 0
a=tool:libavformat 58.20.100
m=video 0 RTP/AVP 96
a=rtpmap:96 H265/90000
a=fmtp:96 sprop-vps=QAEMAf//BAgAAAMAnQgAAAMAAF26AkA=; sprop-sps=QgEBBAgAAAMAnQgAAAMAAF2wAoCALRZbqSTK4BAAAAMAEAAAAwHggA==; sprop-pps=RAHBcrRiQA==
a=control:streamid=0
m=audio 0 RTP/AVP 97
b=AS:128
a=rtpmap:97 MPEG4-GENERIC/44100/2
a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=121056E500
a=control:streamid=1
`
const sdpTplink500 = `v=0
o=- 14665860 31787219 1 IN IP4 192.168.1.60
s=Session streamed by "TP-LINK RTSP Server"
t=0 0
m=video 0 RTP/AVP 96
c=IN IP4 0.0.0.0
b=AS:4096
a=range:npt=0-
a=control:track1
a=rtpmap:96 H265/90000
a=fmtp:96 profile-space=0;profile-id=12;tier-flag=0;level-id=0;interop-constraints=600000000000;sprop-vps=AAAAAUABDAH//wFgAAADAAADAAADAAADAJasCQ==;sprop-sps=AAAAAUIBAQFgAAADAAADAAADAAADAJagAWggBln3ja5JMmuWMAgAAAMACAAAAwB4QA==;sprop-pps=AAAAAUQB4HawJkA=
m=audio 0 RTP/AVP 8
a=rtpmap:8 PCMA/8000
a=control:track2
m=application/TP-LINK 0 RTP/AVP smart/1/90000
a=rtpmap:95 TP-LINK/90000
a=control:track3
`

func Benchmark_ThirdSdpParse(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sdp.ParseString(sdpRaw2)
		}
	})
}

func Test_SDPParse(t *testing.T) {
	t.Run("Test_SDPParse", func(t *testing.T) {
		s1, err := sdp.ParseString(sdpRaw)
		if err != nil {
			t.Errorf("sdp.ParseString() error = %v", err)
		}
		_ = s1

		s2, err := sdp.ParseString(sdpRaw2)
		if err != nil {
			t.Errorf("sdp.ParseString() error = %v", err)
		}
		_ = s2
	})
}
