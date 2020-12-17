// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

// Amf0Type 类型常量
const (
	Amf0TypeNumber            = 0x00
	Amf0TypeBoolean           = 0x01
	Amf0TypeString            = 0x02
	Amf0TypeObject            = 0x03
	Amf0TypeMovieClip         = 0x04 //reserved, not supported
	AMF0TypeNull              = 0x05
	Amf0TypeUndefined         = 0x06
	Amf0TypeReference         = 0x07
	Amf0TypeEcmaArray         = 0x08
	Amf0TypeObjectEnd         = 0x09
	Amf0TypeStrictArray       = 0x0A
	Amf0TypeDate              = 0x0B
	Amf0TypeLongString        = 0x0C
)

// Amf0 数据名称常量
const (
	// TagAmfNData 关联的数据
	Amf0DataOnMetaData   = "onMetaData"
	Amf0DataOnCustomData = "onCustomData"
)
