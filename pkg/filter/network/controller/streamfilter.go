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
	"log"
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

	activeStream     *sourceStream
	stopped          bool
	stoppedNoBuf     bool
	headersContinued bool
}

func (f *activeStreamFilter) Connection() network.Connection {
	return f.activeStream.controller.readCallbacks.Connection()
}

// stream.StreamReceiverFilterHandler
type activeStreamReceiverFilter struct {
	activeStreamFilter

	filter stream.StreamReceiverFilter
}

func newActiveStreamReceiverFilter(idx int, activeStream *sourceStream,
	filter stream.StreamReceiverFilter) *activeStreamReceiverFilter {
	f := &activeStreamReceiverFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}
	filter.SetReceiveFilterHandler(f)

	return f
}

func (f *activeStreamReceiverFilter) ContinueReceiving() {
	f.doContinue()
}

func (f *activeStreamReceiverFilter) doContinue() {
	if f.activeStream.controlProcessDone {
		return
	}

	f.stopped = false
	hasBuffedData := f.activeStream.sourceStreamReqDataBuf != nil
	hasTrailer := f.activeStream.sourceStreamReqTrailers != nil

	if !f.headersContinued {
		f.headersContinued = true

		endStream := f.activeStream.sourceStreamRecvDone && !hasBuffedData && !hasTrailer
		f.activeStream.doReceiveHeaders(f, f.activeStream.sourceStreamReqHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.sourceStreamReqDataBuf == nil {
			f.activeStream.sourceStreamReqDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.sourceStreamRecvDone && !hasTrailer
		f.activeStream.doReceiveData(f, f.activeStream.sourceStreamReqDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doReceiveTrailers(f, f.activeStream.sourceStreamReqTrailers)
	}
}

func (f *activeStreamReceiverFilter) handleBufferData(buf buffer.IoBuffer) {
	if f.activeStream.sourceStreamReqDataBuf != buf {
		if f.activeStream.sourceStreamReqDataBuf == nil {
			f.activeStream.sourceStreamReqDataBuf = buffer.NewIoBuffer(buf.Len())
			f.activeStream.sourceStreamReqDataBuf.Count(1)
		}

		f.activeStream.sourceStreamReqDataBuf.ReadFrom(buf)
	}
}

// Status Handler Fucntion
// handleHeaderStatus returns true means stop the iteration
func (f *activeStreamReceiverFilter) handleHeaderStatus(status stream.StreamHeadersFilterStatus) bool {
	if status == stream.StreamHeadersFilterStop {
		f.stopped = true
		return true
	}
	if status != stream.StreamHeadersFilterContinue {
		log.Println("unexpected stream header filter status")
	}

	f.headersContinued = true
	return false
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

// handleTrailerStatus returns true means stop the iteration
func (f *activeStreamReceiverFilter) handleTrailerStatus(status stream.StreamTrailersFilterStatus) bool {
	if status == stream.StreamTrailersFilterContinue {
		if f.stopped {
			f.doContinue()
			return true
		}
	} else {
		return true
	}
	// status == types.StreamDataFilterContinue and f.stopped is false
	return false
}

func (f *activeStreamReceiverFilter) AppendHeaders(headers protocol.HeaderMap, endStream bool) {
	f.activeStream.sourceStreamRespHeaders = headers
	f.activeStream.doAppendHeaders(nil, headers, endStream)
}

func (f *activeStreamReceiverFilter) AppendData(buf buffer.IoBuffer, endStream bool) {
	f.activeStream.doAppendData(nil, buf, endStream)
}

func (f *activeStreamReceiverFilter) AppendTrailers(trailers protocol.HeaderMap) {
	f.activeStream.sourceStreamRespTrailers = trailers
	f.activeStream.doAppendTrailers(nil, trailers)
}

func (f *activeStreamReceiverFilter) SendHijackReply(code int, headers protocol.HeaderMap, doConv bool) {
	f.activeStream.sendHijackReply(code, headers, doConv)
}
