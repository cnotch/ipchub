// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"
	"testing"
	"time"

	"github.com/cnotch/ipchub/protos/rtp"
	"github.com/stretchr/testify/assert"
)

type emptyMulticastable struct {
}

func (m emptyMulticastable) AddMember(io.Closer)     {}
func (m emptyMulticastable) ReleaseMember(io.Closer) {}
func (m emptyMulticastable) MulticastIP() string     { return "234.0.0.1" }
func (m emptyMulticastable) Port(int) int            { return 0 }
func (m emptyMulticastable) SourceIP() string        { return "234.0.0.1" }
func (m emptyMulticastable) TTL() int                { return 0 }

type emptyConsumer struct {
}

func (c emptyConsumer) Consume(pack Pack) {}
func (c emptyConsumer) Close() error      { return nil }

type panicConsumer struct {
	try int
}

func (c *panicConsumer) Consume(pack Pack) {
	c.try++
	if c.try > 3 {
		panic("panicConsumer")
	}
}
func (c *panicConsumer) Close() error { return nil }

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

func TestNewStream(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		options []Option
	}{
		{
			name:    "test01",
			path:    "/live/enter",
			options: []Option{Attr(" ok ", "ok"), Attr("name", "chj"), Multicast(emptyMulticastable{})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStream(tt.path, sdpRaw, tt.options...)
			v := got.Attr("Ok")
			assert.Equal(t, "ok", v, "Must is ok")
			assert.NotNil(t, got.Multicastable(), "Must is not nil")
		})
	}
}

func Test_Consumption_Consume(t *testing.T) {
	s := NewStream("live/test", sdpRaw)

	t.Run("Consumption_Consume", func(t *testing.T) {
		closed := false
		go func() {
			for !closed {
				s.WritePacket(&rtp.Packet{})
				<-time.After(time.Millisecond * 1)
			}
		}()
		cid := s.StartConsume(emptyConsumer{}, RTPPacket, "")
		assert.Equal(t, 1, s.consumptions.Count(), "must is 1")

		<-time.After(time.Millisecond * 1000)
		cinfo, ok := s.GetConsumption(cid)
		assert.True(t, ok, "must is true")
		assert.NotZero(t, cinfo.Flow.OutBytes, "must > 0")

		s.StopConsume(cid)
		assert.Equal(t, 0, s.consumptions.Count(), "must is 0")
		closed = true
		s.Close()
	})
}

func Test_Consumption_ConsumePanic(t *testing.T) {
	s := NewStream("live/test", sdpRaw)
	t.Run("Test_Consumption_ConsumePanic", func(t *testing.T) {
		closed := false
		go func() {
			for !closed {
				s.WritePacket(&rtp.Packet{})
				<-time.After(time.Millisecond * 1)
			}
		}()
		s.StartConsume(&panicConsumer{}, RTPPacket, "")
		assert.Equal(t, 1, s.consumptions.Count(), "must is 1")

		<-time.After(time.Millisecond * 100)
		assert.Equal(t, 0, s.consumptions.Count(), "panic autoclose,must is 0")
		closed = true
		s.Close()
	})
}

func benchDispatch(n int, b *testing.B) {
	s := NewStream("/live/a", sdpRaw)
	for i := 0; i < n; i++ {
		s.StartConsume(emptyConsumer{}, RTPPacket, "")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.WritePacket(&rtp.Packet{})
		}
	})
	s.Close()
}
func Benchmark_Stream_Dispatch1(b *testing.B) {
	benchDispatch(1, b)
}
func Benchmark_Stream_Dispatch5(b *testing.B) {
	benchDispatch(5, b)
}
func Benchmark_Stream_Dispatch10(b *testing.B) {
	benchDispatch(10, b)
}
func Benchmark_Stream_Dispatch50(b *testing.B) {
	benchDispatch(50, b)
}
func Benchmark_Stream_Dispatch100(b *testing.B) {
	benchDispatch(100, b)
}
func Benchmark_Stream_Dispatch500(b *testing.B) {
	benchDispatch(500, b)
}

func Benchmark_Stream_Dispatch1000(b *testing.B) {
	benchDispatch(1000, b)
}

func Benchmark_Stream_Dispatch10000(b *testing.B) {
	benchDispatch(10000, b)
}
