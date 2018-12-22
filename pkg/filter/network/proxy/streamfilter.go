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
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
)

// FilterStage is the type of the filter stage
type FilterStage int

// Const of all stages
const (
	DecodeHeaders = iota
	DecodeData
	DecodeTrailers
	EncodeHeaders
	EncodeData
	EncodeTrailers
)

type activeStreamFilter struct {
	index int

	activeStream     *downStream
	stopped          bool
	stoppedNoBuf     bool
	headersContinued bool
}

func (f *activeStreamFilter) Connection() network.Connection {
	return f.activeStream.proxy.readCallbacks.Connection()
}

func (f *activeStreamFilter) ResetStream() {
	f.activeStream.resetStream()
}

//func (f *activeStreamFilter) Route() Route {
//	return f.activeStream.route
//}

func (f *activeStreamFilter) StreamID() string {
	return strconv.FormatUint(uint64(f.activeStream.ID), 10)
}

//func (f *activeStreamFilter) RequestInfo() RequestInfo {
//	return f.activeStream.requestInfo
//}

// stream.StreamSenderFilterCallbacks
type activeStreamSenderFilter struct {
	activeStreamFilter

	filter stream.StreamSenderFilter
}

func newActiveStreamSenderFilter(idx int, activeStream *downStream,
	filter stream.StreamSenderFilter) *activeStreamSenderFilter {
	f := &activeStreamSenderFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}

	filter.SetEncoderFilterCallbacks(f)

	return f
}

func (f *activeStreamSenderFilter) doContinue() {
	f.stopped = false
	hasBuffedData := f.activeStream.downstreamRespDataBuf != nil
	hasTrailer := f.activeStream.downstreamRespTrailers != nil

	if !f.headersContinued {
		f.headersContinued = true
		endStream := f.activeStream.upstreamProcessDone && !hasBuffedData && !hasTrailer
		f.activeStream.doAppendHeaders(f, f.activeStream.downstreamRespHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.downstreamRespDataBuf == nil {
			f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.downstreamRecvDone && !hasTrailer
		f.activeStream.doAppendData(f, f.activeStream.downstreamRespDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doAppendTrailers(f, f.activeStream.downstreamRespTrailers)
	}
}

func (f *activeStreamSenderFilter) handleBufferData(buf buffer.IoBuffer) {
	if f.activeStream.downstreamRespDataBuf != buf {
		if f.activeStream.downstreamRespDataBuf == nil {
			f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(buf.Len())
		}

		f.activeStream.downstreamRespDataBuf.ReadFrom(buf)
	}
}

// handleDataStatus returns true means stop the iteration
// TODO: check wether the buffer is streaming
func (f *activeStreamSenderFilter) handleDataStatus(status stream.StreamDataFilterStatus, data buffer.IoBuffer) bool {
	if status == stream.StreamDataFilterContinue {
		if f.stopped {
			f.handleBufferData(data)
			f.doContinue()
			return true
		}
	} else {
		f.stopped = true

		switch status {
		case stream.StreamDataFilterStopAndBuffer:
			f.handleBufferData(data)
		case stream.StreamDataFilterStop:
			f.stoppedNoBuf = true
			// make sure no data banked up
			data.Reset()
		}
		return true
	}

	return false
}

func (f *activeStreamSenderFilter) ContinueEncoding() {
	f.doContinue()
}

func (f *activeStreamSenderFilter) EncodingBuffer() buffer.IoBuffer {
	return f.activeStream.downstreamRespDataBuf
}

func (f *activeStreamSenderFilter) AddEncodedData(buf buffer.IoBuffer, streamingFilter bool) {
	f.activeStream.addEncodedData(f, buf, streamingFilter)
}

func (f *activeStreamSenderFilter) SetEncoderBufferLimit(limit uint32) {
	f.activeStream.setBufferLimit(limit)
}

func (f *activeStreamSenderFilter) EncoderBufferLimit() uint32 {
	return f.activeStream.bufferLimit
}

// stream.StreamReceiverFilterCallbacks
type activeStreamReceiverFilter struct {
	activeStreamFilter

	filter stream.StreamReceiverFilter
}

func newActiveStreamReceiverFilter(idx int, activeStream *downStream,
	filter stream.StreamReceiverFilter) *activeStreamReceiverFilter {
	f := &activeStreamReceiverFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}
	filter.SetDecoderFilterCallbacks(f)

	return f
}

func (f *activeStreamReceiverFilter) doContinue() {
	if f.activeStream.upstreamProcessDone {
		return
	}

	f.stopped = false
	hasBuffedData := f.activeStream.downstreamReqDataBuf != nil
	hasTrailer := f.activeStream.downstreamReqTrailers != nil

	if !f.headersContinued {
		f.headersContinued = true

		endStream := f.activeStream.downstreamRecvDone && !hasBuffedData && !hasTrailer
		f.activeStream.doReceiveHeaders(f, f.activeStream.downstreamReqHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.downstreamReqDataBuf == nil {
			f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.downstreamRecvDone && !hasTrailer
		f.activeStream.doReceiveData(f, f.activeStream.downstreamReqDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doReceiveTrailers(f, f.activeStream.downstreamReqTrailers)
	}
}

func (f *activeStreamReceiverFilter) handleBufferData(buf buffer.IoBuffer) {
	if f.activeStream.downstreamReqDataBuf != buf {
		if f.activeStream.downstreamReqDataBuf == nil {
			f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(buf.Len())
		}

		f.activeStream.downstreamReqDataBuf.ReadFrom(buf)
	}
}

// handleDataStatus returns true means stop the iteration
// TODO: check wether the buffer is streaming
func (f *activeStreamReceiverFilter) handleDataStatus(status stream.StreamDataFilterStatus, data buffer.IoBuffer) bool {
	if status == stream.StreamDataFilterContinue {
		if f.stopped {
			f.handleBufferData(data)
			f.doContinue()
			return true
		}
	} else {
		f.stopped = true

		switch status {
		case stream.StreamDataFilterStopAndBuffer:
			f.handleBufferData(data)
		case stream.StreamDataFilterStop:
			f.stoppedNoBuf = true
			// make sure no data banked up
			data.Reset()
		}
		return true
	}

	return false
}

func (f *activeStreamReceiverFilter) ContinueDecoding() {
	f.doContinue()
}

func (f *activeStreamReceiverFilter) DecodingBuffer() buffer.IoBuffer {
	return f.activeStream.downstreamReqDataBuf
}

func (f *activeStreamReceiverFilter) AddDecodedData(buf buffer.IoBuffer, streamingFilter bool) {
	f.activeStream.addDecodedData(f, buf, streamingFilter)
}

func (f *activeStreamReceiverFilter) AppendHeaders(headers protocol.HeaderMap, endStream bool) {
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.doAppendHeaders(nil, headers, endStream)
}

func (f *activeStreamReceiverFilter) AppendData(buf buffer.IoBuffer, endStream bool) {
	f.activeStream.doAppendData(nil, buf, endStream)
}

func (f *activeStreamReceiverFilter) AppendTrailers(trailers protocol.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.doAppendTrailers(nil, trailers)
}

func (f *activeStreamReceiverFilter) SetDecoderBufferLimit(limit uint32) {
	f.activeStream.setBufferLimit(limit)
}

func (f *activeStreamReceiverFilter) DecoderBufferLimit() uint32 {
	return f.activeStream.bufferLimit
}

func (f *activeStreamReceiverFilter) SendHijackReply(code int, headers protocol.HeaderMap, doConv bool) {
	f.activeStream.sendHijackReply(code, headers, doConv)
}
