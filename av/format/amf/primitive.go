// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package amf

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// ReadBool .
func ReadBool(r io.Reader) (value bool, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	marker := buff[0]
	if TypeBoolean != marker {
		err = fmt.Errorf("Amf0ReadBool: TypeBoolean != marker")
		return
	}

	return readBool(r)
}

func readBool(r io.Reader) (value bool, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	if buff[0] != 0 {
		value = true
	} else {
		value = false
	}

	return
}

// WriteBool .
func WriteBool(w io.Writer, value bool) (err error) {
	var buff [2]byte

	buff[0] = TypeBoolean
	if value {
		buff[1] = 1
	} else {
		buff[1] = 0
	}
	_, err = w.Write(buff[:])
	return
}

// ReadNumber .
func ReadNumber(r io.Reader) (value float64, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	marker := buff[0]
	if TypeNumber != marker {
		err = fmt.Errorf("Amf0ReadNumber: TypeNumber != marker")
		return
	}

	return readNumber(r)
}

func readNumber(r io.Reader) (value float64, err error) {
	var buff [8]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	valueTmp := binary.BigEndian.Uint64(buff[:])
	value = math.Float64frombits(valueTmp)

	return
}

// WriteNumber .
func WriteNumber(w io.Writer, value float64) (err error) {
	var buff [9]byte

	buff[0] = TypeNumber
	v2 := math.Float64bits(value)
	binary.BigEndian.PutUint64(buff[1:], v2)

	_, err = w.Write(buff[:])
	return
}

// ReadDate .
func ReadDate(r io.Reader) (value time.Time, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	if TypeDate != buff[0] {
		err = fmt.Errorf("Amf0ReadDate: TypeDate != marker")
		return
	}

	return readDate(r)
}

func readDate(r io.Reader) (value time.Time, err error) {
	var buff [10]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}

	valueTmp := binary.BigEndian.Uint64(buff[:8])
	nano := math.Float64frombits(valueTmp) * float64(time.Millisecond)
	value = time.Unix(0, int64(nano))
	return
}

// WriteDate .
func WriteDate(w io.Writer, value time.Time) (err error) {
	var buff [11]byte // 1+8+2
	buff[0] = TypeDate
	nano := value.UnixNano()
	v2 := math.Float64bits(float64(nano) / float64(time.Millisecond))
	binary.BigEndian.PutUint64(buff[1:], v2)
	_, err = w.Write(buff[:])
	return
}

// ReadString .
func ReadString(r io.Reader) (value string, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}
	if TypeString != buff[0] {
		err = fmt.Errorf("Amf0ReadString: TypeString != marker")
		return
	}
	return readUtf8(r, 2)
}

// WriteString .
func WriteString(w io.Writer, value string) (err error) {
	var buff [1]byte
	buff[0] = TypeString
	if _, err = w.Write(buff[:]); err != nil {
		return
	}

	return writeUtf8(w, value, 2)
}

// ReadLongString .
func ReadLongString(r io.Reader) (value string, err error) {
	var buff [1]byte
	if _, err = io.ReadFull(r, buff[:]); err != nil {
		return
	}
	if TypeLongString != buff[0] {
		err = fmt.Errorf("Amf0ReadLongString: TypeLongString != marker")
		return
	}
	return readUtf8(r, 4)
}

// WriteLongString .
func WriteLongString(w io.Writer, value string) (err error) {
	var buff [1]byte
	buff[0] = TypeLongString
	if _, err = w.Write(buff[:]); err != nil {
		return
	}

	return writeUtf8(w, value, 4)
}

func writeType(w io.Writer, typ byte) (err error) {
	var buff [1]byte
	buff[0] = typ
	_, err = w.Write(buff[:])
	return
}

func readUtf8(r io.Reader, lenSize byte) (value string, err error) {
	var buff [4]byte
	if _, err = io.ReadFull(r, buff[4-lenSize:]); err != nil {
		return
	}

	strLen := binary.BigEndian.Uint32(buff[:])
	if 0 == strLen {
		return
	}

	valueBytes := make([]byte, strLen)
	if _, err = io.ReadFull(r, valueBytes); err != nil {
		return
	}
	value = string(valueBytes)

	return
}

func writeUtf8(w io.Writer, value string, lenSize byte) (err error) {
	var buff [4]byte
	binary.BigEndian.PutUint32(buff[:], uint32(len(value)))
	if _, err = w.Write(buff[4-lenSize:]); err != nil {
		return
	}

	if ws, ok := w.(io.StringWriter); ok {
		_, err = ws.WriteString(value)
	} else {
		_, err = w.Write([]byte(value))
	}
	return
}
