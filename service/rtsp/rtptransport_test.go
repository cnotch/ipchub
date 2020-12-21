// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRTPTransport_parseTransport(t *testing.T) {
	tests := []struct {
		name    string
		rtpType int
		ts      string
		wantErr bool
	}{
		{
			"test1",
			int(ChannelVideo),
			"RTP/AVP;multicast;client_port=18888-18889",
			false,
		},
		{
			"test2",
			int(ChannelAudio),
			"RTP/AVP;multicast;destination=232.248.88.236;source=192.168.1.154;port=16666-0;ttl=255",
			false,
		},
	}
	var ts RTPTransport
	for _, tt := range tests {
		if err := ts.ParseTransport(tt.rtpType, tt.ts); (err != nil) != tt.wantErr {
			t.Errorf("RTPTransport.parseTransport() error = %v, wantErr %v", err, tt.wantErr)
		}
	}

	assert.Equal(t, 18888, ts.ClientPorts[int(ChannelVideo)])
	assert.Equal(t, 18889, ts.ClientPorts[int(ChannelVideo)+1])
	assert.Equal(t, PlaySession, ts.Mode, "play")
	assert.Equal(t, RTPMulticast, ts.Type, "multicast")
	assert.Equal(t, "232.248.88.236", ts.MulticastIP)
	assert.Equal(t, "192.168.1.154", ts.Source)
	assert.Equal(t, 255, ts.TTL)
	assert.Equal(t, 16666, ts.Ports[int(ChannelAudio)])
	assert.Equal(t, 0, ts.Ports[int(ChannelAudio)+1])

}

func TestRTPTransport_parseTransport_error(t *testing.T) {
	tests := []struct {
		name    string
		rtpType int
		ts      string
		wantErr bool
	}{
		{
			"error",
			int(ChannelVideo),
			"RTP/AVP/TCP;multicast;client_port=18888-18889",
			true,
		},
	}
	var ts RTPTransport
	for _, tt := range tests {
		if err := ts.ParseTransport(tt.rtpType, tt.ts); (err != nil) != tt.wantErr {
			t.Errorf("RTPTransport.parseTransport() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func Benchmark_ParseTransport(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var ts RTPTransport
			ts.ParseTransport(int(ChannelVideo), "RTP/AVP;multicast;destination=232.248.88.236;source=192.168.1.154;port=16666-0;ttl=255")
		}
	})
}
