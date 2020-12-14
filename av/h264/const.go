// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package h264

/*
 * Table 7-1 – NAL unit type codes, syntax element categories, and NAL unit type classes in
 * T-REC-H.264-201704
 */
// H264 NAL 单元类型
const (
	NalUnspecified     = 0
	NalSlice           = 1  // 不分区非IDR图像的片
	NalDpa             = 2  // 片分区A
	NalDpb             = 3  // 片分区B
	NalDpc             = 4  // 片分区C
	NalIdrSlice        = 5  // IDR图像中的片（I帧）
	NalSei             = 6  // 补充增强信息单元，sei playload可以使用户自定义数据， 那么我们就可以利用它来传输数据
	NalSps             = 7  // 序列参数集
	NalPps             = 8  // 图像参数集
	NalAud             = 9  // 分界符
	NalEndSequence     = 10 // 序列结束
	NalEndStream       = 11 // 码流结束
	NalFillerData      = 12 // 填充
	NalSpsExt          = 13 //
	NalPrefix          = 14
	NalSubSps          = 15
	NalDps             = 16
	NalReserved17      = 17
	NalReserved18      = 18
	NalAuxiliarySlice  = 19
	NalExtenSlice      = 20
	NalDepthExtenSlice = 21
	NalReserved22      = 22
	NalReserved23      = 23
	NalUnspecified24   = 24
	NalUnspecified25   = 25
	NalUnspecified26   = 26
	NalUnspecified27   = 27
	NalUnspecified28   = 28
	NalUnspecified29   = 29
	NalUnspecified30   = 30
	NalUnspecified31   = 31

	// NAL 在 RTP 包中的扩展
	NalStapaInRtp  = 24 // 单一时间的组合包
	NalStapbInRtp  = 25 // 单一时间的组合包
	NalMtap16InRtp = 26 // 多个时间的组合包
	NalMtap24InRtp = 27 // 多个时间的组合包
	NalFuAInRtp    = 28 // 分片的单元
	NalFuBInRtp    = 29 // 分片的单元

	NalTypeBitmask = 0x1F
)

// 其他常量
const (
	// 7.4.2.1.1: seq_parameter_set_id is in [0, 31].
	MaxSpsCount = 32
	// 7.4.2.2: pic_parameter_set_id is in [0, 255].
	MaxPpsCount = 256

	// A.3: MaxDpbFrames is bounded above by 16.
	MaxDpbFrames = 16
	// 7.4.2.1.1: max_num_ref_frames is in [0, MaxDpbFrames], and
	// each reference frame can have two fields.
	MaxRefs = 2 * MaxDpbFrames

	// 7.4.3.1: modification_of_pic_nums_idc is not equal to 3 at most
	// num_ref_idx_lN_active_minus1 + 1 times (that is, once for each
	// possible reference), then equal to 3 once.
	MaxRplmCount = MaxRefs + 1

	// 7.4.3.3: in the worst case, we begin with a full short-term
	// reference picture list.  Each picture in turn is moved to the
	// long-term list (type 3) and then discarded from there (type 2).
	// Then, we set the length of the long-term list (type 4), mark
	// the current picture as long-term (type 6) and terminate the
	// process (type 0).
	MaxMmcoCount = MaxRefs*2 + 3

	// A.2.1, A.2.3: profiles supporting FMO constrain
	// num_slice_groups_minus1 to be in [0, 7].
	MaxSliceGroups = 8

	// E.2.2: cpb_cnt_minus1 is in [0, 31].
	MaxCpbCnt = 32

	// A.3: in table A-1 the highest level allows a MaxFS of 139264.
	MaxMbPicSize = 139264
	// A.3.1, A.3.2: PicWidthInMbs and PicHeightInMbs are constrained
	// to be not greater than sqrt(MaxFS * 8).  Hence height/width are
	// bounded above by sqrt(139264 * 8) = 1055.5 macroblocks.
	MaxMbWidth  = 1055
	MaxMbHeight = 1055
	MaxWidth    = MaxMbWidth * 16
	MaxHeight   = MaxMbHeight * 16
)
