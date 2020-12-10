// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package stats

import (
	"runtime"
	"time"

	"github.com/kelindar/process"
)

// 创建时间
var (
	StartingTime = time.Now()
)

// Runtime 运行时统计
type Runtime struct {
	Heap   Heap   `json:"heap"`
	MCache Memory `json:"mcache"` // MemStats.MCacheInuse/MCacheSys
	MSpan  Memory `json:"mspan"`  // MemStats.MSpanInuse/MSpanSys
	Stack  Memory `json:"stack"`  // MemStats.StackInuse/StackSys
	GC     GC     `json:"gc"`
	Go     Go     `json:"go"`
}

// Proc 进程信息统计
type Proc struct {
	CPU    float64 `json:"cpu"`    // cpu使用情况
	Priv   int32   `json:"priv"`   // 私有内存 KB
	Virt   int32   `json:"virt"`   // 虚拟内存 KB
	Uptime int32   `json:"uptime"` // 运行时间 S
}

// Heap 运行是堆信息
type Heap struct {
	Inuse    int32 `json:"inuse"`    // KB MemStats.HeapInuse
	Sys      int32 `json:"sys"`      // KB MemStats.HeapSys
	Alloc    int32 `json:"alloc"`    // KB MemStats.HeapAlloc
	Idle     int32 `json:"idle"`     // KB MemStats.HeapIdle
	Released int32 `json:"released"` // KB MemStats.HeapReleased
	Objects  int32 `json:"objects"`  // = MemStats.HeapObjects
}

// Memory 通用内存信息
type Memory struct {
	Inuse int32 `json:"inuse"` // KB
	Sys   int32 `json:"sys"`   // KB
}

// GC 垃圾回收信息
type GC struct {
	CPU float64 `json:"cpu"` // cpu使用情况
	Sys int32   `json:"sys"` // KB MemStats.GCSys
}

// Go Go运行时 goroutines 、threads 和 total memory
type Go struct {
	Count int32 `json:"count"` // runtime.NumGoroutine()
	Procs int32 `json:"procs"` //runtime.NumCPU()
	Sys   int32 `json:"sys"`   // KB MemStats.Sys
	Alloc int32 `json:"alloc"` // KB MemStats.TotalAlloc
}

// MeasureRuntime 获取运行时信息。
func MeasureRuntime() Proc {
	defer recover()
	var memoryPriv, memoryVirtual int64
	var cpu float64
	process.ProcUsage(&cpu, &memoryPriv, &memoryVirtual)
	return Proc{
		CPU:    cpu,
		Priv:   toKB(uint64(memoryPriv)),
		Virt:   toKB(uint64(memoryVirtual)),
		Uptime: int32(time.Now().Sub(StartingTime).Seconds()),
	}
}

// MeasureFullRuntime 获取运行时信息。
func MeasureFullRuntime() *Runtime {
	defer recover()

	// Collect stats
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	return &Runtime{
		// Measure heap information
		Heap: Heap{
			Alloc:    toKB(memory.HeapAlloc),
			Idle:     toKB(memory.HeapIdle),
			Inuse:    toKB(memory.HeapInuse),
			Objects:  int32(memory.HeapObjects),
			Released: toKB(memory.HeapReleased),
			Sys:      toKB(memory.HeapSys),
		},
		// Measure off heap memory
		MCache: Memory{
			Inuse: toKB(memory.MCacheInuse),
			Sys:   toKB(memory.MCacheSys),
		},
		MSpan: Memory{
			Inuse: toKB(memory.MSpanInuse),
			Sys:   toKB(memory.MSpanSys),
		},
		// Measure memory
		Stack: Memory{
			Inuse: toKB(memory.StackInuse),
			Sys:   toKB(memory.StackSys),
		},
		// Measure GC
		GC: GC{
			CPU: memory.GCCPUFraction,
			Sys: toKB(memory.GCSys),
		},
		// Measure goroutines and threads and total memory
		Go: Go{
			Count: int32(runtime.NumGoroutine()),
			Procs: int32(runtime.NumCPU()),
			Sys:   toKB(memory.Sys),
			Alloc: toKB(memory.TotalAlloc),
		},
	}
}

// Converts the memory in bytes to KBs, otherwise it would overflow our int32
func toKB(v uint64) int32 {
	return int32(v / 1024)
}
