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
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
)

// stream direction
const (
	Downstream = 1
	Upstream   = 2
)

type streamEvent struct {
	direction int
	streamID  string
	stream    *downStream
}

type startEvent struct {
	streamEvent
}

type stopEvent struct {
	streamEvent
}

type resetEvent struct {
	streamEvent

	reason stream.StreamResetReason
}

type receiveHeadersEvent struct {
	streamEvent

	headers   protocol.HeaderMap
	endStream bool
}

type receiveDataEvent struct {
	streamEvent

	data      buffer.IoBuffer
	endStream bool
}

type receiveTrailerEvent struct {
	streamEvent

	trailers protocol.HeaderMap
}

func (s *streamEvent) Source() uint32 {
	source, err := strconv.ParseUint(s.streamID, 10, 32)
	if err != nil {
		panic("streamID must be numeric, unexpected :" + s.streamID)
	}
	return uint32(source)
}

func eventDispatch(shard int, jobChan <-chan interface{}) {
	// stream process status map with shard, we use this to indicate a given stream is processing or not
	streamMap := make(map[string]bool, 1<<10)

	for event := range jobChan {
		eventProcess(shard, streamMap, event)
	}
}

func eventProcess(shard int, streamMap map[string]bool, event interface{}) {
	// TODO: event handles by itself. just call event.handle() here
	switch event.(type) {
	case *startEvent:
		e := event.(*startEvent)
		//log.DefaultLogger.Errorf("[start] %d %d %s", shard, e.direction, e.streamID)

		streamMap[e.streamID] = false
	case *stopEvent:
		e := event.(*stopEvent)
		//log.DefaultLogger.Errorf("[stop] %d %d %s", shard, e.direction, e.streamID)
		e.stream.GiveStream()
		delete(streamMap, e.streamID)
	case *resetEvent:
		e := event.(*resetEvent)
		//log.DefaultLogger.Errorf("[reset] %d %d %s", shard, e.direction, e.streamID)

		if _, ok := streamMap[e.streamID]; ok {
			switch e.direction {
			case Downstream:
				e.stream.ResetStream(e.reason)
			case Upstream:
				e.stream.upstreamRequest.ResetStream(e.reason)
			default:
				log.Println("Unknown receiveTrailerEvent direction:", e.direction)
			}
			//	streamMap[e.streamID] = streamMap[e.streamID] || streamProcessDone(e.stream)
		}
	case *receiveHeadersEvent:
		e := event.(*receiveHeadersEvent)
		//log.DefaultLogger.Errorf("[header] %d %d %s", shard, e.direction, e.streamID)

		if done, ok := streamMap[e.streamID]; ok && !(done || streamProcessDone(e.stream)) {
			switch e.direction {
			case Downstream:
				e.stream.ReceiveHeaders(e.headers, e.endStream)
			case Upstream:
				e.stream.upstreamRequest.ReceiveHeaders(e.headers, e.endStream)
			default:
				log.Println("Unknown receiveHeadersEvent direction:", e.direction)
			}
			streamMap[e.streamID] = streamMap[e.streamID] || streamProcessDone(e.stream)
		}
	case *receiveDataEvent:
		e := event.(*receiveDataEvent)
		//log.DefaultLogger.Errorf("[data] %d %d %s", shard, e.direction, e.streamID)

		if done, ok := streamMap[e.streamID]; ok && !(done || streamProcessDone(e.stream)) {
			switch e.direction {
			case Downstream:
				if e.stream.upstreamRequest == nil {
					// Sometimes runReceiveHeaderFilters will return doReceiveHeaders early,
					// but will not block OnReceiveData, at this scene the upstreamRequest is nil.
					// even if the upstreamRequest is nil, the ReceiveData->doReceiveData will return by runReceiveDataFilters
					// it is ok.
					log.Println("data error:", shard, e.stream)
				}
				e.stream.ReceiveData(e.data, e.endStream)
			case Upstream:
				e.stream.upstreamRequest.ReceiveData(e.data, e.endStream)
			default:
				log.Println("Unknown receiveDataEvent direction:", e.direction)
			}
			streamMap[e.streamID] = streamMap[e.streamID] || streamProcessDone(e.stream)
		}
	case *receiveTrailerEvent:
		e := event.(*receiveTrailerEvent)
		//log.DefaultLogger.Errorf("[trailer] %d %d %s", shard, e.direction, e.stream.streamID)

		if done, ok := streamMap[e.streamID]; ok && !(done || streamProcessDone(e.stream)) {
			switch e.direction {
			case Downstream:
				e.stream.ReceiveTrailers(e.trailers)
			case Upstream:
				e.stream.upstreamRequest.ReceiveTrailers(e.trailers)
			default:
				log.Println("Unknown receiveTrailerEvent direction:", e.direction)
			}
			streamMap[e.streamID] = streamMap[e.streamID] || streamProcessDone(e.stream)
		}

	default:
		log.Println("Unknown event type:", event)
	}
}

func streamProcessDone(s *downStream) bool {
	return s.upstreamProcessDone || s.downstreamReset == 1
}