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

package proxy

import (
	"log"
	"math"
)

type direction uint8
type eventType uint8

// stream direction
const (
	// direction
	diFromDownstream direction = 1
	diFromUpstream   direction = 2

	// event type
	recvHeader  eventType = 1
	recvData    eventType = 2
	recvTrailer eventType = 3
	reset       eventType = 4
)

var (
	// ratio MUST < 1
	ratio = 3.0 / 10.0
	gap   uint8
)

func initEvent(shardsNum int) {
	// gap should be never changed after init
	gap = uint8(math.Round(float64(shardsNum) * ratio))

	if gap == 0 {
		gap = 1
	}
	if gap >= uint8(shardsNum) {
		gap = uint8(shardsNum - 1)
	}
}

type event struct {
	id  uint32
	dir direction
	evt eventType
	sid uint64

	handle func()
}

func (e *event) Source(sourceShards uint64) (source, targetShards, offset uint64) {
	switch e.dir {
	case diFromDownstream:
		source = e.sid
		targetShards = sourceShards - uint64(gap)
	case diFromUpstream:
		source = e.sid
		targetShards = uint64(gap)
		offset = sourceShards - uint64(gap)
	default:
		log.Println("unsupported event direction")
	}
	return
}

func eventDispatch(shard int, jobChan <-chan interface{}) {
	for job := range jobChan {
		eventProcess(shard, job)
	}
}

func eventProcess(shard int, job interface{}) {
	if ev, ok := job.(*event); ok {
		log.Println("enter event process with dir/type/shard/proxyID", ev.dir, ev.evt, shard, "#", ev.id)

		ev.handle()
	}
}
