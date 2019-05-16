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
)

type direction uint8
type eventType uint8

// stream direction
const (
	// direction
	diDownstream direction = 1
	diUpstream   direction = 2

	// event type
	recvHeader  eventType = 1
	recvData    eventType = 2
	recvTrailer eventType = 3
	reset       eventType = 4
)

type event struct {
	id  uint32
	dir direction
	evt eventType

	handle func()
}

func (e *event) Source(sourceShards uint32) (source, targetShards uint32) {
	if e.dir == diDownstream {
		source = e.id
		targetShards = sourceShards - 1
	} else {
		source = sourceShards - 1
		targetShards = sourceShards
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
		log.Println("enter event process with proxyID/dir/type", ev.id, ev.dir, ev.evt)

		ev.handle()
	}
}
