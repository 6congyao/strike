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
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
)

// stream.StreamEventListener
// stream.StreamReceiver
// stream.PoolEventListener
type upstreamRequest struct {
	proxy      *proxy
	element    *list.Element
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
	workerPool.Offer(&resetEvent{
		streamEvent: streamEvent{
			direction: Upstream,
			streamID:  r.downStream.ID,
			stream:    r.downStream,
		},
		reason: reason,
	})
}

func (r *upstreamRequest) ReceiveHeaders(headers protocol.HeaderMap, endStream bool) {
	r.upstreamRespHeaders = headers
	r.downStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) ReceiveData(data buffer.IoBuffer, endStream bool) {
	if !r.setupRetry {
		r.downStream.onUpstreamData(data, endStream)
	}
}

func (r *upstreamRequest) ReceiveTrailers(trailers protocol.HeaderMap) {
	if !r.setupRetry {
		r.downStream.onUpstreamTrailers(trailers)
	}
}

func (r *upstreamRequest) appendHeaders(headers protocol.HeaderMap, endStream bool) {
	r.sendComplete = endStream
}

func (r *upstreamRequest) appendData(data buffer.IoBuffer, endStream bool) {
	r.sendComplete = endStream
	r.dataSent = true
	//r.requestSender.AppendData(r.downStream.context, r.convertData(data), endStream)
}
