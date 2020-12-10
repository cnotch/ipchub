// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import "testing"

func Test_alwaysMatcher_Match(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"always", "/a/b", true},
		{"always", "/a", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewPathMatcher(endWildcard)
			if got := m.Match(tt.path); got != tt.want {
				t.Errorf("alwaysMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pathMacher_Match(t *testing.T) {
	tests := []struct {
		name     string
		pathMask string
		path     string
		want     bool
	}{
		{"g1", "/a", "/a", true},
		{"g2", "/a", "/a/b", false},
		{"e1", "/a/*", "/a", true},
		{"e2", "/a/*", "/a/b", true},
		{"e3", "/a/*", "/a/b/c", true},
		{"e4", "/a/*", "/b", false},
		{"c1", "/a/+/c/*", "/a/b/c", true},
		{"c2", "/a/+/c/*", "/a/d/c", true},
		{"c3", "/a/+/c/*", "/a/b/c/d", true},
		{"c4", "/a/+/c/*", "/a/b/c/d/e", true},
		{"c5", "/a/+/c/*", "/a/c/d/e", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewPathMatcher(tt.pathMask)
			if got := m.Match(tt.path); got != tt.want {
				t.Errorf("pathMacher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
