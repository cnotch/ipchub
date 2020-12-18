// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package amf

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ObjectProperty amf 对象属性
type ObjectProperty struct {
	Name  string
	Value interface{}
}

// PropertyValue  获取属性数组中指定属性名的值
func PropertyValue(properties []ObjectProperty, name string) (value interface{}, ok bool) {
	for _, prop := range properties {
		if name == prop.Name {
			return prop.Value, true
		}
	}
	return nil, false
}

// EcmaArray 表示 TypeEcmaArray 类型存储的值
type EcmaArray []ObjectProperty

// ReadEcmaArray .
func ReadEcmaArray(r io.Reader) (value EcmaArray, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	if TypeEcmaArray != buff[0] {
		err = fmt.Errorf("error: Amf0ReadEcmaArray: TypeEcmaArray != marker")
		return
	}
	return readEcmaArray(r)
}

func readEcmaArray(r io.Reader) (value EcmaArray, err error) {
	var buff [4]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	count := int(binary.BigEndian.Uint32(buff[:]))
	value = make(EcmaArray, 0, count)
	for {
		var elem ObjectProperty
		if elem.Name, err = readUtf8(r, 2); err != nil {
			return
		}

		if elem.Value, err = ReadAny(r); err != nil {
			return
		}

		// 判断是否是 ObjectEnd
		if _, ok := elem.Value.(objectEndValue); ok && len(elem.Name) == 0 {
			break
		}

		value = append(value, elem)
	}

	if len(value) != count {
		err = fmt.Errorf("Does not match the expected length of the array and the actual; expected = %d ,actual = %d", count, len(value))
	}
	return
}

// WriteEcmaArray .
func WriteEcmaArray(w io.Writer, arr EcmaArray) (err error) {
	var buff [5]byte

	buff[0] = TypeEcmaArray
	binary.BigEndian.PutUint32(buff[1:], uint32(len(arr)))
	if _, err = w.Write(buff[:]); err != nil {
		return
	}

	for _, elem := range arr {
		if err = writeUtf8(w, elem.Name, 2); err != nil {
			return
		}
		if err = WriteAny(w, elem.Value); err != nil {
			return
		}
	}

	//eof
	if _, err = w.Write([]byte{0x00, 0x00, TypeObjectEnd}); err != nil {
		return
	}
	return
}

// Object 表示 TypeObject 类型存储的值
type Object []ObjectProperty

// ReadObject .
func ReadObject(r io.Reader) (value Object, err error) {
	var data [1]byte
	if _, err = io.ReadFull(r, data[:]); err != nil {
		return
	}

	if TypeObject != data[0] {
		err = fmt.Errorf("error: Amf0ReadObject: TypeObject != marker")
		return
	}
	return readObject(r)
}

func readObject(r io.Reader) (value Object, err error) {
	for {
		var prop ObjectProperty
		if prop.Name, err = readUtf8(r, 2); err != nil {
			return
		}

		if prop.Value, err = ReadAny(r); err != nil {
			return
		}

		// 判断是否是 ObjectEnd
		if _, ok := prop.Value.(objectEndValue); ok && len(prop.Name) == 0 {
			break
		}
		value = append(value, prop)
	}

	return
}

// WriteObject .
func WriteObject(w io.Writer, obj Object) (err error) {
	buff := [1]byte{TypeObject}
	if _, err = w.Write(buff[:]); err != nil {
		return
	}

	for _, prop := range obj {
		if err = writeUtf8(w, prop.Name, 2); err != nil {
			return
		}
		if err = WriteAny(w, prop.Value); err != nil {
			return
		}
	}

	//eof
	if _, err = w.Write([]byte{0x00, 0x00, TypeObjectEnd}); err != nil {
		return
	}

	return
}

// StrictArray 表示 TypeStrictArray 类型存储的值
type StrictArray []interface{}

// ReadStrictArray .
func ReadStrictArray(r io.Reader) (value StrictArray, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	if TypeStrictArray != buff[0] {
		err = fmt.Errorf("error: Amf0ReadStrictArray: TypeStrictArray != marker")
		return
	}
	return readStrictArray(r)
}

func readStrictArray(r io.Reader) (value StrictArray, err error) {
	var buff [4]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	count := int(binary.BigEndian.Uint32(buff[:]))

	for i := 0; i < count; i++ {
		var elem interface{}
		elem, err = ReadAny(r)
		value = append(value, elem)
	}

	return
}

// WriteStrictArray .
func WriteStrictArray(w io.Writer, arr StrictArray) (err error) {
	var buff [5]byte

	buff[0] = TypeStrictArray
	binary.BigEndian.PutUint32(buff[1:], uint32(len(arr)))

	for _, elem := range arr {
		if err = WriteAny(w, elem); err != nil {
			return
		}
	}
	return
}
