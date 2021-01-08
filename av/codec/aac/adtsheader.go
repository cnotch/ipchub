// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

// ADTSHeader adts header include fixed and varlable header
type ADTSHeader [7]byte

func NewADTSHeader(profile, sampleRateIdx, channelConfig byte, payloadSize int) ADTSHeader {
	// AAC-ADTS
	// 6.2 Audio Data Transport Stream, ADTS
	// in aac-iso-13818-7.pdf, page 26.
	// fixed 7bytes header
	adtsHeader := ADTSHeader{0xff, 0xf1, 0x00, 0x00, 0x00, 0x0f, 0xfc}
	// the frame length is the AAC raw data plus the adts header size.
	frameLen := payloadSize + 7

	// adts_fixed_header
	// 2B, 16bits
	// int16_t syncword; //12bits, '1111 1111 1111'
	// int8_t ID; //1bit, '0'
	// int8_t layer; //2bits, '00'
	// int8_t protection_absent; //1bit, can be '1'

	// 12bits
	// int8_t profile; //2bit, 7.1 Profiles, page 40
	// TSAacSampleFrequency sampling_frequency_index; //4bits, Table 35, page 46
	// int8_t private_bit; //1bit, can be '0'
	// int8_t channel_configuration; //3bits, Table 8
	// int8_t original_or_copy; //1bit, can be '0'
	// int8_t home; //1bit, can be '0'

	// adts_variable_header
	// 28bits
	// int8_t copyright_identification_bit; //1bit, can be '0'
	// int8_t copyright_identification_start; //1bit, can be '0'
	// int16_t frame_length; //13bits
	// int16_t adts_buffer_fullness; //11bits, 7FF signals that the bitstream is a variable rate bitstream.
	// int8_t number_of_raw_data_blocks_in_frame; //2bits, 0 indicating 1 raw_data_block()

	// profile, 2bits
	adtsHeader[2] = (profile << 6) & 0xc0
	// sampling_frequency_index 4bits
	adtsHeader[2] |= (sampleRateIdx << 2) & 0x3c
	// channel_configuration 3bits
	adtsHeader[2] |= (channelConfig >> 2) & 0x01
	adtsHeader[3] = (channelConfig << 6) & 0xc0
	// frame_length 13bits
	adtsHeader[3] |= uint8((frameLen >> 11) & 0x03)
	adtsHeader[4] = uint8((frameLen >> 3) & 0xff)
	adtsHeader[5] = uint8((frameLen << 5) & 0xe0)
	// adts_buffer_fullness; //11bits
	adtsHeader[5] |= 0x1f

	return adtsHeader
}

func (h ADTSHeader) Profile() uint8 {
	return (h[2] >> 6)
}

func (h ADTSHeader) SamplingIndex() uint8 {
	return h[2] >> 2 & 0xf
}

func (h ADTSHeader) SampleRate() int {
	return SampleRates[int(h.SamplingIndex())]
}

func (h ADTSHeader) ChannelConfig() uint8 {
	return (h[2]&0x1)<<2 | h[3]>>6
}

func (h ADTSHeader) Channels() uint8 {
	return aacAudioChannels[int(h.ChannelConfig())]
}

func (h ADTSHeader) FrameLength() int {
	return int((uint32(h[3]&0x3) << 11) |
		(uint32(h[4]) << 3) |
		uint32((h[5]>>5)&0x7))
}

func (h ADTSHeader) PayloadSize() int {
	return h.FrameLength() - len(h)
}

func (h ADTSHeader) ToAsc() []byte {
	return Encode2BytesASC(h.Profile() + 1,h.SamplingIndex(),h.ChannelConfig())
}
