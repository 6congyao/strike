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
	"log"
	"runtime/debug"
	"strike/utils/container/squeue"
)

const (
	maxRespawnTimes = 1 << 6
)

type shard struct {
	index        int
	respawnTimes uint32
	//jobChan      chan interface{}
	queue *squeue.Queue
}

type shardWorkerPool struct {
	// workerFunc should never exit, always try to acquire jobs from jobs channel
	workerFunc WorkerFunc
	shards     []*shard
	numShards  int
}

// NewShardWorkerPool creates a new shard worker pool.
func NewShardWorkerPool(numShards int, workerFunc WorkerFunc) (ShardWorkerPool, error) {
	//shardCap := size / numShards
	shards := make([]*shard, numShards)
	for i := range shards {
		shards[i] = &shard{
			index: i,
			queue: squeue.New(),
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

func (pool *shardWorkerPool) Shard(source, numShards, offset uint64) uint64 {
	if numShards == 0 {
		return 0
	}
	return source%numShards + offset
}

func (pool *shardWorkerPool) Offer(job ShardJob, block bool) {
	// use shard to avoid excessive synchronization
	i := pool.Shard(job.Source(uint64(pool.numShards)))

	if block {
		pool.shards[i].queue.C <- job
	} else {
		pool.shards[i].queue.Push(job)
		//select {
		//case pool.shards[i].jobChan <- job:
		//default:
		//	log.Println("jobChan over full:", i)
		//}
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
		pool.workerFunc(shard.index, shard.queue.C)
	}()
}
