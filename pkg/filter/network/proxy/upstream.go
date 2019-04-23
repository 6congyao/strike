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
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	"strike/pkg/upstream"
)

// stream.StreamEventListener
// stream.StreamReceiver
// stream.PoolEventListener
type upstreamRequest struct {
	proxy      *proxy
	downStream *downStream
	//host          types.Host
	requestSender stream.StreamSender
	connPool      stream.ConnectionPool

	// ~~~ upstream response buf
	upstreamRespHeaders protocol.HeaderMap

	//~~~ state
	sendComplete bool
	dataSent     bool
	trailerSent  bool
	setupRetry   bool

	element *list.Element
}

// reset upstream request in proxy context
// 1. downstream cleanup
// 2. on upstream global timeout
// 3. on upstream per req timeout
// 4. on upstream response receive error
// 5. before a retry
func (r *upstreamRequest) resetStream() {
	// only reset a alive request sender stream
	if r.requestSender != nil {
		r.requestSender.GetStream().RemoveEventListener(r)
		r.requestSender.GetStream().ResetStream(stream.StreamLocalReset)
	}
}

func (r *upstreamRequest) ResetStream(reason stream.StreamResetReason) {
	r.requestSender = nil

	if !r.setupRetry {
		// todo: check if we get a reset on encode request headers. e.g. send failed
		//r.downStream.onUpstreamReset(UpstreamReset, reason)
	}
}

// stream.StreamEventListener
// Called by stream layer normally
func (r *upstreamRequest) OnResetStream(reason stream.StreamResetReason) {
	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: diUpstream,
		evt: reset,
		handle: func() {
			r.ResetStream(reason)
		},
	}, false)
}

func (r *upstreamRequest) OnDestroyStream() {}

func (r *upstreamRequest) endStream() {
	// todo: record upstream process time in request info
}

func (r *upstreamRequest) ReceiveHeaders(headers protocol.HeaderMap, endStream bool) {
	r.upstreamRespHeaders = headers
	r.downStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) ReceiveData(data buffer.IoBuffer, endStream bool) {
	if r.downStream.processDone() {
		return
	}

	if !r.setupRetry {
		r.downStream.onUpstreamData(data, endStream)
	}
}

func (r *upstreamRequest) ReceiveTrailers(trailers protocol.HeaderMap) {
	if r.downStream.processDone() {
		return
	}

	if !r.setupRetry {
		r.downStream.onUpstreamTrailers(trailers)
	}
}

func (r *upstreamRequest) appendHeaders(headers protocol.HeaderMap, endStream bool) {
	if r.downStream.processDone() {
		return
	}

	r.sendComplete = endStream
	r.connPool.NewStream(r.downStream.context, r, r)
}

func (r *upstreamRequest) appendData(data buffer.IoBuffer, endStream bool) {
	if r.downStream.processDone() {
		return
	}

	r.sendComplete = endStream
	r.dataSent = true
	if r.requestSender == nil {
		r.downStream.upstreamProcessDone = true
		r.downStream.sendHijackReply(types.UpstreamOverFlowCode, r.downStream.downstreamReqHeaders, false)
		//r.proxy.readCallbacks.Connection().Close(network.FlushWrite, network.LocalClose)
		log.Println("Request sender is nil while appending data.")
		return
	}
	r.requestSender.AppendData(r.downStream.context, data, endStream)
}

func (r *upstreamRequest) appendTrailers(trailers protocol.HeaderMap) {
	if r.downStream.processDone() {
		return
	}

	r.sendComplete = true
	r.trailerSent = true
	r.requestSender.AppendTrailers(r.downStream.context, trailers)
}

// stream.StreamReceiver
// Method to decode upstream's response message
func (r *upstreamRequest) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) {
	// save response code
	//if status, ok := headers.Get(protocol.StrikeResponseStatusCode); ok {
	//	if code, err := strconv.Atoi(status); err == nil {
	//		r.downStream.requestInfo.SetResponseCode(uint32(code))
	//	}
	//	headers.Del(protocol.StrikeResponseStatusCode)
	//}

	if endStream {
		r.endStream()
	}

	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: diUpstream,
		evt: recvHeader,
		handle: func() {
			r.ReceiveHeaders(headers, endStream)
		},
	}, true)
}

func (r *upstreamRequest) OnReceiveData(context context.Context, data buffer.IoBuffer, endStream bool) {
	r.downStream.downstreamRespDataBuf = data.Clone()
	data.Drain(data.Len())

	if endStream {
		r.endStream()
	}

	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: diUpstream,
		evt: recvData,
		handle: func() {
			r.ReceiveData(r.downStream.downstreamRespDataBuf, endStream)
		},
	}, true)
}

func (r *upstreamRequest) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) {
	r.endStream()

	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: diUpstream,
		evt: recvTrailer,
		handle: func() {
			r.ReceiveTrailers(trailers)
		},
	}, true)
}

func (r *upstreamRequest) OnDecodeError(context context.Context, err error, headers protocol.HeaderMap) {
	r.OnResetStream(stream.StreamLocalReset)
}

// stream.PoolEventListener
func (r *upstreamRequest) OnFailure(reason stream.PoolFailureReason, host upstream.Host) {
	var resetReason stream.StreamResetReason

	switch reason {
	case stream.Overflow:
		resetReason = stream.StreamOverflow
	case stream.ConnectionFailure:
		resetReason = stream.StreamConnectionFailed
	}

	r.ResetStream(resetReason)
}

func (r *upstreamRequest) OnReady(sender stream.StreamSender, host upstream.Host) {
	r.requestSender = sender
	r.requestSender.GetStream().AddEventListener(r)

	endStream := r.sendComplete && !r.dataSent && !r.trailerSent
	r.requestSender.AppendHeaders(r.downStream.context, r.convertHeader(r.downStream.downstreamReqHeaders), endStream)

	// todo: check if we get a reset on send headers
}

func (r *upstreamRequest) convertHeader(headers protocol.HeaderMap) protocol.HeaderMap {
	return headers
}
