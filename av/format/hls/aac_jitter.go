// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

import "github.com/cnotch/ipchub/av/codec/aac"

const (
	// in ms, for HLS aac sync time.
	hlsConfDefaultAacSync = 100
)

// jitter correct for audio,
// the sample rate 44100/32000 will lost precise,
// when mp4/ts(tbn=90000) covert to flv/rtmp(1000),
// so the Hls on ipad or iphone will corrupt,
// @see nginx-rtmp: est_pts
type hlsAacJitter struct {
	basePts   int64
	nbSamples int64
	syncMs    int
}

func newHlsAacJitter() *hlsAacJitter {
	return &hlsAacJitter{
		syncMs: hlsConfDefaultAacSync,
	}
}

// when buffer start, calc the "correct" pts for ts,
// @param flv_pts, the flv pts calc from flv header timestamp,
// @param sample_rate, the sample rate in format(flv/RTMP packet header).
// @param aac_sample_rate, the sample rate in codec(sequence header).
// @return the calc correct pts.
func (ha *hlsAacJitter) onBufferStart(pts int64, sampleRate int) (calcCorrectPts int64) {
	// sync time set to 0, donot adjust the aac timestamp.
	if 0 == ha.syncMs {
		return pts
	}

	// @see: ngx_rtmp_hls_audio
	// drop the rtmp audio packet timestamp, re-calc it by sample rate.
	//
	// resample for the tbn of ts is 90000, flv is 1000,
	// we will lost timestamp if use audio packet timestamp,
	// so we must resample. or audio will corrupt in IOS.
	estPts := ha.basePts + ha.nbSamples*int64(90000)*int64(aac.SamplesPerFrame)/int64(sampleRate)
	dpts := estPts - pts

	if (dpts <= int64(ha.syncMs)*90) && (dpts >= int64(ha.syncMs)*int64(-90)) {
		ha.nbSamples++
		return estPts
	}

	// resync
	ha.basePts = pts
	ha.nbSamples = 1

	return pts
}

// when buffer continue, muxer donot write to file,
// the audio buffer continue grow and donot need a pts,
// for the ts audio PES packet only has one pts at the first time.
func (ha *hlsAacJitter) onBufferContinue() {
	ha.nbSamples++

	return
}
