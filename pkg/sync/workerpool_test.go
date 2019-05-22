/*
 * Copyright (c) 2019. LuCongyao <6congyao@gmail.com> .
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this work except in compliance with the License.
 * You may obtain a copy of the License in the LICENSE file, or at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sync

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
	GiB // 1073741824
	TiB // 1099511627776             (超过了int32的范围)
	PiB // 1125899906842624
	EiB // 1152921504606846976
	ZiB // 1180591620717411303424    (超过了int64的范围)
	YiB // 1208925819614629174706176
)

const (
	Param    = 100
	Size     = 1000
	TestSize = 10000
	n        = 100000
)

var curMem uint64

func demoFunc() {
	n := 10
	time.Sleep(time.Duration(n) * time.Millisecond)
}

func demoPoolFunc(args interface{}) {
	n := args.(int)
	time.Sleep(time.Duration(n) * time.Millisecond)
}

func TestWorkerPoolWaitToGetWorker(t *testing.T) {
	var wg sync.WaitGroup
	p := NewWorkerPool(Size, 0)
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		p.Serve(func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()
	t.Logf("pool, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestWorkerPoolGetWorkerFromCache(t *testing.T) {
	p := NewWorkerPool(TestSize, 0)
	defer p.Release()

	for i := 0; i < Size; i++ {
		p.Serve(demoFunc)
	}
	time.Sleep(2 * DEFAULT_PURGE_INTERVAL_TIME * time.Second)
	p.Serve(demoFunc)
	t.Logf("pool, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestNoPool(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}

	wg.Wait()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestWorkerPool(t *testing.T) {
	p := NewWorkerPool(DEFAULT_CONCURRENCY, 0)
	defer p.Release()
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		p.Serve(func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()

	t.Logf("pool, capacity:%d", p.Cap())
	t.Logf("pool, running workers number:%d", p.Running())
	t.Logf("pool, free workers number:%d", p.Free())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestPoolPanicWithoutHandler(t *testing.T) {
	p := NewWorkerPool(10, 0)
	defer p.Release()
	p.Serve(func() {
		panic("Oops!")
	})
}

func TestPurge(t *testing.T) {
	p := NewWorkerPool(10, 0)
	defer p.Release()

	p.Serve(demoFunc)
	time.Sleep(3 * DEFAULT_PURGE_INTERVAL_TIME * time.Second)
	if p.Running() != 0 {
		t.Error("all p should be purged")
	}
}
