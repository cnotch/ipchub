// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Translate from FFmpeg cbs_h265.h cbs_h265_syntax_template.c
//
package hevc

import (
	"encoding/base64"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/ipchub/utils/bits"
)

type H265RawNALUnitHeader struct {
	Nal_unit_type         uint8
	Nuh_layer_id          uint8
	Nuh_temporal_id_plus1 uint8
}

func (h *H265RawNALUnitHeader) decode(r *bits.Reader) (err error) {
	r.Skip(1) //forbidden_zero_bit
	h.Nal_unit_type = r.ReadUint8(6)
	h.Nuh_layer_id = r.ReadUint8(6)
	h.Nuh_temporal_id_plus1 = r.ReadUint8(3)
	return
}

type H265RawProfileTierLevel struct {
	General_profile_space uint8
	General_tier_flag     uint8
	General_profile_idc   uint8

	General_profile_compatibility_flag [32]uint8
	GeneralProfileCompatibilityFlags   uint32 // shortcut flags 32bits

	General_progressive_source_flag    uint8
	General_interlaced_source_flag     uint8
	General_non_packed_constraint_flag uint8
	General_frame_only_constraint_flag uint8

	General_max_12bit_constraint_flag        uint8
	General_max_10bit_constraint_flag        uint8
	General_max_8bit_constraint_flag         uint8
	General_max_422chroma_constraint_flag    uint8
	General_max_420chroma_constraint_flag    uint8
	General_max_monochrome_constraint_flag   uint8
	General_intra_constraint_flag            uint8
	General_one_picture_only_constraint_flag uint8
	General_lower_bit_rate_constraint_flag   uint8
	General_max_14bit_constraint_flag        uint8

	General_inbld_flag              uint8
	GeneralConstraintIndicatorFlags uint64 // shortcut flags 48bits

	General_level_idc uint8

	Sub_layer_profile_present_flag [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_level_present_flag   [HEVC_MAX_SUB_LAYERS]uint8

	Sub_layer_profile_space [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_tier_flag     [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_profile_idc   [HEVC_MAX_SUB_LAYERS]uint8

	Sub_layer_profile_compatibility_flag [HEVC_MAX_SUB_LAYERS][32]uint8

	Sub_layer_progressive_source_flag    [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_interlaced_source_flag     [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_non_packed_constraint_flag [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_frame_only_constraint_flag [HEVC_MAX_SUB_LAYERS]uint8

	Sub_layer_max_12bit_constraint_flag        [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_max_10bit_constraint_flag        [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_max_8bit_constraint_flag         [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_max_422chroma_constraint_flag    [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_max_420chroma_constraint_flag    [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_max_monochrome_constraint_flag   [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_intra_constraint_flag            [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_one_picture_only_constraint_flag [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_lower_bit_rate_constraint_flag   [HEVC_MAX_SUB_LAYERS]uint8
	Sub_layer_max_14bit_constraint_flag        [HEVC_MAX_SUB_LAYERS]uint8

	Sub_layer_inbld_flag [HEVC_MAX_SUB_LAYERS]uint8

	Sub_layer_level_idc [HEVC_MAX_SUB_LAYERS]uint8
}
type profile_compatible struct {
	profile_idc                uint8
	profile_compatibility_flag [32]uint8
}

func (pc profile_compatible) compatible(idc uint8) bool {
	return pc.profile_idc == idc || pc.profile_compatibility_flag[idc] == 1
}

func (ptl *H265RawProfileTierLevel) decode(r *bits.Reader,
	profile_present_flag bool, max_num_sub_layers_minus1 int) (err error) {

	if profile_present_flag {
		ptl.General_profile_space = r.ReadUint8(2)
		ptl.General_tier_flag = r.ReadBit()
		ptl.General_profile_idc = r.ReadUint8(5)

		ptl.GeneralProfileCompatibilityFlags = uint32(r.Peek(32))
		for j := 0; j < 32; j++ {
			ptl.General_profile_compatibility_flag[j] = r.ReadBit()
		}

		ptl.GeneralConstraintIndicatorFlags = r.Peek(48)
		ptl.General_progressive_source_flag = r.ReadBit()
		ptl.General_interlaced_source_flag = r.ReadBit()
		ptl.General_non_packed_constraint_flag = r.ReadBit()
		ptl.General_frame_only_constraint_flag = r.ReadBit()

		pc := profile_compatible{ptl.General_profile_idc, ptl.General_profile_compatibility_flag}
		if pc.compatible(4) || pc.compatible(5) ||
			pc.compatible(6) || pc.compatible(7) ||
			pc.compatible(8) || pc.compatible(9) ||
			pc.compatible(10) {
			ptl.General_max_12bit_constraint_flag = r.ReadBit()
			ptl.General_max_10bit_constraint_flag = r.ReadBit()
			ptl.General_max_8bit_constraint_flag = r.ReadBit()
			ptl.General_max_422chroma_constraint_flag = r.ReadBit()
			ptl.General_max_420chroma_constraint_flag = r.ReadBit()
			ptl.General_max_monochrome_constraint_flag = r.ReadBit()
			ptl.General_intra_constraint_flag = r.ReadBit()
			ptl.General_one_picture_only_constraint_flag = r.ReadBit()
			ptl.General_lower_bit_rate_constraint_flag = r.ReadBit()

			if pc.compatible(5) || pc.compatible(9) || pc.compatible(10) {
				ptl.General_max_14bit_constraint_flag = r.ReadBit()
				r.Skip(33) // general_reserved_zero_33bits

			} else {
				r.Skip(34) //general_reserved_zero_34bits
			}
		} else if pc.compatible(2) {
			r.Skip(7) // general_reserved_zero_7bits
			ptl.General_one_picture_only_constraint_flag = r.ReadBit()
			r.Skip(35) // general_reserved_zero_35bits
		} else {
			r.Skip(43) // general_reserved_zero_43bits
		}

		if pc.compatible(1) || pc.compatible(2) ||
			pc.compatible(3) || pc.compatible(4) ||
			pc.compatible(5) || pc.compatible(9) {
			ptl.General_inbld_flag = r.ReadBit()
		} else {
			r.Skip(1) // general_reserved_zero_bit
		}
	}

	ptl.General_level_idc = r.ReadUint8(8)

	for i := 0; i < max_num_sub_layers_minus1; i++ {
		ptl.Sub_layer_profile_present_flag[i] = r.ReadBit()
		ptl.Sub_layer_level_present_flag[i] = r.ReadBit()
	}

	if max_num_sub_layers_minus1 > 0 {
		for i := max_num_sub_layers_minus1; i < 8; i++ {
			r.Skip(2) // reserved_zero_2bits
		}
	}

	for i := 0; i < max_num_sub_layers_minus1; i++ {
		if ptl.Sub_layer_profile_present_flag[i] == 1 {
			ptl.Sub_layer_profile_space[i] = r.ReadUint8(2)
			ptl.Sub_layer_tier_flag[i] = r.ReadBit()
			ptl.Sub_layer_profile_idc[i] = r.ReadUint8(5)

			for j := 0; j < 32; j++ {
				ptl.Sub_layer_profile_compatibility_flag[i][j] = r.ReadBit()
			}

			ptl.Sub_layer_progressive_source_flag[i] = r.ReadBit()
			ptl.Sub_layer_interlaced_source_flag[i] = r.ReadBit()
			ptl.Sub_layer_non_packed_constraint_flag[i] = r.ReadBit()
			ptl.Sub_layer_frame_only_constraint_flag[i] = r.ReadBit()

			pc := profile_compatible{ptl.Sub_layer_profile_idc[i], ptl.Sub_layer_profile_compatibility_flag[i]}
			if pc.compatible(4) || pc.compatible(5) ||
				pc.compatible(6) || pc.compatible(7) ||
				pc.compatible(8) || pc.compatible(9) ||
				pc.compatible(10) {
				ptl.Sub_layer_max_12bit_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_max_10bit_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_max_8bit_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_max_422chroma_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_max_420chroma_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_max_monochrome_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_intra_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_one_picture_only_constraint_flag[i] = r.ReadBit()
				ptl.Sub_layer_lower_bit_rate_constraint_flag[i] = r.ReadBit()

				if pc.compatible(5) {
					ptl.Sub_layer_max_14bit_constraint_flag[i] = r.ReadBit()
					r.Skip(33) // sub_layer_reserved_zero_33bits
				} else {
					r.Skip(34) // sub_layer_reserved_zero_34bits
				}
			} else if pc.compatible(2) {
				r.Skip(7) // sub_layer_reserved_zero_7bits
				ptl.Sub_layer_one_picture_only_constraint_flag[i] = r.ReadBit()
				r.Skip(35) // sub_layer_reserved_zero_35bits
			} else {
				r.Skip(43) // sub_layer_reserved_zero_43bits
			}

			if pc.compatible(1) || pc.compatible(2) ||
				pc.compatible(3) || pc.compatible(4) ||
				pc.compatible(5) || pc.compatible(9) {
				ptl.Sub_layer_inbld_flag[i] = r.ReadBit()
			} else {
				r.Skip(1) // sub_layer_reserved_zero_bit
			}
		}
		if ptl.Sub_layer_level_present_flag[i] == 1 {
			ptl.Sub_layer_level_idc[i] = r.ReadUint8(8)
		}
	}
	return
}

type H265RawSubLayerHRDParameters struct {
	Bit_rate_value_minus1    [HEVC_MAX_CPB_CNT]uint32
	Cpb_size_value_minus1    [HEVC_MAX_CPB_CNT]uint32
	Cpb_size_du_value_minus1 [HEVC_MAX_CPB_CNT]uint32
	Bit_rate_du_value_minus1 [HEVC_MAX_CPB_CNT]uint32
	Cbr_flag                 [HEVC_MAX_CPB_CNT]uint8
}

func (shrd *H265RawSubLayerHRDParameters) decode(r *bits.Reader,
	sub_pic_hrd_params_present_flag bool, cpb_cnt_minus1 int) (err error) {
	for i := 0; i <= cpb_cnt_minus1; i++ {
		shrd.Bit_rate_value_minus1[i] = r.ReadUe()
		shrd.Cpb_size_value_minus1[i] = r.ReadUe()
		if sub_pic_hrd_params_present_flag {
			shrd.Cpb_size_du_value_minus1[i] = r.ReadUe()
			shrd.Bit_rate_du_value_minus1[i] = r.ReadUe()
		}
		shrd.Cbr_flag[i] = r.ReadBit()
	}
	return
}

type H265RawHRDParameters struct {
	Nal_hrd_parameters_present_flag uint8
	Vcl_hrd_parameters_present_flag uint8

	Sub_pic_hrd_params_present_flag              uint8
	Tick_divisor_minus2                          uint8
	Du_cpb_removal_delay_increment_length_minus1 uint8
	Sub_pic_cpb_params_in_pic_timing_sei_flag    uint8
	Dpb_output_delay_du_length_minus1            uint8

	Bit_rate_scale    uint8
	Cpb_size_scale    uint8
	Cpb_size_du_scale uint8

	Initial_cpb_removal_delay_length_minus1 uint8
	Au_cpb_removal_delay_length_minus1      uint8
	Dpb_output_delay_length_minus1          uint8

	Fixed_pic_rate_general_flag     [HEVC_MAX_SUB_LAYERS]uint8
	Fixed_pic_rate_within_cvs_flag  [HEVC_MAX_SUB_LAYERS]uint8
	Elemental_duration_in_tc_minus1 [HEVC_MAX_SUB_LAYERS]uint16
	Low_delay_hrd_flag              [HEVC_MAX_SUB_LAYERS]uint8
	Cpb_cnt_minus1                  [HEVC_MAX_SUB_LAYERS]uint8
	Nal_sub_layer_hrd_parameters    [HEVC_MAX_SUB_LAYERS]H265RawSubLayerHRDParameters
	Vcl_sub_layer_hrd_parameters    [HEVC_MAX_SUB_LAYERS]H265RawSubLayerHRDParameters
}

func (hrd *H265RawHRDParameters) decode(r *bits.Reader,
	common_inf_present_flag bool, max_num_sub_layers_minus1 int) (err error) {
	if common_inf_present_flag {
		hrd.Nal_hrd_parameters_present_flag = r.ReadBit()
		hrd.Vcl_hrd_parameters_present_flag = r.ReadBit()

		if hrd.Nal_hrd_parameters_present_flag == 1 ||
			hrd.Vcl_hrd_parameters_present_flag == 1 {
			hrd.Sub_pic_hrd_params_present_flag = r.ReadBit()
			if hrd.Sub_pic_hrd_params_present_flag == 1 {
				hrd.Tick_divisor_minus2 = r.ReadUint8(8)
				hrd.Du_cpb_removal_delay_increment_length_minus1 = r.ReadUint8(5)
				hrd.Sub_pic_cpb_params_in_pic_timing_sei_flag = r.ReadBit()
				hrd.Dpb_output_delay_du_length_minus1 = r.ReadUint8(5)
			}

			hrd.Bit_rate_scale = r.ReadUint8(4)
			hrd.Cpb_size_scale = r.ReadUint8(4)
			if hrd.Sub_pic_hrd_params_present_flag == 1 {
				hrd.Cpb_size_du_scale = r.ReadUint8(4)

			}

			hrd.Initial_cpb_removal_delay_length_minus1 = r.ReadUint8(5)
			hrd.Au_cpb_removal_delay_length_minus1 = r.ReadUint8(5)
			hrd.Dpb_output_delay_length_minus1 = r.ReadUint8(5)
		} else {
			hrd.Sub_pic_hrd_params_present_flag = 0

			hrd.Initial_cpb_removal_delay_length_minus1 = 23
			hrd.Au_cpb_removal_delay_length_minus1 = 23
			hrd.Dpb_output_delay_length_minus1 = 23
		}
	}

	for i := 0; i <= max_num_sub_layers_minus1; i++ {
		hrd.Fixed_pic_rate_general_flag[i] = r.ReadBit()

		hrd.Fixed_pic_rate_within_cvs_flag[i] = 1
		if hrd.Fixed_pic_rate_general_flag[i] == 0 {
			hrd.Fixed_pic_rate_within_cvs_flag[i] = r.ReadBit()
		}

		if hrd.Fixed_pic_rate_within_cvs_flag[i] == 1 {
			hrd.Elemental_duration_in_tc_minus1[i] = r.ReadUe16()
			hrd.Low_delay_hrd_flag[i] = 0
		} else {
			hrd.Low_delay_hrd_flag[i] = r.ReadBit()
		}

		hrd.Cpb_cnt_minus1[i] = 0
		if hrd.Low_delay_hrd_flag[i] == 0 {
			hrd.Cpb_cnt_minus1[i] = r.ReadUe8()
		}

		if hrd.Nal_hrd_parameters_present_flag == 1 {
			hrd.Nal_sub_layer_hrd_parameters[i].decode(r, hrd.Sub_pic_hrd_params_present_flag == 1, int(hrd.Cpb_cnt_minus1[i]))
		}
		if hrd.Vcl_hrd_parameters_present_flag == 1 {
			hrd.Vcl_sub_layer_hrd_parameters[i].decode(r, hrd.Sub_pic_hrd_params_present_flag == 1, int(hrd.Cpb_cnt_minus1[i]))
		}
	}

	return
}

type H265RawVPS struct {
	Nal_unit_header H265RawNALUnitHeader

	Vps_video_parameter_set_id uint8

	Vps_base_layer_internal_flag  uint8
	Vps_base_layer_available_flag uint8
	Vps_max_layers_minus1         uint8
	Vps_max_sub_layers_minus1     uint8
	Vps_temporal_id_nesting_flag  uint8

	Profile_tier_level H265RawProfileTierLevel

	Vps_sub_layer_ordering_info_present_flag uint8
	Vps_max_dec_pic_buffering_minus1         [HEVC_MAX_SUB_LAYERS]uint8
	Vps_max_num_reorder_pics                 [HEVC_MAX_SUB_LAYERS]uint8
	Vps_max_latency_increase_plus1           [HEVC_MAX_SUB_LAYERS]uint32

	Vps_max_layer_id          uint8
	Vps_num_layer_sets_minus1 uint16
	Layer_id_included_flag    [][HEVC_MAX_LAYERS]uint8 //[HEVC_MAX_LAYER_SETS][HEVC_MAX_LAYERS]uint8

	Vps_timing_info_present_flag        uint8
	Vps_num_units_in_tick               uint32
	Vps_time_scale                      uint32
	Vps_poc_proportional_to_timing_flag uint8
	Vps_num_ticks_poc_diff_one_minus1   uint32
	Vps_num_hrd_parameters              uint16
	Hrd_layer_set_idx                   []uint16               //[HEVC_MAX_LAYER_SETS]uint16
	Cprms_present_flag                  []uint8                //[HEVC_MAX_LAYER_SETS]uint8
	Hrd_parameters                      []H265RawHRDParameters //[HEVC_MAX_LAYER_SETS]H265RawHRDParameters

	Vps_extension_flag uint8
	// extension_data     H265RawExtensionData
}

// DecodeString 从 base64 字串解码 vps NAL
func (vps *H265RawVPS) DecodeString(b64 string) error {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return vps.Decode(data)
}

// Decode 从字节序列中解码 vps NAL
func (vps *H265RawVPS) Decode(data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("RawVPS decode panic；r = %v \n %s", r, debug.Stack())
		}
	}()

	vpsWEB := utils.RemoveH264or5EmulationBytes(data)
	if len(vpsWEB) < 4 {
		return errors.New("The data is not enough")
	}

	r := bits.NewReader(vpsWEB)
	if err = vps.Nal_unit_header.decode(r); err != nil {
		return
	}

	if vps.Nal_unit_header.Nal_unit_type != NalVps {
		return errors.New("not is vps NAL UNIT")
	}

	vps.Vps_video_parameter_set_id = r.ReadUint8(4)

	vps.Vps_base_layer_internal_flag = r.ReadBit()
	vps.Vps_base_layer_available_flag = r.ReadBit()
	vps.Vps_max_layers_minus1 = r.ReadUint8(6)
	vps.Vps_max_sub_layers_minus1 = r.ReadUint8(3)
	vps.Vps_temporal_id_nesting_flag = r.ReadBit()

	if vps.Vps_max_sub_layers_minus1 == 0 &&
		vps.Vps_temporal_id_nesting_flag != 1 {
		return errors.New("Invalid stream: vps_temporal_id_nesting_flag must be 1 if vps_max_sub_layers_minus1 is 0.\n")
	}

	r.Skip(16) // vps_reserved_0xffff_16bits
	if err = vps.Profile_tier_level.decode(r, true, int(vps.Vps_max_sub_layers_minus1)); err != nil {
		return
	}

	vps.Vps_sub_layer_ordering_info_present_flag = r.ReadBit()
	i := vps.Vps_max_sub_layers_minus1
	if vps.Vps_sub_layer_ordering_info_present_flag == 1 {
		i = 0
	}
	for ; i <= vps.Vps_max_sub_layers_minus1; i++ {
		vps.Vps_max_dec_pic_buffering_minus1[i] = r.ReadUe8()
		vps.Vps_max_num_reorder_pics[i] = r.ReadUe8()
		vps.Vps_max_latency_increase_plus1[i] = r.ReadUe()
	}
	if vps.Vps_sub_layer_ordering_info_present_flag == 0 {
		for i := uint8(0); i < vps.Vps_max_sub_layers_minus1; i++ {
			vps.Vps_max_dec_pic_buffering_minus1[i] =
				vps.Vps_max_dec_pic_buffering_minus1[vps.Vps_max_sub_layers_minus1]
			vps.Vps_max_num_reorder_pics[i] =
				vps.Vps_max_num_reorder_pics[vps.Vps_max_sub_layers_minus1]
			vps.Vps_max_latency_increase_plus1[i] =
				vps.Vps_max_latency_increase_plus1[vps.Vps_max_sub_layers_minus1]
		}
	}

	vps.Vps_max_layer_id = r.ReadUint8(6)
	vps.Vps_num_layer_sets_minus1 = r.ReadUe16()
	vps.Layer_id_included_flag = make([][HEVC_MAX_LAYERS]uint8, vps.Vps_num_layer_sets_minus1+1)
	for i := uint16(1); i <= vps.Vps_num_layer_sets_minus1; i++ {
		for j := uint8(0); j <= vps.Vps_max_layer_id; j++ {
			vps.Layer_id_included_flag[i][j] = r.ReadBit()
		}
	}
	for j := uint8(0); j <= vps.Vps_max_layer_id; j++ {
		vps.Layer_id_included_flag[0][j] = 1
		if j == 0 {
			vps.Layer_id_included_flag[0][j] = 0
		}
	}
	vps.Vps_timing_info_present_flag = r.ReadBit()
	if vps.Vps_timing_info_present_flag == 1 {
		vps.Vps_num_units_in_tick = r.ReadUint32(32)
		vps.Vps_time_scale = r.ReadUint32(32)
		vps.Vps_poc_proportional_to_timing_flag = r.ReadBit()
		if vps.Vps_poc_proportional_to_timing_flag == 1 {
			vps.Vps_num_ticks_poc_diff_one_minus1 = r.ReadUe()
		}

		vps.Vps_num_hrd_parameters = r.ReadUe16()
		if vps.Vps_num_hrd_parameters > 0 {
			vps.Hrd_layer_set_idx = make([]uint16, vps.Vps_num_hrd_parameters)
			vps.Cprms_present_flag = make([]uint8, vps.Vps_num_hrd_parameters)
			vps.Hrd_parameters = make([]H265RawHRDParameters, vps.Vps_num_hrd_parameters)
		}
		for i := uint16(0); i < vps.Vps_num_hrd_parameters; i++ {
			vps.Hrd_layer_set_idx[i] = r.ReadUe16()
			if i > 0 {
				vps.Cprms_present_flag[i] = r.ReadBit()
			} else {
				vps.Cprms_present_flag[0] = 1
			}
			if err = vps.Hrd_parameters[i].decode(r,
				vps.Cprms_present_flag[i] == 1,
				int(vps.Vps_max_sub_layers_minus1)); err != nil {
				return
			}
		}
	}

	vps.Vps_extension_flag = r.ReadBit()
	return
}
