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
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

type TestJob struct {
	i uint32
}

func (t *TestJob) Source(sourceShards uint32) (uint32, uint32) {
	return t.i, 0
}

type TestJob2 struct {
	i uint32
	dir uint32
}

func (t2 *TestJob2) Source2(sourceShards uint32) (source, targetShards uint32) {
	if t2.dir == 1 {
		source = t2.i
		targetShards = sourceShards - 1
	} else {
		source = sourceShards - 1
		targetShards = sourceShards
	}
	return
}

func TestSource2(t *testing.T) {
	shardsNum := 7
	shardEvents := 512
	//wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		//prev := 0
		//count := 0
		//
		//for job := range jobChan {
		//	if testJob, ok := job.(*TestJob); ok {
		//		if int(testJob.i) <= prev {
		//			t.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
		//			wg.Done()
		//			return
		//		}
		//
		//		prev = int(testJob.i)
		//		count++
		//
		//		if count >= shardEvents {
		//			wg.Done()
		//			return
		//		}
		//	}
		//}

	}
	pool, _ := NewShardWorkerPool(shardsNum*64, shardsNum, consumer)
	pool.Init()

	for i := 0; i < shardsNum; i++ {
			for j := 0; j < shardEvents; j++ {
				tj1 := &TestJob2{
					i: uint32(j),
					dir: 1,
				}
				index1 := pool.Shard(tj1.Source2(uint32(shardsNum)))
				if index1 >= uint32(shardsNum - 1) {
					t.Errorf("unexpected shard index, shard %d, shardNum %d", index1, shardsNum - 1)
				}

				tj2 := &TestJob2{
					i: uint32(j),
					dir: 2,
				}
				index2 := pool.Shard(tj2.Source2(uint32(shardsNum)))
				if index2 != uint32(shardsNum - 1) {
					t.Errorf("unexpected shard index, shard %d, shardNum %d", index2, shardsNum - 1)
				}
			}

	}

	for i := 0; i < shardEvents; i++ {
	}
}

// TestJobOrder test worker pool's event dispatch functionality, which should ensure the FIFO order
func TestJobOrder(t *testing.T) {
	shardEvents := 512
	wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		prev := 0
		count := 0

		for job := range jobChan {
			if testJob, ok := job.(*TestJob); ok {
				if int(testJob.i) <= prev {
					t.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
					wg.Done()
					return
				}

				prev = int(testJob.i)
				count++

				if count >= shardEvents {
					wg.Done()
					return
				}
			}
		}

	}

	shardsNum := runtime.NumCPU()
	// shard cap is 64
	pool, _ := NewShardWorkerPool(shardsNum*64, shardsNum, consumer)
	pool.Init()

	// multi goroutine offer is not guaranteed FIFO order, because race condition may happen in Offer method
	// so we let the producer and consumer to be one-to-one relation.
	for i := 0; i < shardsNum; i++ {
		wg.Add(1)
		counter := uint32(i)
		go func() {
			for j := 0; j < shardEvents; j++ {
				pool.Offer(&TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))}, true)
			}
		}()
	}

	wg.Wait()
}

func eventProcess(b *testing.B) {
	shardEvents := 512
	wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		prev := 0
		count := 0

		for job := range jobChan {
			if testJob, ok := job.(*TestJob); ok {
				if int(testJob.i) <= prev {
					b.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
					wg.Done()
					return
				}

				prev = int(testJob.i)
				count++

				if count >= shardEvents {
					wg.Done()
					return
				}
			}
		}

	}

	shardsNum := runtime.NumCPU()
	// shard cap is 64
	pool, _ := NewShardWorkerPool(shardsNum*64, shardsNum, consumer)
	pool.Init()

	// multi goroutine offer is not guaranteed FIFO order, because race condition may happen in Offer method
	// so we let the producer and consumer to be one-to-one relation.
	for i := 0; i < shardsNum; i++ {
		wg.Add(1)
		counter := uint32(i)
		go func() {
			for j := 0; j < shardEvents; j++ {
				pool.Offer(&TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))}, true)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkShardWorkerPool(b *testing.B) {

	for i := 0; i < b.N; i++ {
		eventProcess(b)
	}
}

// Implements from https://medium.com/capital-one-developers/building-an-unbounded-channel-in-go-789e175cd2cd
func MakeInfinite() (chan<- interface{}, <-chan interface{}) {
	in := make(chan interface{})
	out := make(chan interface{})

	go func() {
		var inQueue []interface{}
		outCh := func() chan interface{} {
			if len(inQueue) == 0 {
				return nil
			}
			return out
		}

		curVal := func() interface{} {
			if len(inQueue) == 0 {
				return nil
			}
			return inQueue[0]
		}

		for len(inQueue) > 0 || in != nil {
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					inQueue = append(inQueue, v)
				}
			case outCh() <- curVal():
				inQueue = inQueue[1:]
			}
		}
		close(out)
	}()
	return in, out
}

func eventProcessWithUnboundedChannel(b *testing.B) {
	shardEvents := 512
	wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		prev := 0
		count := 0

		for job := range jobChan {
			if testJob, ok := job.(*TestJob); ok {
				if int(testJob.i) <= prev {
					b.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
					wg.Done()
					return
				}

				prev = int(testJob.i)
				count++

				if count >= shardEvents {
					wg.Done()
					return
				}
			}
		}

	}

	shardsNum := runtime.NumCPU()

	// multi goroutine offer is not guaranteed FIFO order, because race condition may happen in Offer method
	// so we let the producer and consumer to be one-to-one relation.
	for i := 0; i < shardsNum; i++ {
		wg.Add(1)
		counter := uint32(i)
		in, out := MakeInfinite()

		go func() {
			for j := 0; j < shardEvents; j++ {
				in <- &TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))}
			}
		}()
		go func(shard int) {
			consumer(shard, out)
		}(i)
	}

	wg.Wait()
}

func BenchmarkUnboundChannel(b *testing.B) {

	for i := 0; i < b.N; i++ {
		eventProcessWithUnboundedChannel(b)
	}
}
