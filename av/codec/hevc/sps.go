// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hevc

import (
	"encoding/base64"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/ipchub/utils/bits"
)

type H265RawScalingList struct {
	Scaling_list_pred_mode_flag       [4][6]uint8
	Scaling_list_pred_matrix_id_delta [4][6]uint8
	Scaling_list_dc_coef_minus8       [4][6]int16
	Scaling_list_delta_coeff          [4][6][64]int8
}

func (sl *H265RawScalingList) decode(r *bits.Reader) error {
	for sizeId := 0; sizeId < 4; sizeId++ {
		step := 1 // (sizeId == 3 ? 3 : 1)
		if sizeId == 3 {
			step = 3
		}
		for matrixId := 0; matrixId < 6; matrixId += step {
			sl.Scaling_list_pred_mode_flag[sizeId][matrixId] = r.ReadBit()
			if sl.Scaling_list_pred_mode_flag[sizeId][matrixId] == 0 {
				sl.Scaling_list_pred_matrix_id_delta[sizeId][matrixId] = r.ReadUe8()
			} else {
				n := 1 << (4 + (sizeId << 1))
				if n > 64 {
					n = 64
				}
				if sizeId > 1 {
					sl.Scaling_list_dc_coef_minus8[sizeId-2][matrixId] = r.ReadSe16()
				}
				for i := 0; i < n; i++ {
					sl.Scaling_list_delta_coeff[sizeId][matrixId][i] = r.ReadSe8()
				}
			}
		}
	}
	return nil
}

type H265RawVUI struct {
	Aspect_ratio_info_present_flag uint8
	Aspect_ratio_idc               uint8
	Sar_width                      uint16
	Sar_height                     uint16

	Overscan_info_present_flag uint8
	Overscan_appropriate_flag  uint8

	Video_signal_type_present_flag  uint8
	Video_format                    uint8
	Video_full_range_flag           uint8
	Colour_description_present_flag uint8
	Colour_primaries                uint8
	Transfer_characteristics        uint8
	Matrix_coefficients             uint8

	Chroma_loc_info_present_flag        uint8
	Chroma_sample_loc_type_top_field    uint8
	Chroma_sample_loc_type_bottom_field uint8

	Neutral_chroma_indication_flag uint8
	Field_seq_flag                 uint8
	Frame_field_info_present_flag  uint8

	Default_display_window_flag uint8
	Def_disp_win_left_offset    uint16
	Def_disp_win_right_offset   uint16
	Def_disp_win_top_offset     uint16
	Def_disp_win_bottom_offset  uint16

	Vui_timing_info_present_flag        uint8
	Vui_num_units_in_tick               uint32
	Vui_time_scale                      uint32
	Vui_poc_proportional_to_timing_flag uint8
	Vui_num_ticks_poc_diff_one_minus1   uint32
	Vui_hrd_parameters_present_flag     uint8
	Hrd_parameters                      H265RawHRDParameters

	Bitstream_restriction_flag              uint8
	Tiles_fixed_structure_flag              uint8
	Motion_vectors_over_pic_boundaries_flag uint8
	Restricted_ref_pic_lists_flag           uint8
	Min_spatial_segmentation_idc            uint16
	Max_bytes_per_pic_denom                 uint8
	Max_bits_per_min_cu_denom               uint8
	Log2_max_mv_length_horizontal           uint8
	Log2_max_mv_length_vertical             uint8
}

func (vui *H265RawVUI) setDefault(sps *H265RawSPS) {
	vui.Aspect_ratio_idc = 0

	vui.Video_format = 5
	vui.Video_full_range_flag = 0
	vui.Colour_primaries = 2
	vui.Transfer_characteristics = 2
	vui.Matrix_coefficients = 2

	vui.Chroma_sample_loc_type_top_field = 0
	vui.Chroma_sample_loc_type_bottom_field = 0

	vui.Tiles_fixed_structure_flag = 0
	vui.Motion_vectors_over_pic_boundaries_flag = 1
	vui.Min_spatial_segmentation_idc = 0
	vui.Max_bytes_per_pic_denom = 2
	vui.Max_bits_per_min_cu_denom = 1
	vui.Log2_max_mv_length_horizontal = 15
	vui.Log2_max_mv_length_vertical = 15
}

func (vui *H265RawVUI) decode(r *bits.Reader, sps *H265RawSPS) error {
	vui.Aspect_ratio_info_present_flag = r.ReadBit()
	if vui.Aspect_ratio_info_present_flag == 1 {
		vui.Aspect_ratio_idc = r.ReadUint8(8)
		if vui.Aspect_ratio_idc == 255 {
			vui.Sar_width = r.ReadUint16(16)
			vui.Sar_height = r.ReadUint16(16)
		}
	} else {
		vui.Aspect_ratio_idc = 0
	}

	vui.Overscan_info_present_flag = r.ReadBit()
	if vui.Overscan_info_present_flag == 1 {
		vui.Overscan_appropriate_flag = r.ReadBit()
	}

	vui.Video_signal_type_present_flag = r.ReadBit()
	if vui.Video_signal_type_present_flag == 1 {
		vui.Video_format = r.ReadUint8(3)
		vui.Video_full_range_flag = r.ReadBit()
		vui.Colour_description_present_flag = r.ReadBit()
		if vui.Colour_description_present_flag == 1 {
			vui.Colour_primaries = r.ReadUint8(8)
			vui.Transfer_characteristics = r.ReadUint8(8)
			vui.Matrix_coefficients = r.ReadUint8(8)
		} else {
			vui.Colour_primaries = 2
			vui.Transfer_characteristics = 2
			vui.Matrix_coefficients = 2
		}
	} else {
		vui.Video_format = 5
		vui.Video_full_range_flag = 0
		vui.Colour_primaries = 2
		vui.Transfer_characteristics = 2
		vui.Matrix_coefficients = 2
	}

	vui.Chroma_loc_info_present_flag = r.ReadBit()
	if vui.Chroma_loc_info_present_flag == 1 {
		vui.Chroma_sample_loc_type_top_field = r.ReadUe8()
		vui.Chroma_sample_loc_type_bottom_field = r.ReadUe8()
	} else {
		vui.Chroma_sample_loc_type_top_field = 0
		vui.Chroma_sample_loc_type_bottom_field = 0
	}

	vui.Neutral_chroma_indication_flag = r.ReadBit()
	vui.Field_seq_flag = r.ReadBit()
	vui.Frame_field_info_present_flag = r.ReadBit()

	vui.Default_display_window_flag = r.ReadBit()
	if vui.Default_display_window_flag == 1 {
		vui.Def_disp_win_left_offset = r.ReadUe16()
		vui.Def_disp_win_right_offset = r.ReadUe16()
		vui.Def_disp_win_top_offset = r.ReadUe16()
		vui.Def_disp_win_bottom_offset = r.ReadUe16()
	}

	vui.Vui_timing_info_present_flag = r.ReadBit()
	if vui.Vui_timing_info_present_flag == 1 {
		vui.Vui_num_units_in_tick = r.ReadUint32(32)
		vui.Vui_time_scale = r.ReadUint32(32)
		vui.Vui_poc_proportional_to_timing_flag = r.ReadBit()
		if vui.Vui_poc_proportional_to_timing_flag == 1 {
			vui.Vui_num_ticks_poc_diff_one_minus1 = r.ReadUe()
		}

		vui.Vui_hrd_parameters_present_flag = r.ReadBit()
		if vui.Vui_hrd_parameters_present_flag == 1 {
			if err := vui.Hrd_parameters.decode(r, true, int(sps.Sps_max_sub_layers_minus1)); err != nil {
				return err
			}
		}
	}

	vui.Bitstream_restriction_flag = r.ReadBit()
	if vui.Bitstream_restriction_flag == 1 {
		vui.Tiles_fixed_structure_flag = r.ReadBit()
		vui.Motion_vectors_over_pic_boundaries_flag = r.ReadBit()
		vui.Restricted_ref_pic_lists_flag = r.ReadBit()
		vui.Min_spatial_segmentation_idc = r.ReadUe16()
		vui.Max_bytes_per_pic_denom = r.ReadUe8()
		vui.Max_bits_per_min_cu_denom = r.ReadUe8()
		vui.Log2_max_mv_length_horizontal = r.ReadUe8()
		vui.Log2_max_mv_length_vertical = r.ReadUe8()
	} else {
		vui.Tiles_fixed_structure_flag = 0
		vui.Motion_vectors_over_pic_boundaries_flag = 1
		vui.Min_spatial_segmentation_idc = 0
		vui.Max_bytes_per_pic_denom = 2
		vui.Max_bits_per_min_cu_denom = 1
		vui.Log2_max_mv_length_horizontal = 15
		vui.Log2_max_mv_length_vertical = 15
	}

	return nil
}

type H265RawSTRefPicSet struct {
	Inter_ref_pic_set_prediction_flag uint8

	Delta_idx_minus1     uint8
	Delta_rps_sign       uint8
	Abs_delta_rps_minus1 uint16

	Used_by_curr_pic_flag [HEVC_MAX_REFS]uint8
	Use_delta_flag        [HEVC_MAX_REFS]uint8

	Num_negative_pics        uint8
	Num_positive_pics        uint8
	Delta_poc_s0_minus1      [HEVC_MAX_REFS]uint16
	Used_by_curr_pic_s0_flag [HEVC_MAX_REFS]uint8
	Delta_poc_s1_minus1      [HEVC_MAX_REFS]uint16
	Used_by_curr_pic_s1_flag [HEVC_MAX_REFS]uint8
}

func (ps *H265RawSTRefPicSet) decode(r *bits.Reader, st_rps_idx uint8, sps *H265RawSPS) error {
	if st_rps_idx != 0 {
		ps.Inter_ref_pic_set_prediction_flag = r.ReadBit()
	} else {
		ps.Inter_ref_pic_set_prediction_flag = 0
	}

	if ps.Inter_ref_pic_set_prediction_flag == 1 {
		var ref_rps_idx, num_delta_pocs, num_ref_pics uint8
		var ref *H265RawSTRefPicSet
		var delta_rps, d_poc int
		var ref_delta_poc_s0, ref_delta_poc_s1, delta_poc_s0, delta_poc_s1 [HEVC_MAX_REFS]int
		var used_by_curr_pic_s0, used_by_curr_pic_s1 [HEVC_MAX_REFS]uint8

		if st_rps_idx == sps.Num_short_term_ref_pic_sets {
			ps.Delta_idx_minus1 = r.ReadUe8()
		} else {
			ps.Delta_idx_minus1 = 0
		}

		ref_rps_idx = st_rps_idx - (ps.Delta_idx_minus1 + 1)
		ref = &sps.St_ref_pic_set[ref_rps_idx]
		num_delta_pocs = ref.Num_negative_pics + ref.Num_positive_pics
		// av_assert0(num_delta_pocs < HEVC_MAX_DPB_SIZE);

		ps.Delta_rps_sign = r.ReadBit()
		ps.Abs_delta_rps_minus1 = r.ReadUe16()
		delta_rps = int((1 - 2*ps.Delta_rps_sign)) * int(ps.Abs_delta_rps_minus1+1)

		num_ref_pics = 0
		for j := 0; j <= int(num_delta_pocs); j++ {
			ps.Used_by_curr_pic_flag[j] = r.ReadBit()
			if ps.Used_by_curr_pic_flag[j] == 0 {
				ps.Use_delta_flag[j] = r.ReadBit()
			} else {
				ps.Use_delta_flag[j] = 1
			}
			if ps.Use_delta_flag[j] == 1 {
				num_ref_pics++
			}
		}
		if num_ref_pics >= HEVC_MAX_DPB_SIZE {
			return errors.New("Invalid stream: short-term ref pic set %d contains too many pictures.\n")
		}

		// Since the stored form of an RPS here is actually the delta-step
		// form used when inter_ref_pic_set_prediction_flag is not set, we
		// need to reconstruct that here in order to be able to refer to
		// the RPS later (which is required for parsing, because we don't
		// even know what syntax elements appear without it).  Therefore,
		// this code takes the delta-step form of the reference set, turns
		// it into the delta-array form, applies the prediction process of
		// 7.4.8, converts the result back to the delta-step form, and
		// stores that as the current set for future use.  Note that the
		// inferences here mean that writers using prediction will need
		// to fill in the delta-step values correctly as well - since the
		// whole RPS prediction process is somewhat overly sophisticated,
		// this hopefully forms a useful check for them to ensure their
		// predicted form actually matches what was intended rather than
		// an onerous additional requirement.

		d_poc = 0
		for i := 0; i < int(ref.Num_negative_pics); i++ {
			d_poc -= int(ref.Delta_poc_s0_minus1[i] + 1)
			ref_delta_poc_s0[i] = d_poc
		}
		d_poc = 0
		for i := 0; i < int(ref.Num_positive_pics); i++ {
			d_poc += int(ref.Delta_poc_s1_minus1[i] + 1)
			ref_delta_poc_s1[i] = d_poc
		}

		i := 0
		for j := ref.Num_positive_pics - 1; j >= 0; j-- {
			d_poc = ref_delta_poc_s1[j] + delta_rps
			if d_poc < 0 && ps.Use_delta_flag[ref.Num_negative_pics+j] == 1 {
				delta_poc_s0[i] = d_poc
				i++
				used_by_curr_pic_s0[i] =
					ps.Used_by_curr_pic_flag[ref.Num_negative_pics+j]
			}
		}
		if delta_rps < 0 && ps.Use_delta_flag[num_delta_pocs] == 1 {
			delta_poc_s0[i] = delta_rps
			i++
			used_by_curr_pic_s0[i] =
				ps.Used_by_curr_pic_flag[num_delta_pocs]
		}
		for j := 0; j < int(ref.Num_negative_pics); j++ {
			d_poc = ref_delta_poc_s0[j] + delta_rps
			if d_poc < 0 && ps.Use_delta_flag[j] == 1 {
				delta_poc_s0[i] = d_poc
				i++
				used_by_curr_pic_s0[i] = ps.Used_by_curr_pic_flag[j]
			}
		}

		ps.Num_negative_pics = uint8(i)
		for i := 0; i < int(ps.Num_negative_pics); i++ {
			if i == 0 {
				ps.Delta_poc_s0_minus1[i] =
					uint16(-delta_poc_s0[i] - 1)
			} else {
				ps.Delta_poc_s0_minus1[i] =
					uint16(-(delta_poc_s0[i] - delta_poc_s0[i-1]) - 1)
			}
			ps.Used_by_curr_pic_s0_flag[i] = used_by_curr_pic_s0[i]
		}

		i = 0
		for j := ref.Num_negative_pics - 1; j >= 0; j-- {
			d_poc = ref_delta_poc_s0[j] + delta_rps
			if d_poc > 0 && ps.Use_delta_flag[j] == 1 {
				delta_poc_s1[i] = d_poc
				i++
				used_by_curr_pic_s1[i] = ps.Used_by_curr_pic_flag[j]
			}
		}
		if delta_rps > 0 && ps.Use_delta_flag[num_delta_pocs] == 1 {
			delta_poc_s1[i] = delta_rps
			i++
			used_by_curr_pic_s1[i] =
				ps.Used_by_curr_pic_flag[num_delta_pocs]
		}
		for j := 0; j < int(ref.Num_positive_pics); j++ {
			d_poc = ref_delta_poc_s1[j] + delta_rps
			if d_poc > 0 && ps.Use_delta_flag[int(ref.Num_negative_pics)+j] == 1 {
				delta_poc_s1[i] = d_poc
				i++
				used_by_curr_pic_s1[i] =
					ps.Used_by_curr_pic_flag[int(ref.Num_negative_pics)+j]
			}
		}

		ps.Num_positive_pics = 1
		for i := 0; i < int(ps.Num_positive_pics); i++ {
			if i == 0 {
				ps.Delta_poc_s1_minus1[i] =
					uint16(delta_poc_s1[i] - 1)
			} else {
				ps.Delta_poc_s1_minus1[i] =
					uint16(delta_poc_s1[i] - delta_poc_s1[i-1] - 1)
			}

			ps.Used_by_curr_pic_s1_flag[i] = used_by_curr_pic_s1[i]
		}

	} else {
		ps.Num_negative_pics = r.ReadUe8()
		ps.Num_positive_pics = r.ReadUe8()

		for i := 0; i < int(ps.Num_negative_pics); i++ {
			ps.Delta_poc_s0_minus1[i] = r.ReadUe16()
			ps.Used_by_curr_pic_s0_flag[i] = r.ReadBit()
		}

		for i := 0; i < int(ps.Num_positive_pics); i++ {
			ps.Delta_poc_s1_minus1[i] = r.ReadUe16()
			ps.Used_by_curr_pic_s1_flag[i] = r.ReadBit()
		}
	}

	return nil
}

type H265RawSPS struct {
	Nal_unit_header H265RawNALUnitHeader

	Sps_video_parameter_set_id uint8

	Sps_max_sub_layers_minus1    uint8
	Sps_temporal_id_nesting_flag uint8

	Profile_tier_level H265RawProfileTierLevel

	Sps_seq_parameter_set_id uint8

	Chroma_format_idc          uint8
	Separate_colour_plane_flag uint8

	Pic_width_in_luma_samples  uint16
	Pic_height_in_luma_samples uint16

	Conformance_window_flag uint8
	Conf_win_left_offset    uint16
	Conf_win_right_offset   uint16
	Conf_win_top_offset     uint16
	Conf_win_bottom_offset  uint16

	Bit_depth_luma_minus8   uint8
	Bit_depth_chroma_minus8 uint8

	Log2_max_pic_order_cnt_lsb_minus4 uint8

	Sps_sub_layer_ordering_info_present_flag uint8
	Sps_max_dec_pic_buffering_minus1         [HEVC_MAX_SUB_LAYERS]uint8
	Sps_max_num_reorder_pics                 [HEVC_MAX_SUB_LAYERS]uint8
	Sps_max_latency_increase_plus1           [HEVC_MAX_SUB_LAYERS]uint32

	Log2_min_luma_coding_block_size_minus3      uint8
	Log2_diff_max_min_luma_coding_block_size    uint8
	Log2_min_luma_transform_block_size_minus2   uint8
	Log2_diff_max_min_luma_transform_block_size uint8
	Max_transform_hierarchy_depth_inter         uint8
	Max_transform_hierarchy_depth_intra         uint8

	Scaling_list_enabled_flag          uint8
	Sps_scaling_list_data_present_flag uint8
	Scaling_list                       *H265RawScalingList

	Amp_enabled_flag                    uint8
	Sample_adaptive_offset_enabled_flag uint8

	Pcm_enabled_flag                             uint8
	Pcm_sample_bit_depth_luma_minus1             uint8
	Pcm_sample_bit_depth_chroma_minus1           uint8
	Log2_min_pcm_luma_coding_block_size_minus3   uint8
	Log2_diff_max_min_pcm_luma_coding_block_size uint8
	Pcm_loop_filter_disabled_flag                uint8

	Num_short_term_ref_pic_sets uint8
	St_ref_pic_set              []H265RawSTRefPicSet //[HEVC_MAX_SHORT_TERM_REF_PIC_SETS]H265RawSTRefPicSet

	Long_term_ref_pics_present_flag uint8
	Num_long_term_ref_pics_sps      uint8
	Lt_ref_pic_poc_lsb_sps          [HEVC_MAX_LONG_TERM_REF_PICS]uint16
	Used_by_curr_pic_lt_sps_flag    [HEVC_MAX_LONG_TERM_REF_PICS]uint8

	Sps_temporal_mvp_enabled_flag       uint8
	Strong_intra_smoothing_enabled_flag uint8

	Vui_parameters_present_flag uint8
	Vui                         H265RawVUI

	Sps_extension_present_flag    uint8
	Sps_range_extension_flag      uint8
	Sps_multilayer_extension_flag uint8
	Sps_3d_extension_flag         uint8
	Sps_scc_extension_flag        uint8
	Sps_extension_4bits           uint8

	// extension_data H265RawExtensionData

	// // Range extension.
	// transform_skip_rotation_enabled_flag    uint8
	// transform_skip_context_enabled_flag     uint8
	// implicit_rdpcm_enabled_flag             uint8
	// explicit_rdpcm_enabled_flag             uint8
	// extended_precision_processing_flag      uint8
	// intra_smoothing_disabled_flag           uint8
	// high_precision_offsets_enabled_flag     uint8
	// persistent_rice_adaptation_enabled_flag uint8
	// cabac_bypass_alignment_enabled_flag     uint8

	// // Screen content coding extension.
	// sps_curr_pic_ref_enabled_flag                  uint8
	// palette_mode_enabled_flag                      uint8
	// palette_max_size                               uint8
	// delta_palette_max_predictor_size               uint8
	// sps_palette_predictor_initializer_present_flag uint8
	// sps_num_palette_predictor_initializer_minus1   uint8
	// sps_palette_predictor_initializers             [3][128]uint16

	// motion_vector_resolution_control_idc  uint8
	// intra_boundary_filtering_disable_flag uint8
}

// Width 视频宽度（像素）
func (sps *H265RawSPS) Width() int {
	return int(sps.Pic_width_in_luma_samples)
}

// Height 视频高度（像素）
func (sps *H265RawSPS) Height() int {
	return int(sps.Pic_height_in_luma_samples)
}

// FrameRate Video frame rate
func (sps *H265RawSPS) FrameRate() float64 {
	if sps.Vui.Vui_num_units_in_tick == 0 {
		return 0.0
	}
	return float64(sps.Vui.Vui_time_scale) / float64(sps.Vui.Vui_num_units_in_tick)
}

// IsFixedFrameRate 是否固定帧率
func (sps *H265RawSPS) IsFixedFrameRate() bool {
	// TODO:
	return sps.FrameRate() > 0
}

// DecodeString 从 base64 字串解码 sps NAL
func (sps *H265RawSPS) DecodeString(b64 string) error {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return sps.Decode(data)
}

// Decode 从字节序列中解码 sps NAL
func (sps *H265RawSPS) Decode(data []byte) (err error) {
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
	if err = sps.Nal_unit_header.decode(r); err != nil {
		return
	}

	if sps.Nal_unit_header.Nal_unit_type != NalSps {
		return errors.New("not is sps NAL UNIT")
	}

	sps.Sps_video_parameter_set_id = r.ReadUint8(4)

	sps.Sps_max_sub_layers_minus1 = r.ReadUint8(3)
	sps.Sps_temporal_id_nesting_flag = r.ReadBit()
	if err = sps.Profile_tier_level.decode(r, true, int(sps.Sps_max_sub_layers_minus1)); err != nil {
		return
	}

	sps.Sps_seq_parameter_set_id = r.ReadUe8()

	sps.Chroma_format_idc = r.ReadUe8()
	if sps.Chroma_format_idc == 3 {
		sps.Separate_colour_plane_flag = r.ReadBit()
	}

	sps.Pic_width_in_luma_samples = r.ReadUe16()
	sps.Pic_height_in_luma_samples = r.ReadUe16()

	sps.Conformance_window_flag = r.ReadBit()
	if sps.Conformance_window_flag == 1 {
		sps.Conf_win_left_offset = r.ReadUe16()
		sps.Conf_win_right_offset = r.ReadUe16()
		sps.Conf_win_top_offset = r.ReadUe16()
		sps.Conf_win_bottom_offset = r.ReadUe16()
	}

	sps.Bit_depth_luma_minus8 = r.ReadUe8()
	sps.Bit_depth_chroma_minus8 = r.ReadUe8()

	sps.Log2_max_pic_order_cnt_lsb_minus4 = r.ReadUe8()

	sps.Sps_sub_layer_ordering_info_present_flag = r.ReadBit()
	loopStart := uint8(0)
	if sps.Sps_sub_layer_ordering_info_present_flag == 1 {
		loopStart = sps.Sps_max_sub_layers_minus1
	}
	for i := loopStart; i <= sps.Sps_max_sub_layers_minus1; i++ {
		sps.Sps_max_dec_pic_buffering_minus1[i] = r.ReadUe8()
		sps.Sps_max_num_reorder_pics[i] = r.ReadUe8()
		sps.Sps_max_latency_increase_plus1[i] = r.ReadUe()
	}

	if sps.Sps_sub_layer_ordering_info_present_flag == 0 {
		for i := uint8(0); i < sps.Sps_max_sub_layers_minus1; i++ {

			sps.Sps_max_dec_pic_buffering_minus1[i] =
				sps.Sps_max_dec_pic_buffering_minus1[sps.Sps_max_sub_layers_minus1]
			sps.Sps_max_num_reorder_pics[i] =
				sps.Sps_max_num_reorder_pics[sps.Sps_max_sub_layers_minus1]
			sps.Sps_max_latency_increase_plus1[i] =
				sps.Sps_max_latency_increase_plus1[sps.Sps_max_sub_layers_minus1]
		}
	}

	sps.Log2_min_luma_coding_block_size_minus3 = r.ReadUe8()
	min_cb_log2_size_y := sps.Log2_min_luma_coding_block_size_minus3 + 3

	sps.Log2_diff_max_min_luma_coding_block_size = r.ReadUe8()
	// ctb_log2_size_y := min_cb_log2_size_y +
	// 	sps.log2_diff_max_min_luma_coding_block_size

	min_cb_size_y := uint16(1) << min_cb_log2_size_y
	if (sps.Pic_width_in_luma_samples%min_cb_size_y) > 0 ||
		(sps.Pic_height_in_luma_samples%min_cb_size_y) > 0 {
		return fmt.Errorf("Invalid dimensions: %v%v not divisible by MinCbSizeY = %v.\n",
			sps.Pic_width_in_luma_samples,
			sps.Pic_height_in_luma_samples,
			min_cb_size_y)
	}

	sps.Log2_min_luma_transform_block_size_minus2 = r.ReadUe8()
	// min_tb_log2_size_y := sps.log2_min_luma_transform_block_size_minus2 + 2

	sps.Log2_diff_max_min_luma_transform_block_size = r.ReadUe8()

	sps.Max_transform_hierarchy_depth_inter = r.ReadUe8()
	sps.Max_transform_hierarchy_depth_intra = r.ReadUe8()

	sps.Scaling_list_enabled_flag = r.ReadBit()
	if sps.Scaling_list_enabled_flag == 1 {
		sps.Sps_scaling_list_data_present_flag = r.ReadBit()
		if sps.Sps_scaling_list_data_present_flag == 1 {
			sps.Scaling_list = new(H265RawScalingList)
			sps.Scaling_list.decode(r)
		}
	}

	sps.Amp_enabled_flag = r.ReadBit()
	sps.Sample_adaptive_offset_enabled_flag = r.ReadBit()

	sps.Pcm_enabled_flag = r.ReadBit()
	if sps.Pcm_enabled_flag == 1 {
		sps.Pcm_sample_bit_depth_luma_minus1 = r.ReadUint8(4)
		sps.Pcm_sample_bit_depth_chroma_minus1 = r.ReadUint8(4)

		sps.Log2_min_pcm_luma_coding_block_size_minus3 = r.ReadUe8()
		sps.Log2_diff_max_min_pcm_luma_coding_block_size = r.ReadUe8()

		sps.Pcm_loop_filter_disabled_flag = r.ReadBit()
	}

	sps.Num_short_term_ref_pic_sets = r.ReadUe8()
	if sps.Num_short_term_ref_pic_sets > 0 {
		sps.St_ref_pic_set = make([]H265RawSTRefPicSet, sps.Num_short_term_ref_pic_sets)
		for i := uint8(0); i < sps.Num_short_term_ref_pic_sets; i++ {
			sps.St_ref_pic_set[i].decode(r, i, sps)
		}
	}

	sps.Long_term_ref_pics_present_flag = r.ReadBit()
	if sps.Long_term_ref_pics_present_flag == 1 {
		sps.Num_long_term_ref_pics_sps = r.ReadUe8()
		for i := uint8(0); i < sps.Num_long_term_ref_pics_sps; i++ {
			sps.Lt_ref_pic_poc_lsb_sps[i] = r.ReadUint16(int(sps.Log2_max_pic_order_cnt_lsb_minus4 + 4))
			sps.Used_by_curr_pic_lt_sps_flag[i] = r.ReadBit()
		}
	}

	sps.Sps_temporal_mvp_enabled_flag = r.ReadBit()
	sps.Strong_intra_smoothing_enabled_flag = r.ReadBit()

	sps.Vui_parameters_present_flag = r.ReadBit()
	if sps.Vui_parameters_present_flag == 1 {
		sps.Vui.decode(r, sps)
	} else {
		sps.Vui.setDefault(sps)
	}

	sps.Sps_extension_present_flag = r.ReadBit()

	if sps.Sps_extension_present_flag == 1 {
		sps.Sps_range_extension_flag = r.ReadBit()
		sps.Sps_multilayer_extension_flag = r.ReadBit()
		sps.Sps_3d_extension_flag = r.ReadBit()
		sps.Sps_scc_extension_flag = r.ReadBit()
		sps.Sps_extension_4bits = r.ReadUint8(4)
	}

	// if (sps.sps_range_extension_flag)
	//     CHECK(FUNC(sps_range_extension)(ctx, rw, current));
	// if (sps.sps_multilayer_extension_flag)
	//     return AVERROR_PATCHWELCOME;
	// if (sps.sps_3d_extension_flag)
	//     return AVERROR_PATCHWELCOME;
	// if (sps.sps_scc_extension_flag)
	//     CHECK(FUNC(sps_scc_extension)(ctx, rw, current));
	// if (sps.sps_extension_4bits)
	//     CHECK(FUNC(extension_data)(ctx, rw, &sps.extension_data));

	// CHECK(FUNC(rbsp_trailing_bits)(ctx, rw));

	return
}
