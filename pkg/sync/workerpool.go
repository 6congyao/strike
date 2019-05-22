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

package sync

import (
	"errors"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DEFAULT_CONCURRENCY         = 256 * 1024
	DEFAULT_PURGE_INTERVAL_TIME = 1
	CLOSED                      = 1
)

var (
	// ErrPoolClosed will be returned when submitting task to a closed pool.
	ErrPoolClosed = errors.New("this pool has been closed")

	workerChanCap = func() int {
		// Use blocking workerChan if GOMAXPROCS=1.
		// This immediately switches Serve to WorkerFunc, which results
		// in higher performance (under go1.5 at least).
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}

		// Use non-blocking workerChan if GOMAXPROCS>1,
		// since otherwise the Serve caller (Acceptor) may lag accepting
		// new tasks if WorkerFunc is CPU-bound.
		return 1
	}()
)

type workerPool struct {
	capacity int32
	running  int32
	release  int32

	expiryDuration time.Duration

	lock sync.Mutex

	cond *sync.Cond
	once sync.Once

	workers     []*worker
	workerCache sync.Pool

	PanicHandler func(interface{})
}

// NewWorkerPool create a worker pool
func NewWorkerPool(size, expiry int) WorkerPool {
	if size <= 0 {
		size = DEFAULT_CONCURRENCY
	}
	if expiry <= 0 {
		expiry = DEFAULT_PURGE_INTERVAL_TIME
	}
	wp := &workerPool{
		capacity:       int32(size),
		expiryDuration: time.Duration(expiry) * time.Second,
	}
	wp.cond = sync.NewCond(&wp.lock)
	go wp.periodicallyPurge()
	return wp
}

func (wp *workerPool) periodicallyPurge() {
	heartbeat := time.NewTicker(wp.expiryDuration)
	defer heartbeat.Stop()

	for range heartbeat.C {
		if CLOSED == atomic.LoadInt32(&wp.release) {
			break
		}
		currentTime := time.Now()
		wp.lock.Lock()
		idleWorkers := wp.workers
		n := -1
		for i, w := range idleWorkers {
			if currentTime.Sub(w.recycleTime) <= wp.expiryDuration {
				break
			}
			n = i
			w.task <- nil
			idleWorkers[i] = nil
		}
		if n > -1 {
			if n >= len(idleWorkers)-1 {
				wp.workers = idleWorkers[:0]
			} else {
				wp.workers = idleWorkers[n+1:]
			}
		}
		wp.lock.Unlock()
	}
}

func (wp *workerPool) Serve(t func()) error {
	if CLOSED == atomic.LoadInt32(&wp.release) {
		return ErrPoolClosed
	}
	wp.retrieveWorker().task <- t

	return nil
}

func (wp *workerPool) Running() int {
	return int(atomic.LoadInt32(&wp.running))
}

func (wp *workerPool) Free() int {
	return int(atomic.LoadInt32(&wp.capacity) - atomic.LoadInt32(&wp.running))
}

func (wp *workerPool) Cap() int {
	return int(atomic.LoadInt32(&wp.capacity))
}

func (wp *workerPool) Tune(size int) {
	if size == wp.Cap() {
		return
	}
	atomic.StoreInt32(&wp.capacity, int32(size))
	diff := wp.Running() - size
	for i := 0; i < diff; i++ {
		wp.retrieveWorker().task <- nil
	}
}

func (wp *workerPool) Release() error {
	wp.once.Do(func() {
		atomic.StoreInt32(&wp.release, 1)
		wp.lock.Lock()
		idleWorkers := wp.workers
		for i, w := range idleWorkers {
			w.task <- nil
			idleWorkers[i] = nil
		}
		wp.workers = nil
		wp.lock.Unlock()
	})
	return nil
}

func (wp *workerPool) retrieveWorker() *worker {
	var w *worker

	wp.lock.Lock()
	idleWorkers := wp.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		w = idleWorkers[n]
		idleWorkers[n] = nil
		wp.workers = idleWorkers[:n]
		wp.lock.Unlock()
	} else if wp.Running() < wp.Cap() {
		wp.lock.Unlock()
		if cacheWorker := wp.workerCache.Get(); cacheWorker != nil {
			w = cacheWorker.(*worker)
		} else {
			w = &worker{
				pool: wp,
				task: make(chan func(), workerChanCap),
			}
		}
		w.run()
	} else {
		for {
			wp.cond.Wait()
			l := len(wp.workers) - 1
			if l < 0 {
				continue
			}
			w = wp.workers[l]
			wp.workers[l] = nil
			wp.workers = wp.workers[:l]
			break
		}
		wp.lock.Unlock()
	}
	return w
}

func (wp *workerPool) revertWorker(worker *worker) bool {
	if CLOSED == atomic.LoadInt32(&wp.release) {
		return false
	}
	worker.recycleTime = time.Now()
	wp.lock.Lock()
	wp.workers = append(wp.workers, worker)
	// Notify the invoker stuck in 'retrieveWorker()' of there is an available worker in the worker queue.
	wp.cond.Signal()
	wp.lock.Unlock()
	return true
}

func (wp *workerPool) incRunning() {
	atomic.AddInt32(&wp.running, 1)
}

func (wp *workerPool) decRunning() {
	atomic.AddInt32(&wp.running, -1)
}

type worker struct {
	pool        *workerPool
	task        chan func()
	recycleTime time.Time
}

func (w *worker) run() {
	w.pool.incRunning()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				w.pool.decRunning()
				w.pool.workerCache.Put(w)
				if w.pool.PanicHandler != nil {
					w.pool.PanicHandler(p)
				} else {
					log.Printf("worker exits from a panic: %v", p)
				}
			}
		}()

		for f := range w.task {
			if nil == f {
				w.pool.decRunning()
				w.pool.workerCache.Put(w)
				return
			}
			f()
			if ok := w.pool.revertWorker(w); !ok {
				break
			}
		}
	}()
}
