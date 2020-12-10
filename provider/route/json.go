// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package route

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cnotch/tomatox/utils"
)

// JSON json 提供者
var JSON = &jsonProvider{}

type jsonProvider struct {
	filePath string
}

func (p *jsonProvider) Name() string {
	return "json"
}

func (p *jsonProvider) Configure(config map[string]interface{}) error {
	path, ok := config["file"]
	if ok {
		switch v := path.(type) {
		case string:
			p.filePath = v
		default:
			return fmt.Errorf("invalid route table config, file attr: %v", path)
		}
	} else {
		p.filePath = "routetable.json"
	}

	if !filepath.IsAbs(p.filePath) {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		p.filePath = filepath.Join(filepath.Dir(exe), p.filePath)
	}

	return nil
}

func (p *jsonProvider) LoadAll() ([]*Route, error) {
	path := p.filePath
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// 从文件读
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var routes []*Route
	if err := json.Unmarshal(b, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

func (p *jsonProvider) Flush(full []*Route, saves []*Route, removes []*Route) error {
	return utils.EncodeJSONFile(p.filePath, full)
}
