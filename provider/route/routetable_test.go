// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package route

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_routetable(t *testing.T) {

	t.Run("routetable", func(t *testing.T) {
		Save(&Route{"/test/live1", "rtsp://localhost:5540/live1", false})
		assert.Equal(t, 1, len(globalT.l))
		r := Get("/test/live1")
		assert.NotNil(t, r)
		Save(&Route{"/easy/", "rtsp://localhost:5540/test", false})
		assert.Equal(t, 2, len(globalT.l))
		r = Match("/easy/live4")
		assert.NotNil(t, r)
		assert.Equal(t, "rtsp://localhost:5540/test/live4", r.URL)
		Del("/test/live1")
		Save(&Route{"/test/live1", "rtsp://localhost:5540/live1", false})
		Save(&Route{"/test/live1", "rtsp://localhost:5540/live1", false})
		Del("/test/live1")
		Save(&Route{"/test/live1", "rtsp://localhost:5540/live1", false})
		assert.Equal(t, 2, len(globalT.saves))
		assert.Equal(t, 0, len(globalT.removes))
		Flush()
		assert.Equal(t, 0, len(globalT.saves))
	})

}
