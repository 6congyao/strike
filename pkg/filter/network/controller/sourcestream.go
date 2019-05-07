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

package controller

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/server"
	"strike/pkg/stream"
	"strike/pkg/types"
	"strike/utils"
	"sync/atomic"
	"time"
)

// Timeout
type Timeout struct {
	GlobalTimeout time.Duration
	TryTimeout    time.Duration
}

// stream.StreamEventListener
// stream.StreamReceiver
// stream.StreamFilterChainFactoryCallbacks
type sourceStream struct {
	ID         uint32
	controller *controller
	element    *list.Element

	// flow control
	bufferLimit uint32

	// control args
	timeout        *Timeout
	responseSender stream.StreamSender
	perRetryTimer  *utils.Timer
	responseTimer  *utils.Timer

	// sourceStream request buf
	sourceStreamReqHeaders  protocol.HeaderMap
	sourceStreamReqDataBuf  buffer.IoBuffer
	sourceStreamReqTrailers protocol.HeaderMap

	// sourceStream response buf
	sourceStreamRespHeaders  protocol.HeaderMap
	sourceStreamRespDataBuf  buffer.IoBuffer
	sourceStreamRespTrailers protocol.HeaderMap

	// state
	// starts to send back sourceStream response, set on upstream response detected
	sourceStreamResponseStarted bool
	// sourceStream request received done
	sourceStreamRecvDone bool
	// upstream req sent
	upstreamRequestSent bool
	// 1. at the end of upstream response 2. by a upstream reset due to exceptions, such as no healthy upstream, connection close, etc.
	controlProcessDone bool

	sourceStreamReset   uint32
	sourceStreamCleaned uint32
	reuseBuffer         uint32

	context context.Context
}

func newActiveStream(ctx context.Context, controller *controller, responseSender stream.StreamSender) *sourceStream {
	newCtx := buffer.NewBufferPoolContext(ctx)

	proxyBuffers := controllerBuffersByContext(newCtx)

	s := &proxyBuffers.stream
	s.ID = atomic.AddUint32(&currControllerID, 1)
	s.controller = controller

	s.responseSender = responseSender
	s.responseSender.GetStream().AddEventListener(s)
	s.context = newCtx
	s.reuseBuffer = 1

	//todo: set proxy states

	return s
}

func (s *sourceStream) OnDestroyStream() {}

// stream.StreamEventListener
// Called by stream layer normally
func (s *sourceStream) OnResetStream(reason stream.StreamResetReason) {
	if !atomic.CompareAndSwapUint32(&s.sourceStreamReset, 0, 1) {
		return
	}

	return
}

func (s *sourceStream) AddStreamSenderFilter(filter stream.StreamSenderFilter) {

}

func (s *sourceStream) AddStreamReceiverFilter(filter stream.StreamReceiverFilter) {

}

// stream.StreamReceiver
func (s *sourceStream) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) {
	s.sourceStreamRecvDone = endStream
	s.sourceStreamReqHeaders = headers

	ch := context.Value(types.ContextKeyConnHandlerRef).(server.ConnectionHandler)

	if ch != nil {
		fmt.Println(ch.NumConnections())
	}

	s.sendHijackReply(200, headers, false)
}

func (s *sourceStream) OnReceiveData(context context.Context, data buffer.IoBuffer, endStream bool) {
	s.sourceStreamReqDataBuf = data.Clone()
	s.sourceStreamReqDataBuf.Count(1)
	data.Drain(data.Len())
}

func (s *sourceStream) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) {

}

func (s *sourceStream) OnDecodeError(context context.Context, err error, headers protocol.HeaderMap) {
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

func (s *sourceStream) sendHijackReply(code int, headers protocol.HeaderMap, doConv bool) {
	if headers == nil {
		log.Println("hijack with no headers, stream id:", s.ID)
		raw := make(map[string]string, 5)
		headers = protocol.CommonHeader(raw)
	}

	headers.Set(types.HeaderStatus, strconv.Itoa(code))

	s.appendHeaders(headers, true)
}

// active stream sender wrapper
func (s *sourceStream) appendHeaders(headers protocol.HeaderMap, endStream bool) {
	s.controlProcessDone = endStream
	s.doAppendHeaders(nil, headers, endStream)
}

func (s *sourceStream) doAppendHeaders(filter interface{}, headers protocol.HeaderMap, endStream bool) {
	//Currently, just log the error
	if err := s.responseSender.AppendHeaders(s.context, headers, endStream); err != nil {
		log.Println("downstream append headers error:", err)
	}

	if endStream {
		s.endStream()
	}
}

func (s *sourceStream) endStream() {
	if s.responseSender != nil && !s.sourceStreamRecvDone {
		// not reuse buffer
		atomic.StoreUint32(&s.reuseBuffer, 0)
	}
	s.cleanStream()

	// note: if proxy logic resets the stream, there maybe some underlying data in the conn.
	// we ignore this for now, fix as a todo
}

func (s *sourceStream) cleanStream() {
	if !atomic.CompareAndSwapUint32(&s.sourceStreamCleaned, 0, 1) {
		return
	}

	// clean up timers
	s.cleanUp()

	// todo:
	// tell filters it's time to destroy
	//for _, ef := range s.senderFilters {
	//	ef.filter.OnDestroy()
	//}



	// recycle if no reset events
	s.giveStream()
}

func (s *sourceStream) cleanUp() {
	// reset pertry timer
	if s.perRetryTimer != nil {
		s.perRetryTimer.Stop()
		s.perRetryTimer = nil
	}

	// reset response timer
	if s.responseTimer != nil {
		s.responseTimer.Stop()
		s.responseTimer = nil
	}
}

func (s *sourceStream) giveStream() {
	if atomic.LoadUint32(&s.reuseBuffer) != 1 {
		return
	}
	// reset downstreamReqBuf
	if s.sourceStreamReqDataBuf != nil {
		if e := buffer.PutIoBuffer(s.sourceStreamReqDataBuf); e != nil {
			log.Println("PutIoBuffer error:", e)
		}
	}

	// Give buffers to bufferPool
	if ctx := buffer.PoolContext(s.context); ctx != nil {
		ctx.Give()
	}
}