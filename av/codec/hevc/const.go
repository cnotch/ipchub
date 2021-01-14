// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hevc

/**
 * Table 7-1 – NAL unit type codes and NAL unit type classes in
 * T-REC-H.265-201802
 */
const (
	NalTrailN    = 0
	NalTrailR    = 1
	NalTsaN      = 2
	NalTsaR      = 3
	NalStsaN     = 4
	NalStsaR     = 5
	NalRadlN     = 6
	NalRadlR     = 7
	NalRaslN     = 8
	NalRaslR     = 9
	NalVclN10    = 10
	NalVclR11    = 11
	NalVclN12    = 12
	NalVclR13    = 13
	NalVclN14    = 14
	NalVclR15    = 15
	NalBlaWLp    = 16
	NalBlaWRadl  = 17
	NalBlaNLp    = 18
	NalIdrWRadl  = 19
	NalIdrNLp    = 20
	NalCraNut    = 21
	NalIrapVcl22 = 22
	NalIrapVcl23 = 23
	NalRsvVcl24  = 24
	NalRsvVcl25  = 25
	NalRsvVcl26  = 26
	NalRsvVcl27  = 27
	NalRsvVcl28  = 28
	NalRsvVcl29  = 29
	NalRsvVcl30  = 30
	NalRsvVcl31  = 31
	NalVps       = 32
	NalSps       = 33
	NalPps       = 34
	NalAud       = 35
	NalEosNut    = 36
	NalEobNut    = 37
	NalFdNut     = 38
	NalSeiPrefix = 39
	NalSeiSuffix = 40
	NalRsvNvcl41 = 41
	NalRsvNvcl42 = 42
	NalRsvNvcl43 = 43
	NalRsvNvcl44 = 44
	NalRsvNvcl45 = 45
	NalRsvNvcl46 = 46
	NalRsvNvcl47 = 47
	NalUnspec48  = 48
	NalUnspec49  = 49
	NalUnspec50  = 50
	NalUnspec51  = 51
	NalUnspec52  = 52
	NalUnspec53  = 53
	NalUnspec54  = 54
	NalUnspec55  = 55
	NalUnspec56  = 56
	NalUnspec57  = 57
	NalUnspec58  = 58
	NalUnspec59  = 59
	NalUnspec60  = 60
	NalUnspec61  = 61
	NalUnspec62  = 62
	NalUnspec63  = 63

	// RTP 中扩展
	NalStapInRtp = 48
	NalFuInRtp = 49
)

// HEVC(h265) 的图像片类型
const (
	SliceB = 0
	SliceP = 1
	SliceI = 2
)

/** ffmpeg 中帧统计代码
static int hevc_probe(char* pbuf, int buf_size)
{
    unsigned int code = -1;
    int vps = 0, sps = 0, pps = 0, irap = 0;
    int i;

    for (i = 0; i < buf_size - 1; i++) {
        code = (code << 8) + pbuf[i];
        if ((code & 0xffffff00) == 0x100) {
            char nal2 = pbuf[i + 1];
            int type = (code & 0x7E) >> 1;

            if (code & 0x81) // forbidden and reserved zero bits
                return 0;

            if (nal2 & 0xf8) // reserved zero
                return 0;

            switch (type) {
                case NalVPS:        vps++;  break;
                case NalSPS:        sps++;  break;
                case NalPPS:        pps++;  break;
                case NalBLA_N_LP:
                case NalBLA_W_LP:
                case NalBLA_W_RADL:
                case NalCRA_NUT:
                case NalIDR_N_LP:
                case NalIDR_W_RADL: irap++; break;
            }
        }
    }

    return 0;
}
**/

const (
	// 7.4.3.1: vps_max_layers_minus1 is in [0, 62].
	HEVC_MAX_LAYERS = 63
	// 7.4.3.1: vps_max_sub_layers_minus1 is in [0, 6].
	HEVC_MAX_SUB_LAYERS = 7
	// 7.4.3.1: vps_num_layer_sets_minus1 is in [0, 1023].
	HEVC_MAX_LAYER_SETS = 1024

	// 7.4.2.1: vps_video_parameter_set_id is u(4).
	HEVC_MAX_VPS_COUNT = 16
	// 7.4.3.2.1: sps_seq_parameter_set_id is in [0, 15].
	HEVC_MAX_SPS_COUNT = 16
	// 7.4.3.3.1: pps_pic_parameter_set_id is in [0, 63].
	HEVC_MAX_PPS_COUNT = 64

	// A.4.2: MaxDpbSize is bounded above by 16.
	HEVC_MAX_DPB_SIZE = 16
	// 7.4.3.1: vps_max_dec_pic_buffering_minus1[i] is in [0, MaxDpbSize - 1].
	HEVC_MAX_REFS = HEVC_MAX_DPB_SIZE

	// 7.4.3.2.1: num_short_term_ref_pic_sets is in [0, 64].
	HEVC_MAX_SHORT_TERM_REF_PIC_SETS = 64
	// 7.4.3.2.1: num_long_term_ref_pics_sps is in [0, 32].
	HEVC_MAX_LONG_TERM_REF_PICS = 32

	// A.3: all profiles require that CtbLog2SizeY is in [4, 6].
	HEVC_MIN_LOG2_CTB_SIZE = 4
	HEVC_MAX_LOG2_CTB_SIZE = 6

	// E.3.2: cpb_cnt_minus1[i] is in [0, 31].
	HEVC_MAX_CPB_CNT = 32

	// A.4.1: in table A.6 the highest level allows a MaxLumaPs of 35 651 584.
	HEVC_MAX_LUMA_PS = 35651584
	// A.4.1: pic_width_in_luma_samples and pic_height_in_luma_samples are
	// constrained to be not greater than sqrt(MaxLumaPs * 8).  Hence height/
	// width are bounded above by sqrt(8 * 35651584) = 16888.2 samples.
	HEVC_MAX_WIDTH  = 16888
	HEVC_MAX_HEIGHT = 16888

	// A.4.1: table A.6 allows at most 22 tile rows for any level.
	HEVC_MAX_TILE_ROWS = 22
	// A.4.1: table A.6 allows at most 20 tile columns for any level.
	HEVC_MAX_TILE_COLUMNS = 20

	// A.4.2: table A.6 allows at most 600 slice segments for any level.
	HEVC_MAX_SLICE_SEGMENTS = 600

	// 7.4.7.1: in the worst case (tiles_enabled_flag and
	// entropy_coding_sync_enabled_flag are both set), entry points can be
	// placed at the beginning of every Ctb row in every tile, giving an
	// upper bound of (num_tile_columns_minus1 + 1) * PicHeightInCtbsY - 1.
	// Only a stream with very high resolution and perverse parameters could
	// get near that, though, so set a lower limit here with the maximum
	// possible value for 4K video (at most 135 16x16 Ctb rows).
	HEVC_MAX_ENTRY_POINT_OFFSETS = HEVC_MAX_TILE_COLUMNS * 135
)
