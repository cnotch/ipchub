// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

type memProvider struct {
}

func (p *memProvider) Name() string {
	return "memory"
}

func (p *memProvider) Configure(config map[string]interface{}) error {
	return nil
}

func (p *memProvider) LoadAll() ([]*User, error) {
	return []*User{{
		Name:     "admin",
		Password: "admin",
		Admin:    true,
	}}, nil
}

func (p *memProvider) Flush(full []*User, saves []*User, removes []*User) error {
	return nil
}
