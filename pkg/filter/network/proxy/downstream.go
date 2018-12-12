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
	"container/list"
	"context"
	"log"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	"sync"
	"sync/atomic"
)

// stream.StreamEventListener
// stream.StreamReceiver
// stream.StreamFilterChainFactoryCallbacks
// Downstream stream, as a controller to handle downstream and upstream proxy flow
type downStream struct {
	ID      uint32
	proxy   *proxy
	element *list.Element

	// flow control
	bufferLimit uint32

	// control args
	timeout         *Timeout
	responseSender  stream.StreamSender
	upstreamRequest *upstreamRequest
	perRetryTimer   *timer
	responseTimer   *timer

	// downstream request buf
	downstreamReqHeaders  protocol.HeaderMap
	downstreamReqDataBuf  buffer.IoBuffer
	downstreamReqTrailers protocol.HeaderMap

	// downstream response buf
	downstreamRespHeaders  protocol.HeaderMap
	downstreamRespDataBuf  buffer.IoBuffer
	downstreamRespTrailers protocol.HeaderMap

	// state
	// starts to send back downstream response, set on upstream response detected
	downstreamResponseStarted bool
	// downstream request received done
	downstreamRecvDone bool
	// upstream req sent
	upstreamRequestSent bool
	// 1. at the end of upstream response 2. by a upstream reset due to exceptions, such as no healthy upstream, connection close, etc.
	upstreamProcessDone bool

	filterStage int

	downstreamReset   uint32
	downstreamCleaned uint32
	upstreamReset     uint32

	// filters
	//senderFilters   []*activeStreamSenderFilter
	//receiverFilters []*activeStreamReceiverFilter

	// mux for downstream-upstream flow
	mux sync.Mutex

	context context.Context
}

func newActiveStream(ctx context.Context, streamID string, proxy *proxy, responseSender stream.StreamSender) *downStream {
	newCtx := buffer.NewBufferPoolContext(ctx)

	proxyBuffers := proxyBuffersByContext(newCtx)

	s := &proxyBuffers.stream
	s.ID = atomic.AddUint32(&currProxyID, 1)
	s.proxy = proxy

	s.responseSender = responseSender
	s.responseSender.GetStream().AddEventListener(s)
	s.context = newCtx

	//todo: set proxy states

	// start event process
	s.startEventProcess()
	return s
}

// stream.StreamReceiver
func (s *downStream) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) {
	workerPool.Offer(&receiveHeadersEvent{
		streamEvent: streamEvent{
			direction: Downstream,
			streamID:  s.ID,
			stream:    s,
		},
		headers:   headers,
		endStream: endStream,
	})
}

func (s *downStream) OnReceiveData(context context.Context, data buffer.IoBuffer, endStream bool) {
	s.downstreamReqDataBuf = data.Clone()
	s.downstreamReqDataBuf.Count(1)
	data.Drain(data.Len())

	workerPool.Offer(&receiveDataEvent{
		streamEvent: streamEvent{
			direction: Downstream,
			streamID:  s.ID,
			stream:    s,
		},
		data:      s.downstreamReqDataBuf,
		endStream: endStream,
	})
}

func (s *downStream) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) {
	workerPool.Offer(&receiveTrailerEvent{
		streamEvent: streamEvent{
			direction: Downstream,
			streamID:  s.ID,
			stream:    s,
		},
		trailers: trailers,
	})
}

func (s *downStream) GiveStream() {
	if s.upstreamReset == 1 || s.downstreamReset == 1 {
		return
	}
	// reset downstreamReqBuf
	if s.downstreamReqDataBuf != nil {
		buffer.PutIoBuffer(s.downstreamReqDataBuf)
	}

	// Give buffers to bufferPool
	if ctx := buffer.PoolContext(s.context); ctx != nil {
		ctx.Give()
	}
}

func (s *downStream) ResetStream(reason stream.StreamResetReason) {
	s.cleanStream()
}

func (s *downStream) ReceiveHeaders(headers protocol.HeaderMap, endStream bool) {
	s.downstreamRecvDone = endStream
	s.downstreamReqHeaders = headers

	s.doReceiveHeaders(nil, headers, endStream)
}

func (s *downStream) doReceiveHeaders(filter interface{}, headers protocol.HeaderMap, endStream bool) {
	s.sendHijackReply(types.RouterUnavailableCode, headers, false)
}

func (s *downStream) ReceiveData(data buffer.IoBuffer, endStream bool) {
	// if active stream finished before receive data, just ignore further data
	if s.upstreamProcessDone {
		return
	}

	s.downstreamRecvDone = endStream

	s.doReceiveData(nil, data, endStream)
}

func (s *downStream) doReceiveData(filter interface{}, data buffer.IoBuffer, endStream bool) {

}

func (s *downStream) ReceiveTrailers(trailers protocol.HeaderMap) {
	// if active stream finished the lifecycle, just ignore further data
	if s.upstreamProcessDone {
		return
	}

	s.downstreamRecvDone = true

	s.doReceiveTrailers(nil, trailers)
}

func (s *downStream) doReceiveTrailers(filter interface{}, trailers protocol.HeaderMap) {

}

func (s *downStream) onUpstreamHeaders(headers protocol.HeaderMap, endStream bool) {
	s.downstreamRespHeaders = headers

	s.downstreamResponseStarted = true

	if endStream {
		s.onUpstreamResponseRecvFinished()
	}

	// todo: insert proxy headers
	nh := s.convertHeader(headers)
	s.appendHeaders(nh, endStream)
}

func (s *downStream) onUpstreamData(data buffer.IoBuffer, endStream bool) {
	if endStream {
		s.onUpstreamResponseRecvFinished()
	}

	s.appendData(data, endStream)
}

func (s *downStream) onUpstreamTrailers(trailers protocol.HeaderMap) {
	s.onUpstreamResponseRecvFinished()

	s.appendTrailers(trailers)
}

func (s *downStream) onUpstreamResponseRecvFinished() {
	if !s.upstreamRequestSent {
		s.upstreamRequest.resetStream()
	}

	// todo: stats
	// todo: logs

	s.cleanUp()
}

// active stream sender wrapper
func (s *downStream) appendHeaders(headers protocol.HeaderMap, endStream bool) {
	s.upstreamProcessDone = endStream
	s.doAppendHeaders(nil, headers, endStream)
}

func (s *downStream) appendData(data buffer.IoBuffer, endStream bool) {
	s.upstreamProcessDone = endStream
	s.doAppendData(nil, s.convertData(data), endStream)
}

func (s *downStream) appendTrailers(trailers protocol.HeaderMap) {
	s.upstreamProcessDone = true
	s.doAppendTrailers(nil, s.convertTrailer(trailers))
}

func (s *downStream) doAppendHeaders(filter interface{}, headers protocol.HeaderMap, endStream bool) {
	//Currently, just log the error
	if err := s.responseSender.AppendHeaders(s.context, headers, endStream); err != nil {
		log.Println("downstream append headers error:", err)
	}

	if endStream {
		s.endStream()
	}
}

func (s *downStream) doAppendData(filter interface{}, data buffer.IoBuffer, endStream bool) {
}

func (s *downStream) doAppendTrailers(filter interface{}, trailers protocol.HeaderMap) {
}

func (s *downStream) convertHeader(headers protocol.HeaderMap) protocol.HeaderMap {
	dp, up := s.proxy.convertProtocol()

	//if dp != up {
	//	if convHeader, err := protocol.ConvertHeader(s.context, up, dp, headers); err == nil {
	//		return convHeader
	//	} else {
			log.Println("convert header failed:", up, dp)
	//	}
	//}

	return headers
}

func (s *downStream) convertData(data buffer.IoBuffer) buffer.IoBuffer {
	// todo: need data convert
	//dp := types.Protocol(s.proxy.config.DownstreamProtocol)
	//up := types.Protocol(s.proxy.config.UpstreamProtocol)
	//
	//if dp != up {
	//	if convData, err := protocol.ConvertData(s.context, up, dp, data); err == nil {
	//		return convData
	//	} else {
	//		s.logger.Errorf("convert data from %s to %s failed, %s", up, dp, err.Error())
	//	}
	//}
	return data
}

func (s *downStream) convertTrailer(trailers protocol.HeaderMap) protocol.HeaderMap {
	// todo: need protocol convert
	//dp := types.Protocol(s.proxy.config.DownstreamProtocol)
	//up := types.Protocol(s.proxy.config.UpstreamProtocol)
	//
	//if dp != up {
	//	if convTrailer, err := protocol.ConvertTrailer(s.context, up, dp, trailers); err == nil {
	//		return convTrailer
	//	} else {
	//		s.logger.Errorf("convert header from %s to %s failed, %s", up, dp, err.Error())
	//	}
	//}
	return trailers
}

// stream.StreamEventListener
// Called by stream layer normally
func (s *downStream) OnResetStream(reason stream.StreamResetReason) {
	// set downstreamReset flag before real reset logic
	if !atomic.CompareAndSwapUint32(&s.downstreamReset, 0, 1) {
		return
	}

	workerPool.Offer(&resetEvent{
		streamEvent: streamEvent{
			direction: Downstream,
			streamID:  s.ID,
			stream:    s,
		},
		reason: reason,
	})
}

// Clean up on the very end of the stream: end stream or reset stream
// Resources to clean up / reset:
// 	+ upstream request
// 	+ all timers
// 	+ all filters
//  + remove stream in proxy context
func (s *downStream) cleanStream() {
	if !atomic.CompareAndSwapUint32(&s.downstreamCleaned, 0, 1) {
		return
	}

	// reset corresponding upstream stream
	if s.upstreamRequest != nil {
		s.upstreamRequest.resetStream()
	}

	// clean up timers
	s.cleanUp()

	// todo:
	// tell filters it's time to destroy
	//for _, ef := range s.senderFilters {
	//	ef.filter.OnDestroy()
	//}
	//
	//for _, ef := range s.receiverFilters {
	//	ef.filter.OnDestroy()
	//}

	// stop event process
	s.stopEventProcess()
	// delete stream
	s.proxy.deleteActiveStream(s)
}

// case 1: downstream's lifecycle ends normally
// case 2: downstream got reset. See downStream.resetStream for more detail
func (s *downStream) endStream() {
	var isReset bool
	if s.responseSender != nil {
		if !s.downstreamRecvDone || !s.upstreamProcessDone {
			// if downstream req received not done, or local proxy process not done by handle upstream response,
			// just mark it as done and reset stream as a failed case
			s.upstreamProcessDone = true

			// reset downstream will trigger a clean up, see OnResetStream
			s.responseSender.GetStream().ResetStream(stream.StreamLocalReset)
			isReset = true
		}
	}

	if !isReset {
		s.cleanStream()
	}

	// note: if proxy logic resets the stream, there maybe some underlying data in the conn.
	// we ignore this for now, fix as a todo
}

func (s *downStream) cleanUp() {
	// reset upstream request
	// if a downstream filter ends downstream before send to upstream, upstreamRequest will be nil
	if s.upstreamRequest != nil {
		s.upstreamRequest.requestSender = nil
	}

	// reset pertry timer
	if s.perRetryTimer != nil {
		s.perRetryTimer.stop()
		s.perRetryTimer = nil
	}

	// reset response timer
	if s.responseTimer != nil {
		s.responseTimer.stop()
		s.responseTimer = nil
	}
}

func (s *downStream) AddStreamSenderFilter(filter stream.StreamSenderFilter) {

}

func (s *downStream) AddStreamReceiverFilter(filter stream.StreamReceiverFilter) {

}

func (s *downStream) OnDecodeError(context context.Context, err error, headers protocol.HeaderMap) {
	// if active stream finished the lifecycle, just ignore further data
	if s.upstreamProcessDone {
		return
	}

	// todo: enrich headers' information to do some hijack
	// Check headers' info to do hijack
	switch err.Error() {
	case types.CodecException:
		s.sendHijackReply(types.CodecExceptionCode, headers, true)
	case types.DeserializeException:
		s.sendHijackReply(types.DeserialExceptionCode, headers, true)
	default:
		s.sendHijackReply(types.UnknownCode, headers, true)
	}

	s.OnResetStream(stream.StreamLocalReset)
}

func (s *downStream) sendHijackReply(code int, headers protocol.HeaderMap, doConv bool) {
	if headers == nil {
		log.Println("hijack with no headers, stream id:", s.ID)
		raw := make(map[string]string, 5)
		headers = protocol.CommonHeader(raw)
	}

	headers.Set(types.HeaderStatus, strconv.Itoa(code))

	nh := headers
	if doConv {
		nh = s.convertHeader(headers)
	}

	s.appendHeaders(nh, true)
}

func (s *downStream) startEventProcess() {
	// offer start event so that there is no lock contention on the streamProcessMap[shard]
	// all read/write operation should be able to trace back to the ShardWorkerPool goroutine
	workerPool.Offer(&startEvent{
		streamEvent: streamEvent{
			direction: Downstream,
			streamID:  s.ID,
			stream:    s,
		},
	})
}

func (s *downStream) stopEventProcess() {
	workerPool.Offer(&stopEvent{
		streamEvent: streamEvent{
			direction: Downstream,
			streamID:  s.ID,
			stream:    s,
		},
	})
}
