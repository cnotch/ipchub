// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

// AudioIsAAC judge audio is AAC
func AudioIsAAC(data []uint8) bool {

	if len(data) < 1 {
		return false
	}

	soundFormat := data[0]
	soundFormat = (soundFormat >> 4) & 0x0f

	return soundFormat == SoundFormatAAC
}

// AudioIsSequenceHeader judge audio is aac sequence header
func AudioIsSequenceHeader(data []uint8) bool {

	if !AudioIsAAC(data) {
		return false
	}

	if len(data) < 2 {
		return false
	}

	aacPacketType := data[1]

	return aacPacketType == AACPacketTypeSequenceHeader
}

// VideoIsH264 judge video is h264
func VideoIsH264(data []uint8) bool {

	if len(data) < 1 {
		return false
	}

	codecID := data[0]
	codecID &= 0x0f

	return CodecIDAVC == codecID
}

// VideoH264IsKeyframe judge video is h264 key frame
func VideoH264IsKeyframe(data []uint8) bool {
	// 2bytes required.
	if len(data) < 2 {
		return false
	}

	frameType := data[0]
	frameType = (frameType >> 4) & 0x0F

	return frameType == FrameTypeKeyFrame
}

// VideoH264IsSequenceHeader judge video is h264 sequence header and key frame
// payload: 0x17 0x00
func VideoH264IsSequenceHeader(data []uint8) bool {
	// sequence header only for h264
	if !VideoIsH264(data) {
		return false
	}

	// 2bytes required.
	if len(data) < 2 {
		return false
	}

	frameType := data[0]
	frameType = (frameType >> 4) & 0x0F

	avcPacketType := data[1]

	return frameType == FrameTypeKeyFrame &&
		avcPacketType == AVCPacketTypeSequenceHeader
}
