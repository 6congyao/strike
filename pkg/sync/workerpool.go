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
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

const (
	maxRespwanTimes = 1 << 6
)

type shard struct {
	sync.Mutex
	index        int
	respawnTimes uint32
	jobChan      chan interface{}
	jobQueue     []interface{}
}

type shardWorkerPool struct {
	// workerFunc should never exit, always try to acquire jobs from jobs channel
	workerFunc WorkerFunc
	shards     []*shard
	numShards  int
	// represents whether job scheduler for queued jobs is started or not
	schedule uint32
}

// NewShardWorkerPool creates a new shard worker pool.
func NewShardWorkerPool(size int, numShards int, workerFunc WorkerFunc) (ShardWorkerPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("worker pool size too small: %d", size)
	}
	if size < numShards {
		numShards = size
	}
	shardCap := size / numShards
	shards := make([]*shard, numShards)
	for i := range shards {
		shards[i] = &shard{
			index:   i,
			jobChan: make(chan interface{}, shardCap),
		}
	}
	return &shardWorkerPool{
		workerFunc: workerFunc,
		shards:     shards,
		numShards:  numShards,
	}, nil
}

func (pool *shardWorkerPool) Init() {
	for i := range pool.shards {
		pool.spawnWorker(pool.shards[i])
	}
}

func (pool *shardWorkerPool) Shard(source uint32) uint32 {
	return source % uint32(pool.numShards)
}

func (pool *shardWorkerPool) Offer(job ShardJob) {
	// use shard to avoid excessive synchronization
	i := pool.Shard(job.Source())
	shard := pool.shards[i]
	// put jobs to the jobChan or jobQueue, which determined by the shard workload
	shard.Lock()
	if len(shard.jobQueue) == 0 && cap(shard.jobChan) > len(shard.jobChan) {
		shard.jobChan <- job
	} else {
		shard.jobQueue = append(shard.jobQueue, job)
		// schedule flush if
		if atomic.CompareAndSwapUint32(&pool.schedule, 0, 1) {
			pool.flush()
		}
	}

	shard.Unlock()
}

func (pool *shardWorkerPool) spawnWorker(shard *shard) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Println("worker panic:", p)
				debug.PrintStack()
				//try respawn worker
				if shard.respawnTimes < maxRespwanTimes {
					shard.respawnTimes++
					pool.spawnWorker(shard)
				}
			}
		}()
		pool.workerFunc(shard.index, shard.jobChan)
	}()
}

func (pool *shardWorkerPool) flush() {
	go func() {
		for {
			clear := true
			for i := range pool.shards {
				shard := pool.shards[i]
				shard.Lock()
				pending := len(shard.jobQueue)
				slots := cap(shard.jobChan) - len(shard.jobChan)
				if clear && pending > 0 {
					clear = false
				}
				// the number we can write is determined by the minimal of pending jobs and available chan data slots
				writable := min(pending, slots)
				if writable > 0 {
					log.Println("flush job to shard:", writable, i)
					for j := 0; j < writable; j++ {
						shard.jobChan <- shard.jobQueue[j]
					}
					shard.jobQueue = shard.jobQueue[writable:]
				}
				shard.Unlock()
			}
			// all shards' job queue are clear, stop flush goroutine
			if clear {
				break
			}
			// wait for next schedule
			runtime.Gosched()
		}
		// end flush schedule
		atomic.CompareAndSwapUint32(&pool.schedule, 1, 0)
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type workerPool struct {
	work chan func()
	sem  chan struct{}
}

// NewWorkerPool create a worker pool
func NewWorkerPool(size int) WorkerPool {
	return &workerPool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
}

func (p *workerPool) Schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	}
}

func (p *workerPool) ScheduleAlways(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	default:
		// new temp goroutine for task execution
		go task()
	}
}

func (p *workerPool) spawnWorker(task func()) {
	defer func() { <-p.sem }()
	for {
		task()
		task = <-p.work
	}
}
