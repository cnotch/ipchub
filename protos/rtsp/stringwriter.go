// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import "io"

type writeStringer interface {
	WriteString(s string) (n int, err error)
}

type stringWriter struct {
	w io.Writer
}

func (sw stringWriter) WriteString(s string) (n int, err error) {
	return sw.w.Write([]byte(s))
}
