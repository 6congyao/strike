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
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
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