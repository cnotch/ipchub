// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package amf

import (
	"fmt"
	"io"
	"reflect"
	"time"
)

// Amf0 类型常量
const (
	TypeNumber            = 0x00
	TypeBoolean           = 0x01
	TypeString            = 0x02
	TypeObject            = 0x03
	TypeMovieClip         = 0x04 //reserved, not supported
	TypeNull              = 0x05
	TypeUndefined         = 0x06
	TypeReference         = 0x07
	TypeEcmaArray         = 0x08
	TypeObjectEnd         = 0x09
	TypeStrictArray       = 0x0A
	TypeDate              = 0x0B
	TypeLongString        = 0x0C
	TypeUnSupported       = 0x0D
	TypeRecordSet         = 0x0E
	TypeXMLDocument       = 0x0F
	TypeTypedObject       = 0x10
	TypeAVMplusObject     = 0x11
	TypeOriginStrictArray = 0x20
	TypeInvalid           = 0x3F
)

// UndefinedValue  表示 TypeUndefined 类型存储的值
type UndefinedValue struct{}

// UnSupportedValue 表示 TypeUnSupported 类型存储的值
type UnSupportedValue struct{}

// objectEndValue 表示 TypeObjectEnd 存储的值
type objectEndValue struct{}

// ReadAny .
func ReadAny(r io.Reader) (value interface{}, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	marker := buff[0]
	switch marker {
	case TypeNumber:
		value, err = readNumber(r)
	case TypeBoolean:
		value, err = readBool(r)
	case TypeString:
		value, err = readUtf8(r, 2)
	case TypeObject:
		value, err = readObject(r)
	case TypeNull:
		value = nil
	case TypeUndefined:
		value = UndefinedValue{}
	case TypeEcmaArray:
		value, err = readEcmaArray(r)
	case TypeObjectEnd:
		value = objectEndValue{}
	case TypeStrictArray:
		value, err = readStrictArray(r)
	case TypeDate:
		value, err = readDate(r)
	case TypeLongString:
		value, err = readUtf8(r, 4)
	case TypeUnSupported:
		value = UnSupportedValue{}
	default:
		err = fmt.Errorf("Amf0ReadAny: unsupported marker - %d", marker)
	}
	return
}

// WriteAny .
func WriteAny(w io.Writer, any interface{}) (err error) {
	if any == nil {
		err = writeType(w, TypeNull)
		return
	}

	switch v := any.(type) {
	case *string:
		if len(*v) > 65535 {
			err = WriteLongString(w, *v)
		} else {
			err = WriteString(w, *v)
		}
	case string:
		if len(v) > 65535 {
			err = WriteLongString(w, v)
		} else {
			err = WriteString(w, v)
		}
	case *bool:
		err = WriteBool(w, *v)
	case bool:
		err = WriteBool(w, v)
	case *int:
		err = WriteNumber(w, float64(*v))
	case int:
		err = WriteNumber(w, float64(v))
	case *int8:
		err = WriteNumber(w, float64(*v))
	case int8:
		err = WriteNumber(w, float64(v))
	case *int16:
		err = WriteNumber(w, float64(*v))
	case int16:
		err = WriteNumber(w, float64(v))
	case *int32:
		err = WriteNumber(w, float64(*v))
	case int32:
		err = WriteNumber(w, float64(v))
	case *int64:
		err = WriteNumber(w, float64(*v))
	case int64:
		err = WriteNumber(w, float64(v))
	case *uint:
		err = WriteNumber(w, float64(*v))
	case uint:
		err = WriteNumber(w, float64(v))
	case *uint8:
		err = WriteNumber(w, float64(*v))
	case uint8:
		err = WriteNumber(w, float64(v))
	case *uint16:
		err = WriteNumber(w, float64(*v))
	case uint16:
		err = WriteNumber(w, float64(v))
	case *uint32:
		err = WriteNumber(w, float64(*v))
	case uint32:
		err = WriteNumber(w, float64(v))
	case *uint64:
		err = WriteNumber(w, float64(*v))
	case uint64:
		err = WriteNumber(w, float64(v))
	case *float32:
		err = WriteNumber(w, float64(*v))
	case float32:
		err = WriteNumber(w, float64(v))
	case *float64:
		err = WriteNumber(w, *v)
	case float64:
		err = WriteNumber(w, v)
	case *time.Time:
		err = WriteDate(w, *v)
	case time.Time:
		err = WriteDate(w, v)
	case *UndefinedValue:
		err = writeType(w, TypeUndefined)
	case UndefinedValue:
		err = writeType(w, TypeUndefined)
	case *EcmaArray:
		err = WriteEcmaArray(w, *v)
	case EcmaArray:
		err = WriteEcmaArray(w, v)
	case *Object:
		err = WriteObject(w, *v)
	case Object:
		err = WriteObject(w, v)
	case *StrictArray:
		err = WriteStrictArray(w, *v)
	case StrictArray:
		err = WriteStrictArray(w, v)
	default:
		err = fmt.Errorf("Unsupported type : %v", reflect.TypeOf(v))
	}

	return
}
