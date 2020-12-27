// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// @see: ngx_rtmp_mpegts_header
var mpegtsHeader = []uint8{
	/* TS */
	0x47, 0x40, 0x00, 0x10, 0x00,

	/* PSI */
	0x00, 0xb0, 0x0d, 0x00, 0x01, 0xc1, 0x00, 0x00,

	/* PAT */
	0x00, 0x01, 0xf0, 0x01,

	/* CRC */
	0x2e, 0x70, 0x19, 0x05,

	/* stuffing 167 bytes */
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

	/* TS */
	0x47, 0x50, 0x01, 0x10, 0x00,

	/* PSI */
	0x02, 0xb0, 0x17, 0x00, 0x01, 0xc1, 0x00, 0x00,

	/* PMT */
	0xe1, 0x00,
	0xf0, 0x00,
	0x1b, 0xe1, 0x00, 0xf0, 0x00, /* h264, pid=0x100=256 */
	0x0f, 0xe1, 0x01, 0xf0, 0x00, /* aac, pid=0x101=257 */

	/*0x03, 0xe1, 0x01, 0xf0, 0x00,*/ /* mp3 */
	/* CRC */
	0x2f, 0x44, 0xb9, 0x9b, /* crc for aac */
	/*0x4e, 0x59, 0x3d, 0x1e,*/ /* crc for mp3 */

	/* stuffing 157 bytes */
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// mpegts stuff using  0xff
var mpegtsStuff [188]uint8

var buffers = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024*2))
	},
}

func init() {
	for i := 0; i < len(mpegtsStuff); i++ {
		mpegtsStuff[i] = 0xff
	}
}

// Writer flv Writer
type Writer struct {
	w       io.Writer
	videoCC int
	audioCC int
}

// NewWriter .
func NewWriter(w io.Writer) (*Writer, error) {
	writer := &Writer{
		w: w,
	}

	if err := writer.writeMpegtsHeader(); err != nil {
		return nil, err
	}

	return writer, nil
}

func (w *Writer) writeMpegtsHeader() error {
	if _, err := w.w.Write(mpegtsHeader); err != nil {
		return fmt.Errorf("write ts file header failed,resean=%v", err)
	}
	return nil
}

// WriteMpegtsFrame write mpegts frame
func (w *Writer) WriteMpegtsFrame(frame *Frame) (err error) {
	if len(frame.Payload) <= 0 {
		return
	}

	buf := buffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer buffers.Put(buf)

	buf.Write(frame.Header)
	buf.Write(frame.Payload)
	// 额外帧
	for next := frame.next; next != nil; next = next.next {
		buf.Write(next.Header)
		buf.Write(next.Payload)
	}

	avdata := buf.Bytes()

	last := len(avdata)
	pos := 0

	first := true
	pkt := [188]byte{}
	cc := &w.videoCC
	if frame.Pid == tsAudioPid {
		cc = &w.audioCC
	}

	for {
		if pos >= last {
			break
		}

		// position of pkt
		p := 0

		*cc++

		// sync_byte; //8bits
		pkt[p] = 0x47
		p++

		// pid; //13bits
		pkt[p] = byte((frame.Pid >> 8) & 0x1f)
		p++

		// payload_unit_start_indicator; //1bit
		if first {
			pkt[p-1] |= 0x40
		}

		pkt[p] = byte(frame.Pid)
		p++

		// transport_scrambling_control; //2bits
		// adaption_field_control; //2bits, 0x01: PayloadOnly
		// continuity_counter; //4bits
		pkt[p] = byte(0x10 | (*cc & 0x0f))
		p++

		if first {
			first = false
			if frame.key {
				pkt[p-1] |= 0x20 // Both Adaption and Payload

				pkt[p] = 7 //size
				p++

				pkt[p] = 0x50 // random access + PCR
				p++

				writePcr(&pkt, &p, frame.Dts)
			}

			// PES header
			// packet_start_code_prefix; //24bits, '00 00 01'
			pkt[p] = 0x00
			p++
			pkt[p] = 0x00
			p++
			pkt[p] = 0x01
			p++

			//8bits
			pkt[p] = byte(frame.StreamID)
			p++

			// pts(33bits) need 5bytes.
			var headerSize uint8 = 5
			var flags uint8 = 0x80 // pts

			// dts(33bits) need 5bytes also
			if frame.Dts != frame.Pts {
				headerSize += 5
				flags |= 0x40 // dts
			}

			// 3bytes: flag fields from PES_packet_length to PES_header_data_length
			pesSize := (last - pos) + int(headerSize) + 3
			if pesSize > 0xffff {
				// when actual packet length > 0xffff(65535),
				// which exceed the max u_int16_t packet length,
				// use 0 packet length, the next unit start indicates the end of packet.
				pesSize = 0
			}

			// PES_packet_length; //16bits
			pkt[p] = byte(pesSize >> 8)
			p++
			pkt[p] = byte(pesSize)
			p++

			// PES_scrambling_control; //2bits, '10'
			// PES_priority; //1bit
			// data_alignment_indicator; //1bit
			// copyright; //1bit
			// original_or_copy; //1bit
			pkt[p] = 0x80 /* H222 */
			p++

			// PTS_DTS_flags; //2bits
			// ESCR_flag; //1bit
			// ES_rate_flag; //1bit
			// DSM_trick_mode_flag; //1bit
			// additional_copy_info_flag; //1bit
			// PES_CRC_flag; //1bit
			// PES_extension_flag; //1bit
			pkt[p] = flags
			p++

			// PES_header_data_length; //8bits
			pkt[p] = headerSize
			p++

			// pts; // 33bits
			//  p = write_pts(p, flags >> 6, frame->pts);
			writePts(&pkt, &p, flags>>6, frame.Pts)

			// dts; // 33bits
			if frame.Dts != frame.Pts {
				writePts(&pkt, &p, 1, frame.Dts)
			}
		} // end of first

		bodySize := 188 - p
		inSize := last - pos

		if bodySize <= inSize {
			copy(pkt[p:], avdata[pos:pos+bodySize])
			pos += bodySize
		} else {
			fillStuff(&pkt, &p, bodySize, inSize)
			copy(pkt[p:], avdata[pos:pos+inSize])
			pos = last
		}

		// write ts packet
		if _, err = w.w.Write(pkt[:]); err != nil {
			return fmt.Errorf("write ts ts packet failed,resean=%v", err)
		}
	}

	return

}
func writePcr(pkt *[188]byte, pos *int, pcr int64) {

	v := pcr

	pkt[*pos] = byte(v >> 25)
	*pos++

	pkt[*pos] = byte(v >> 17)
	*pos++

	pkt[*pos] = byte(v >> 9)
	*pos++

	pkt[*pos] = byte(v >> 1)
	*pos++

	pkt[*pos] = byte(v<<7 | 0x7e)
	*pos++

	pkt[*pos] = 0
	*pos++
}

func writePts(pkt *[188]byte, pos *int, fb uint8, pts int64) {
	val := 0

	val = int(int(fb)<<4 | int(((pts>>30)&0x07)<<1) | 1)

	pkt[*pos] = byte(val)
	*pos++

	val = ((int(pts>>15) & 0x7fff) << 1) | 1
	pkt[*pos] = byte(val >> 8)
	*pos++
	pkt[*pos] = byte(val)
	*pos++

	val = ((int(pts) & 0x7fff) << 1) | 1
	pkt[*pos] = byte(val >> 8)
	*pos++
	pkt[*pos] = byte(val)
	*pos++

}

func fillStuff(pkt *[188]byte, pos *int, bodySize int, inSize int) {

	// insert the stuff bytes before PES body
	stuffSize := bodySize - inSize

	// adaption_field_control; //2bits
	if v := pkt[3] & 0x20; v != 0 {
		//  has adaptation
		// packet[4]: adaption_field_length
		// packet[5]: adaption field data
		// base: start of PES body

		base := 5 + int(pkt[4])

		len := *pos - base
		copy(pkt[base+stuffSize:], pkt[base:base+len])
		// increase the adaption field size.
		pkt[4] += byte(stuffSize)

		*pos = base + stuffSize + len

		return
	}

	// create adaption field.
	// adaption_field_control; //2bits
	pkt[3] |= 0x20
	// base: start of PES body
	base := 4
	len := *pos - base
	copy(pkt[base+stuffSize:], pkt[base:base+len])
	*pos = base + stuffSize + len

	// adaption_field_length; //8bits
	pkt[4] = byte(stuffSize - 1)
	if stuffSize >= 2 {
		// adaption field flags.
		pkt[5] = 0

		// adaption data.
		if stuffSize > 2 {
			copy(pkt[6:6+stuffSize-2], mpegtsStuff[:])
		}
	}
}
