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
	"context"
	"errors"
	"log"
	"strconv"
	"strike/pkg/admin"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	striketime "strike/utils/time"
	"strings"
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

	// flow control
	bufferLimit uint32

	// control args
	timeout        *Timeout
	responseSender stream.StreamSender
	perRetryTimer  *striketime.Timer
	responseTimer  *striketime.Timer

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
	controlRequestSent bool
	// 1. at the end of upstream response 2. by a upstream reset due to exceptions, such as no healthy upstream, connection close, etc.
	controlProcessDone bool
	emitter            admin.Emitter

	// filter
	filterStage     int
	receiverFilters []*activeStreamReceiverFilter

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

func (s *sourceStream) doEmit(args ...interface{}) (err error) {
	if s.emitter == nil {
		s.sendHijackReply(types.RouterUnavailableCode, s.sourceStreamRespHeaders, false)
		return
	}

	if action, ok := s.sourceStreamReqHeaders.Get(protocol.StrikeHeaderMethod); ok {
		err = s.emitter.Emit(action, args...)
	} else {
		err = errors.New("no method found in request header")
	}

	if err != nil {
		s.sendHijackReply(types.RouterUnavailableCode, s.sourceStreamRespHeaders, false)
	} else {
		s.sendHijackReply(types.SuccessCode, s.sourceStreamRespHeaders, false)
	}

	return
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
	sf := newActiveStreamReceiverFilter(len(s.receiverFilters), s, filter)
	s.receiverFilters = append(s.receiverFilters, sf)
}

// stream.StreamReceiveListener
func (s *sourceStream) OnReceive(ctx context.Context, headers protocol.HeaderMap, data buffer.IoBuffer, trailers protocol.HeaderMap, prioritized bool) {
	s.sourceStreamReqHeaders = headers
	if data != nil {
		s.sourceStreamReqDataBuf = data.Clone()
		data.Drain(data.Len())
	}
	s.sourceStreamReqTrailers = trailers

	s.doReceive()
}

func (s *sourceStream) doReceive() {
	hasBody := s.sourceStreamReqDataBuf != nil
	hasTrailer := s.sourceStreamReqTrailers != nil

	s.doReceiveHeaders(nil, s.sourceStreamReqHeaders, !hasBody)

	if hasBody {
		s.doReceiveData(nil, s.sourceStreamReqDataBuf, !hasTrailer)
	}

	if hasTrailer {
		s.doReceiveTrailers(nil, s.sourceStreamReqTrailers)
	}
}

func (s *sourceStream) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) {
	s.sourceStreamRecvDone = endStream
	s.sourceStreamReqHeaders = headers

	s.doReceiveHeaders(nil, headers, endStream)
}

func (s *sourceStream) doReceiveHeaders(filter *activeStreamReceiverFilter, headers protocol.HeaderMap, endStream bool) {
	// run stream filters
	if s.runReceiveHeadersFilters(filter, headers, endStream) {
		return
	}

	var topic string
	if path, ok := s.sourceStreamReqHeaders.Get(protocol.StrikeHeaderPathKey); ok {
		topic = strings.TrimLeft(path, "/")
		s.emitter, _ = s.controller.GetTargetEmitter(topic)
	}

	if endStream {
		s.doEmit()
	}
}

func (s *sourceStream) OnReceiveData(context context.Context, data buffer.IoBuffer, endStream bool) {
	if s.processDone() {
		return
	}

	s.sourceStreamRecvDone = endStream

	s.doReceiveData(nil, data, endStream)
}

func (s *sourceStream) doReceiveData(filter *activeStreamReceiverFilter, data buffer.IoBuffer, endStream bool) {
	if endStream {
		s.doEmit(data.Bytes())
		data.Drain(data.Len())
	}
}

func (s *sourceStream) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) {
}

func (s *sourceStream) doReceiveTrailers(filter *activeStreamReceiverFilter, trailers protocol.HeaderMap) {
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

func (s *sourceStream) runReceiveHeadersFilters(filter *activeStreamReceiverFilter, headers protocol.HeaderMap, endStream bool) bool {
	var index int
	var f *activeStreamReceiverFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.receiverFilters); index++ {
		f = s.receiverFilters[index]

		s.filterStage |= DecodeHeaders
		status := f.filter.OnReceiveHeaders(s.context, headers, endStream)
		s.filterStage &= ^DecodeHeaders
		if f.handleHeaderStatus(status) {
			// TODO: If it is the last filter, continue with
			// processing since we need to handle the case where a terminal filter wants to buffer, but
			// a previous filter has added body.
			return true
		}
		// TODO: Handle the case where we have a header only request, but a filter adds a body to it.
	}

	return false
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
		log.Println("sourceStream append headers error:", err)
	}

	if endStream {
		s.endStream()
	}
}

func (s *sourceStream) appendData(data buffer.IoBuffer, endStream bool) {
	s.controlProcessDone = endStream
	s.doAppendData(nil, data, endStream)
}

func (s *sourceStream) doAppendData(filter interface{}, data buffer.IoBuffer, endStream bool) {
	s.responseSender.AppendData(s.context, data, endStream)
	if endStream {
		s.endStream()
	}
}

func (s *sourceStream) appendTrailers(trailers protocol.HeaderMap) {
	s.controlProcessDone = true
	s.doAppendTrailers(nil, trailers)
}

func (s *sourceStream) doAppendTrailers(filter interface{}, trailers protocol.HeaderMap) {
	s.responseSender.AppendTrailers(s.context, trailers)
	s.endStream()
}

func (s *sourceStream) endStream() {
	if s.responseSender != nil && !s.sourceStreamRecvDone {
		// not reuse buffer
		atomic.StoreUint32(&s.reuseBuffer, 0)
	}
	s.cleanStream()
}

func (s *sourceStream) cleanStream() {
	if !atomic.CompareAndSwapUint32(&s.sourceStreamCleaned, 0, 1) {
		return
	}

	// clean up timers
	s.cleanUp()

	for _, ef := range s.receiverFilters {
		ef.filter.OnDestroy()
	}

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
	// reset sourceStreamReqBuf
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

func (s *sourceStream) processDone() bool {
	return s.controlProcessDone || atomic.LoadUint32(&s.sourceStreamReset) == 1 || atomic.LoadUint32(&s.sourceStreamCleaned) == 1
}
