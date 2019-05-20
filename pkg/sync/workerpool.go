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
	"runtime/debug"
)

const (
	maxRespawnTimes = 1 << 6
)

type shard struct {
	index        int
	respawnTimes uint32
	jobChan      chan interface{}
}

type shardWorkerPool struct {
	// workerFunc should never exit, always try to acquire jobs from jobs channel
	workerFunc WorkerFunc
	shards     []*shard
	numShards  int
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

func (pool *shardWorkerPool) Shard(source, numShards, offset uint32) uint32 {
	if numShards == 0 {
		return 0
	}
	return source%numShards + offset
}

func (pool *shardWorkerPool) Offer(job ShardJob, block bool) {
	// use shard to avoid excessive synchronization
	i := pool.Shard(job.Source(uint32(pool.numShards)))

	if block {
		pool.shards[i].jobChan <- job
	} else {
		select {
		case pool.shards[i].jobChan <- job:
		default:
			log.Println("jobChan over full")
		}
	}
}

func (pool *shardWorkerPool) spawnWorker(shard *shard) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Println("worker panic", p)
				debug.PrintStack()
				//try respawn worker
				if shard.respawnTimes < maxRespawnTimes {
					shard.respawnTimes++
					pool.spawnWorker(shard)
				}
			}
		}()
		pool.workerFunc(shard.index, shard.jobChan)
	}()
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
