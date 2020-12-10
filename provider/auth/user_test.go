// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import "testing"

func TestUser_ValidePermission(t *testing.T) {
	u := User{
		Name:       "cao",
		Password:   "ok",
		PushAccess: "/a/+/c",
		PullAccess: "/a/*",
	}
	u.init()

	tests := []struct {
		name  string
		path  string
		right AccessRight
		want  bool
	}{
		{"2", "/a/b/c", PushRight, true},
		{"3", "/a/c", PushRight, false},
		{"4", "/a", PullRight, true},
		{"5", "/a/c", PullRight, true},
		{"6", "/a/c/d", PullRight, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := u.ValidatePermission(tt.path, tt.right); got != tt.want {
				t.Errorf("User.ValidePermission() = %v, want %v", got, tt.want)
			}
		})
	}
}
