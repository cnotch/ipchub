// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Translate from FFmpeg cbs_h264.h cbs_h264_syntax_template.c
//
package h264

import (
	"encoding/base64"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/ipchub/utils/bits"
)

// RawNALUnitHeader 原始 h264 Nal单元头
type RawNALUnitHeader struct {
	ForbiddenZeroBit uint8
	NalRefIdc        uint8
	NalUnitType      uint8
	// svc_extension_flag    uint8
	// avc_3d_extension_flag uint8
}

// Set .
func (h *RawNALUnitHeader) Set(nal uint8) (err error) {
	h.ForbiddenZeroBit = (nal >> 7) & 1
	h.NalRefIdc = (nal >> 5) & 3
	h.NalUnitType = nal & 31

	if h.NalUnitType == NalPrefix ||
		h.NalUnitType == NalExtenSlice ||
		h.NalUnitType == NalDepthExtenSlice {
		err = fmt.Errorf("SVC,3DAVC,MVC not supported. nal_unit_type = %d", h.NalUnitType)
	}

	return
}

// RawHRD .
type RawHRD struct {
	CpbCntMinus1 uint8
	BitRateScale uint8
	CpbSizeScale uint8

	BitRateValueMinus1 [MaxCpbCnt]uint32
	CpbSizeValueMinus1 [MaxCpbCnt]uint32
	CbrFlag            [MaxCpbCnt]uint8

	InitialCpbRemovalDelayLengthMinus1 uint8
	CpbRemovalDelayLengthMinus1        uint8
	DpbOutputDelayLengthMinus1         uint8
	TimeOffsetLength                   uint8
}

// RawVUI .
type RawVUI struct {
	// aspect_ratio_info_present_flag
	// 等于1 表示aspect_ratio_idc 存在。
	// aspect_ratio_info_present_flag 等于0 表示 aspect_ratio_idc不存在。
	// aspect_ratio_idc
	// 表示亮度样值的样点高宽比的取值。表E-1 给出代码的含义。
	// 当aspect_ratio_idc 的取值表示是Extended_SAR，样点高宽比由sar_width 和sar_height 描述。
	// 当aspect_ratio_idc 语法元素不存在，aspect_ratio_idc的值应被推定为0。
	// sar_width和sar_height应是互质的或等于0。
	// 当aspect_ratio_idc等于0或sar_width等于0或sar_height等于0时，
	// 样点高宽比应被视为本建议书 | 国际标准未定义的。
	AspectRatioInfoPresentFlag uint8
	AspectRatioIdc             uint8
	SarWidth                   uint16 // 表示样点高宽比的水平尺寸（以任意单位）。
	SarHeight                  uint16 // 表示样点高宽比的垂直尺寸（以与sar_width相同的任意单位）。

	// overscan_info_present_flag
	// 等于1 表示overscan_appropriate_flag 存在。
	// 等于0或不存在时，视频信号的优选显示方法未定义。
	// overscan_appropriate_flag
	// 等于1 表示被剪切的解码图像输出适合以过扫描显示。
	// overscan_appropriate_flag等于0 表示被剪切的解码图像输出在向外到图像剪切矩形边缘的整个区域包含重要的可视信息，
	// 因此被剪切的解码图像输出不应以过扫描显示。
	// 相反地，它应以显示区域和剪切矩形的完全匹配方式显示或以欠扫描显示。
	// 注1 — 例如，overscan_appropriate_flag等于1可以用于娱乐电视节目，或视频会议中人物的现场图像，
	// 而 overscan_appropriate_flag 等于0可以用于计算机屏幕捕捉或保安摄像内容。
	OverscanInfoPresentFlag uint8
	OverscanAppropriateFlag uint8

	// video_signal_type_present_flag
	// 等于1表示video_format，video_full_range_flag和 colour_description_present_flag 存在。
	// 等于0 表示video_format，video_full_range_flag 和 colour_description_present_flag不存在。
	// video_format
	// 表示图像在国际标准编码前的制式，见表E-2的规定。
	// 当video_format语法元素不存在，video_format的值应被推定为5。
	// video_full_range_flag
	// 表示黑电平和亮度与色度信号的范围由E’Y, E’PB, 和E’PR 或 E’R, E’G, 和 E’B模拟信号分量得到。
	// 当video_full_range_flag语法元素不存在时，video_full_range_flag的值应被推定为等于0。
	// colour_description_present_flag
	// 等于1表示colour_primaries，transfer_characteristics和 matrix_coefficients存在。
	// 等于0表示colour_primaries，transfer_characteristics和 matrix_coefficients不存在。
	// colour_primaries
	// 表示最初的原色的色度坐标，按照CIE 1931的规定（见表E-3），x和y的定义由SO/CIE10527规定。
	// 当colour_primaries 语法元素不存在时，colour_primaries 的值应被推定为等于2（色度未定义或由应用决定）。
	VideoSignalTypePresentFlag   uint8
	VideoFormat                  uint8
	VideoFullRangeFlag           uint8
	ColourDescriptionPresentFlag uint8
	ColourPrimaries              uint8
	TransferCharacteristics      uint8
	MatrixCoefficients           uint8

	ChromaLocInfoPresentFlag       uint8
	ChromaSampleLocTypeTopField    uint8
	ChromaSampleLocTypeBottomField uint8

	// 和帧率相关
	TimingInfoPresentFlag uint8
	NumUnitsInTick        uint32
	TimeScale             uint32
	FixedFrameRateFlag    uint8

	NalHrdParametersPresentFlag uint8
	NalHrdParameters            RawHRD
	VclHrdParametersPresentFlag uint8
	VclHrdParameters            RawHRD
	LowDelayHrdFlag             uint8

	PicStructPresentFlag uint8

	BitstreamRestrictionFlag           uint8
	MotionVectorsOverPicBoundariesFlag uint8
	MaxBytesPerPicDenom                uint8
	MaxBitsPerMbDenom                  uint8
	Log2MaxMvLengthHorizontal          uint8
	Log2MaxMvLengthVertical            uint8
	MaxNumReorderFrames                uint8
	MaxDecFrameBuffering               uint8
}

// RawSPS .
type RawSPS struct {
	NalUnitHeader RawNALUnitHeader

	// 指明所用  profile、level、及对附录A.2的遵循情况
	// Set0 -> A.2.1 ，依次递推
	ProfileIdc         uint8
	ConstraintSet0Flag uint8
	ConstraintSet1Flag uint8
	ConstraintSet2Flag uint8
	ConstraintSet3Flag uint8
	ConstraintSet4Flag uint8
	ConstraintSet5Flag uint8
	ReservedZero2Bits  uint8
	LevelIdc           uint8

	// 指明本序列参数集的  id 号，这个 id 号将被 picture 参数集引用，
	// 本句法元素的值应该在[0，31]。
	// 编码需要产生新的序列集时，使用新的id，而不是改变原来参数集的内容
	SeqParameterSetID uint8

	ChromaFormatIdc                 uint8
	SeparateColourPlaneFlag         uint8
	BitDepthLumaMinus8              uint8
	BitDepthChromaMinus8            uint8
	QpprimeYZeroTransformBypassFlag uint8

	SeqScalingMatrixPresentFlag uint8
	SeqScalingListPresentFlag   [12]uint8
	ScalingList4x4              [6][64]int8
	ScalingList8x8              [6][64]int8

	// 为读取另一个句法元素 frame_num 服务的，frame_num 是最重要的句法元素之一，
	// 它标识所属图像的解码顺序 。这个句法元素同时也指明了 frame_num 的所能达到的最大值：
	// MaxFrameNum = 2*exp( Log2MaxFrameNumMinus4 + 4 )
	Log2MaxFrameNumMinus4          uint8
	PicOrderCntType                uint8      // 指明了 poc  (picture  order  count)  的编码方法，poc 标识图像的播放顺序。由poc 可以由 frame-num 通过映射关系计算得来，也可以索性由编码器显式地传送。
	Log2MaxPicOrderCntLsbMinus4    uint8      // 指明了变量  MaxPicOrderCntLsb 的值: MaxPicOrderCntLsb = pow(2, (log2_max_pic_order_cnt_lsb_minus4 + 4) )
	DeltaPicOrderAlwaysZeroFlag    uint8      // 等于 1 时,句法元素 delta_pic_order_cnt[0]和 delta_pic_order_cnt[1]
	OffsetForNonRefPic             int32      // 被用来计算非参考帧或场的 POC,本句法元素的值应该在[pow(-2, 31)  , pow(2, 31)  – 1]。
	OffsetForTopToBottomField      int32      // 被用来计算帧的底场的 POC,  本句法元素的值应该在[pow(-2, 31)  , pow(2, 31)  – 1]。
	NumRefFramesInPicOrderCntCycle uint8      // 被用来解码POC, 本句法元素的值应该在[0,255]。
	OffsetForRefFrame              [256]int32 // offset_for_ref__frame[i]  用于解码 POC，本句法元素对循环num_ref_frames_in_pic_order_cycle 中的每一个元素指定一个偏移。

	// max_num_ref_frames
	// 指定参考帧队列可能达到的最大长度，解码器依照这个句法元素的值开辟存储区，
	// 这个存储区用于存放已解码的参考帧，H.264 规定最多可用 16 个参考帧，本句法元素的值最大为 16。
	// 值得注意的是这个长度以帧为单位，如果在场模式下，应该相应地扩展一倍。
	// gaps_in_frame_num_value_allowed_flag
	// 这个句法元素等于 1 时，表示允许句法元素 frame_num 可以不连续。
	// 当传输信道堵塞严重时，编码器来不及将编码后的图像全部发出，这时允许丢弃若干帧图像。
	MaxNumRefFrames           uint8
	GapsInFrameNumAllowedFlag uint8

	// pic_width_in_mbs_minus1
	// 本句法元素加 1 后指明图像宽度，以宏块为单位： PicWidthInMbs = PicWidthInMbsMinus1 + 1。
	// 通过这个句法元素解码器可以计算得到亮度分量以像素为单位的图像宽度： PicWidthInSamples = PicWidthInMbs * 16
	// pic_height_in_map_units_minus1 同理
	PicWidthInMbsMinus1       uint16
	PicHeightInMapUnitsMinus1 uint16

	// frame_mbs_only_flag
	// 本句法元素等于 0 时表示本序列中所有图像的编码模式都是帧，没有其他编码模式存在；
	// 本句法元素等于 1 时  ，表示本序列中图像的编码模式可能是帧，也可能是场或帧场自适应，某个图像具体是哪一种要由其他句法元素决定。
	// mb_adaptive_frame_field_flag
	// 指明本序列是否属于帧场自适应模式。
	// mb_adaptive_frame_field_flag等于１时表明在本序列中的图像如果不是场模式就是帧场自适应模式
	// 等于０时表示本序列中的图像如果不是场模式就是帧模式。
	// direct_8x8_inference_flag    用于指明 B 片的直接和 skip 模式下运动矢量的预测方法。
	FrameMbsOnlyFlag         uint8
	MbAdaptiveFrameFieldFlag uint8
	Direct8x8InferenceFlag   uint8

	// frame_cropping_flag
	// 用于指明解码器是否要将图像裁剪后输出，如果是的话，后面紧跟着的四个句法元素分别指出左右、上下裁剪的宽度。
	FrameCroppingFlag     uint8
	FrameCropLeftOffset   uint16
	FrameCropRightOffset  uint16
	FrameCropTopOffset    uint16
	FrameCropBottomOffset uint16

	// vui_parameters_present_flag
	// 指明 vui 子结构是否出现在码流中，vui 用以表征视频格式等额外信息。
	VuiParametersPresentFlag uint8
	Vui                      RawVUI
}

// Width 视频宽度（像素）
func (sps *RawSPS) Width() int {
	w := (sps.PicWidthInMbsMinus1+1)*16 - sps.FrameCropLeftOffset*2 - sps.FrameCropRightOffset*2
	return int(w)
}

// Height 视频高度（像素）
func (sps *RawSPS) Height() int {
	h := (2-uint16(sps.FrameMbsOnlyFlag))*(sps.PicHeightInMapUnitsMinus1+1)*16 - sps.FrameCropTopOffset*2 - sps.FrameCropBottomOffset*2
	return int(h)
}

// FrameRate Video frame rate
func (sps *RawSPS) FrameRate() float64 {
	if sps.Vui.NumUnitsInTick == 0 {
		return 0.0
	}
	return float64(sps.Vui.TimeScale) / float64(sps.Vui.NumUnitsInTick*2)
}

// IsFixedFrameRate 是否固定帧率
func (sps *RawSPS) IsFixedFrameRate() bool {
	return sps.Vui.FixedFrameRateFlag == 1
}

// DecodeString 从 base64 字串解码 sps NAL
func (sps *RawSPS) DecodeString(b64 string) error {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return sps.Decode(data)
}

// Decode 从字节序列中解码 sps NAL
func (sps *RawSPS) Decode(data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("RawSPS decode panic；r = %v \n %s", r, debug.Stack())
		}
	}()

	spsWEB := utils.RemoveH264or5EmulationBytes(data)
	if len(spsWEB) < 4 {
		return errors.New("The data is not enough")
	}

	r := bits.NewReader(spsWEB)
	if err = sps.NalUnitHeader.decode(r); err != nil {
		return
	}

	if sps.NalUnitHeader.NalUnitType != NalSps {
		return errors.New("not is sps NAL UNIT")
	}

	// 前三个字节
	sps.ProfileIdc = r.ReadUint8(8)

	sps.ConstraintSet0Flag = r.ReadBit()
	sps.ConstraintSet1Flag = r.ReadBit()
	sps.ConstraintSet2Flag = r.ReadBit()
	sps.ConstraintSet3Flag = r.ReadBit()
	sps.ConstraintSet4Flag = r.ReadBit()
	sps.ConstraintSet5Flag = r.ReadBit()
	sps.ReservedZero2Bits = r.ReadUint8(2)

	sps.LevelIdc = r.ReadUint8(8)

	// seq_parameter_set_id
	sps.SeqParameterSetID = r.ReadUe8()

	if sps.ProfileIdc == 100 || sps.ProfileIdc == 110 ||
		sps.ProfileIdc == 122 || sps.ProfileIdc == 244 ||
		sps.ProfileIdc == 44 || sps.ProfileIdc == 83 ||
		sps.ProfileIdc == 86 || sps.ProfileIdc == 118 {
		sps.ChromaFormatIdc = r.ReadUe8()

		if sps.ChromaFormatIdc == 3 {
			// separate_colour_plane_flag
			sps.SeparateColourPlaneFlag = r.ReadBit()
		} else {
			sps.SeparateColourPlaneFlag = 0
		}

		sps.BitDepthLumaMinus8 = r.ReadUe8()
		sps.BitDepthChromaMinus8 = r.ReadUe8()

		// qpprime_y_zero_transform_bypass_flag
		sps.QpprimeYZeroTransformBypassFlag = r.ReadBit()

		sps.SeqScalingMatrixPresentFlag = r.ReadBit()

		if sps.SeqScalingMatrixPresentFlag != 0 {
			maxI := 8
			if sps.ChromaFormatIdc == 3 {
				maxI = 12
			}

			for i := 0; i < maxI; i++ {
				sps.SeqScalingListPresentFlag[i] = r.ReadBit()
				if sps.SeqScalingListPresentFlag[i] != 0 {
					sps.scanList(r, i)
				}
			}
		}
	} else {
		if sps.ProfileIdc == 183 {
			sps.ChromaFormatIdc = 0
		} else {
			sps.ChromaFormatIdc = 1
		}

		sps.SeparateColourPlaneFlag = 0
		sps.BitDepthLumaMinus8 = 0
		sps.BitDepthChromaMinus8 = 0
	}

	// log2_max_frame_num_minus4
	sps.Log2MaxFrameNumMinus4 = r.ReadUe8()

	// pic_order_cnt_type
	sps.PicOrderCntType = r.ReadUe8()
	if sps.PicOrderCntType == 0 {
		// log2_max_pic_order_cnt_lsb_minus4
		sps.Log2MaxPicOrderCntLsbMinus4 = r.ReadUe8()
	} else if sps.PicOrderCntType == 1 {
		// delta_pic_order_always_zero_flag
		sps.DeltaPicOrderAlwaysZeroFlag = r.ReadBit()
		// offset_for_non_ref_pic
		sps.OffsetForNonRefPic = r.ReadSe()

		// offset_for_top_to_bottom_field
		sps.OffsetForTopToBottomField = r.ReadSe()

		// num_ref_frames_in_pic_order_cnt_cycle
		sps.NumRefFramesInPicOrderCntCycle = r.ReadUe8()

		for i := uint8(0); i < sps.NumRefFramesInPicOrderCntCycle; i++ {
			// offset_for_ref_frame
			sps.OffsetForRefFrame[i] = r.ReadSe()
		}
	}

	// max_num_ref_frames
	sps.MaxNumRefFrames = r.ReadUe8()

	// gaps_in_frame_num_allowed_flag
	sps.GapsInFrameNumAllowedFlag = r.ReadBit()

	// pic_width_in_mbs_minus1
	sps.PicWidthInMbsMinus1 = r.ReadUe16()

	// pic_height_in_map_units_minus1
	sps.PicHeightInMapUnitsMinus1 = r.ReadUe16()

	// frame_mbs_only_flag
	sps.FrameMbsOnlyFlag = r.ReadBit()

	if sps.FrameMbsOnlyFlag == 0 {
		// mb_adaptive_frame_field_flag
		sps.MbAdaptiveFrameFieldFlag = r.ReadBit()
	}

	// direct_8x8_inference_flag
	sps.Direct8x8InferenceFlag = r.ReadBit()

	// frame_cropping_flag
	sps.FrameCroppingFlag = r.ReadBit()

	if sps.FrameCroppingFlag == 1 {
		// frame_crop_left_offset
		sps.FrameCropLeftOffset = r.ReadUe16()
		// frame_crop_right_offset
		sps.FrameCropRightOffset = r.ReadUe16()
		// frame_crop_top_offset
		sps.FrameCropTopOffset = r.ReadUe16()
		// frame_crop_bottom_offset
		sps.FrameCropBottomOffset = r.ReadUe16()
	}

	// vui_parameters_present_flag
	sps.VuiParametersPresentFlag = r.ReadBit()

	// vui parameters
	if sps.VuiParametersPresentFlag == 1 {
		if err = sps.Vui.decode(r, sps); err != nil {
			return
		}
	} else {
		sps.Vui.parametersDefault(sps)
	}

	return
}

func (sps *RawSPS) scanList(r *bits.Reader, i int) (err error) {
	var current *[64]int8
	var sizeOfScan int
	if i < 6 {
		current = &sps.ScalingList4x4[i]
		sizeOfScan = 16
	} else {
		current = &sps.ScalingList8x8[i-6]
		sizeOfScan = 64
	}

	scale := 8
	for i = 0; i < sizeOfScan; i++ {
		current[i] = r.ReadSe8()
		scale = (scale + int(current[i]) + 256) % 256
		if scale == 0 {
			break
		}
	}

	return
}

func (h *RawNALUnitHeader) decode(r *bits.Reader) (err error) {
	h.ForbiddenZeroBit = r.ReadBit()
	h.NalRefIdc = r.ReadUint8(2)
	h.NalUnitType = r.ReadUint8(5)

	if h.NalUnitType == NalPrefix ||
		h.NalUnitType == NalExtenSlice ||
		h.NalUnitType == NalDepthExtenSlice {
		err = fmt.Errorf("SVC,3DAVC,MVC not supported.nal_unit_type = %d", h.NalUnitType)
	}
	return
}

func (vui *RawVUI) decode(r *bits.Reader, sps *RawSPS) (err error) {
	vui.AspectRatioInfoPresentFlag = r.ReadBit()
	if vui.AspectRatioInfoPresentFlag == 1 {
		vui.AspectRatioIdc = r.ReadUint8(8)
		if vui.AspectRatioIdc == 255 {
			vui.SarWidth = r.ReadUint16(16)
			vui.SarHeight = r.ReadUint16(16)
		}
	} else {
		vui.AspectRatioIdc = 0
	}

	vui.OverscanInfoPresentFlag = r.ReadBit()
	if vui.OverscanInfoPresentFlag == 1 {
		vui.OverscanAppropriateFlag = r.ReadBit()
	}

	vui.VideoSignalTypePresentFlag = r.ReadBit()
	if vui.VideoSignalTypePresentFlag == 1 {
		vui.VideoFormat = r.ReadUint8(3)
		vui.VideoFullRangeFlag = r.ReadBit()
		vui.ColourDescriptionPresentFlag = r.ReadBit()
		if vui.ColourDescriptionPresentFlag == 1 {
			vui.ColourPrimaries = r.ReadUint8(8)
			vui.TransferCharacteristics = r.ReadUint8(8)
			vui.MatrixCoefficients = r.ReadUint8(8)
		}
	} else {
		vui.VideoFormat = 5
		vui.VideoFullRangeFlag = 0
		vui.ColourPrimaries = 2
		vui.TransferCharacteristics = 2
		vui.MatrixCoefficients = 2
	}

	vui.ChromaLocInfoPresentFlag = r.ReadBit()
	if vui.ChromaLocInfoPresentFlag == 1 {
		vui.ChromaSampleLocTypeTopField = r.ReadUe8()
		vui.ChromaSampleLocTypeBottomField = r.ReadUe8()
	} else {
		vui.ChromaSampleLocTypeTopField = 0
		vui.ChromaSampleLocTypeBottomField = 0
	}

	vui.TimingInfoPresentFlag = r.ReadBit()
	if vui.TimingInfoPresentFlag == 1 {
		vui.NumUnitsInTick = r.ReadUint32(32)
		vui.TimeScale = r.ReadUint32(32)
		vui.FixedFrameRateFlag = r.ReadBit()
	} else {
		vui.FixedFrameRateFlag = 0
	}

	vui.NalHrdParametersPresentFlag = r.ReadBit()
	if vui.NalHrdParametersPresentFlag == 1 {
		if err = vui.NalHrdParameters.decode(r); err != nil {
			return
		}
	}

	vui.VclHrdParametersPresentFlag = r.ReadBit()
	if vui.VclHrdParametersPresentFlag == 1 {
		if err = vui.VclHrdParameters.decode(r); err != nil {
			return
		}
	}

	if vui.NalHrdParametersPresentFlag == 1 ||
		vui.VclHrdParametersPresentFlag == 1 {
		vui.LowDelayHrdFlag = r.ReadBit()
	} else {
		vui.LowDelayHrdFlag = 1 - vui.FixedFrameRateFlag
	}

	vui.PicStructPresentFlag = r.ReadBit()

	vui.BitstreamRestrictionFlag = r.ReadBit()
	if vui.BitstreamRestrictionFlag == 1 {
		vui.MotionVectorsOverPicBoundariesFlag = r.ReadBit()
		vui.MaxBytesPerPicDenom = r.ReadUe8()
		vui.MaxBitsPerMbDenom = r.ReadUe8()
		// The current version of the standard constrains this to be in
		// [0,15], but older versions allow 16.
		vui.Log2MaxMvLengthHorizontal = r.ReadUe8()
		vui.Log2MaxMvLengthVertical = r.ReadUe8()
		vui.MaxNumReorderFrames = r.ReadUe8()
		vui.MaxDecFrameBuffering = r.ReadUe8()
	} else {
		vui.MotionVectorsOverPicBoundariesFlag = 1
		vui.MaxBytesPerPicDenom = 2
		vui.MaxBitsPerMbDenom = 1
		vui.Log2MaxMvLengthHorizontal = 15
		vui.Log2MaxMvLengthVertical = 15

		if (sps.ProfileIdc == 44 || sps.ProfileIdc == 86 ||
			sps.ProfileIdc == 100 || sps.ProfileIdc == 110 ||
			sps.ProfileIdc == 122 || sps.ProfileIdc == 244) &&
			sps.ConstraintSet3Flag == 1 {
			vui.MaxNumReorderFrames = 0
			vui.MaxDecFrameBuffering = 0
		} else {
			vui.MaxNumReorderFrames = MaxDpbFrames
			vui.MaxDecFrameBuffering = MaxDpbFrames
		}
	}

	return
}

func (vui *RawVUI) parametersDefault(sps *RawSPS) (err error) {
	vui.AspectRatioIdc = 0

	vui.VideoFormat = 5
	vui.VideoFullRangeFlag = 0
	vui.ColourPrimaries = 2
	vui.TransferCharacteristics = 2
	vui.MatrixCoefficients = 2

	vui.ChromaSampleLocTypeTopField = 0
	vui.ChromaSampleLocTypeBottomField = 0

	vui.FixedFrameRateFlag = 0
	vui.LowDelayHrdFlag = 1

	vui.PicStructPresentFlag = 0

	vui.MotionVectorsOverPicBoundariesFlag = 1
	vui.MaxBytesPerPicDenom = 2
	vui.MaxBitsPerMbDenom = 1
	vui.Log2MaxMvLengthHorizontal = 15
	vui.Log2MaxMvLengthVertical = 15

	if (sps.ProfileIdc == 44 || sps.ProfileIdc == 86 ||
		sps.ProfileIdc == 100 || sps.ProfileIdc == 110 ||
		sps.ProfileIdc == 122 || sps.ProfileIdc == 244) &&
		sps.ConstraintSet3Flag == 1 {
		vui.MaxNumReorderFrames = 0
		vui.MaxDecFrameBuffering = 0
	} else {
		vui.MaxNumReorderFrames = MaxDpbFrames
		vui.MaxDecFrameBuffering = MaxDpbFrames
	}

	return
}

func (hrd *RawHRD) decode(r *bits.Reader) (err error) {
	hrd.CpbCntMinus1 = r.ReadUe8()
	hrd.BitRateScale = r.ReadUint8(4)
	hrd.CpbSizeScale = r.ReadUint8(4)

	for i := 0; i <= int(hrd.CpbCntMinus1); i++ {
		hrd.BitRateValueMinus1[i] = r.ReadUe()
		hrd.CpbSizeValueMinus1[i] = r.ReadUe()
		hrd.CbrFlag[i] = r.ReadBit()
	}

	hrd.InitialCpbRemovalDelayLengthMinus1 = r.ReadUint8(5)
	hrd.CpbRemovalDelayLengthMinus1 = r.ReadUint8(5)
	hrd.DpbOutputDelayLengthMinus1 = r.ReadUint8(5)
	hrd.TimeOffsetLength = r.ReadUint8(5)
	return
}
