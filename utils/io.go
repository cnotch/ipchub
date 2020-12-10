// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package utils

import (
	"bytes"
	"encoding/json"
	"os"
)

// EncodeJSONFile 编码 JSON 文件
func EncodeJSONFile(path string, obj interface{}) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	defer f.Close()

	var formatted bytes.Buffer
	body, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if err := json.Indent(&formatted, body, "", "\t"); err != nil {
		return err
	}

	if _, err := f.Write(formatted.Bytes()); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}
