/*
 * Copyright (c) 2018. LuCongyao <6congyao@gmail.com> .
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

package internal

import (
	"runtime"
	"sync/atomic"
)

const (
	PollEvent_None  = 0
	PollEvent_Read  = 1
	PollEvent_Write = 2
)

type AddConnection struct {
	FD int
}

// this is a good candiate for a lock-free structure.

type Spinlock struct{ lock uintptr }

func (l *Spinlock) Lock() {
	for !atomic.CompareAndSwapUintptr(&l.lock, 0, 1) {
		runtime.Gosched()
	}
}
func (l *Spinlock) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}

type noteQueue struct {
	mu    Spinlock
	notes []interface{}
	n     int64
}

func (q *noteQueue) Add(note interface{}) (one bool) {
	q.mu.Lock()
	n := atomic.AddInt64(&q.n, 1)
	q.notes = append(q.notes, note)
	q.mu.Unlock()
	return n == 1
}

func (q *noteQueue) ForEach(iter func(note interface{}) error) error {
	if atomic.LoadInt64(&q.n) == 0 {
		return nil
	}
	q.mu.Lock()
	if len(q.notes) == 0 {
		q.mu.Unlock()
		return nil
	}
	notes := q.notes
	atomic.StoreInt64(&q.n, 0)
	q.notes = nil
	q.mu.Unlock()
	for _, note := range notes {
		if err := iter(note); err != nil {
			return err
		}
	}
	return nil
}
