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
