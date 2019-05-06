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
	"errors"
	"log"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	"strike/pkg/upstream"
	"strike/utils"
	"sync/atomic"
	"time"
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
	perRetryTimer   *utils.Timer
	responseTimer   *utils.Timer

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
	reuseBuffer       uint32

	// filters
	senderFilters   []*activeStreamSenderFilter
	receiverFilters []*activeStreamReceiverFilter

	context context.Context
}

func newActiveStream(ctx context.Context, proxy *proxy, responseSender stream.StreamSender) *downStream {
	newCtx := buffer.NewBufferPoolContext(ctx)

	proxyBuffers := proxyBuffersByContext(newCtx)

	s := &proxyBuffers.stream
	s.ID = atomic.AddUint32(&currProxyID, 1)
	s.proxy = proxy

	s.responseSender = responseSender
	s.responseSender.GetStream().AddEventListener(s)
	s.context = newCtx
	s.reuseBuffer = 1

	//todo: set proxy states

	return s
}

// stream.StreamReceiver
func (s *downStream) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) {
	workerPool.Offer(&event{
		id:  s.ID,
		dir: diDownstream,
		evt: recvHeader,
		handle: func() {
			s.ReceiveHeaders(headers, endStream)
		},
	}, true)
}

func (s *downStream) OnReceiveData(context context.Context, data buffer.IoBuffer, endStream bool) {
	s.downstreamReqDataBuf = data.Clone()
	s.downstreamReqDataBuf.Count(1)
	data.Drain(data.Len())

	workerPool.Offer(&event{
		id:  s.ID,
		dir: diDownstream,
		evt: recvData,
		handle: func() {
			s.ReceiveData(s.downstreamReqDataBuf, endStream)
		},
	}, true)
}

func (s *downStream) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) {
	workerPool.Offer(&event{
		id:  s.ID,
		dir: diDownstream,
		evt: recvTrailer,
		handle: func() {
			s.ReceiveTrailers(trailers)
		},
	}, true)
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

func (s *downStream) ResetStream(reason stream.StreamResetReason) {
	s.cleanStream()
}

func (s *downStream) OnDestroyStream() {}

func (s *downStream) ReceiveHeaders(headers protocol.HeaderMap, endStream bool) {
	s.downstreamRecvDone = endStream
	s.downstreamReqHeaders = headers

	s.doReceiveHeaders(nil, headers, endStream)
}

func (s *downStream) doReceiveHeaders(filter *activeStreamReceiverFilter, headers protocol.HeaderMap, endStream bool) {
	// run stream filters
	if s.runReceiveHeadersFilters(filter, headers, endStream) {
		return
	}

	connPool, err := s.initializeUpstreamConnectionPool()

	if err != nil {
		log.Println("initialize Upstream Connection Pool error:", err)
		if connPool == nil {
			s.sendHijackReply(types.NoHealthUpstreamCode, s.downstreamReqHeaders, false)
		}
		return
	}

	pb := proxyBuffersByContext(s.context)
	s.upstreamRequest = &pb.request
	s.upstreamRequest.downStream = s
	s.upstreamRequest.proxy = s.proxy
	s.upstreamRequest.connPool = connPool

	s.upstreamRequest.appendHeaders(headers, endStream)

	if endStream {
		s.onUpstreamRequestSent()
	}
}

func (s *downStream) ReceiveData(data buffer.IoBuffer, endStream bool) {
	// if active stream finished before receive data, just ignore further data
	if s.processDone() {
		return
	}

	s.downstreamRecvDone = endStream

	s.doReceiveData(nil, data, endStream)
}

func (s *downStream) doReceiveData(filter *activeStreamReceiverFilter, data buffer.IoBuffer, endStream bool) {
	if endStream {
		s.onUpstreamRequestSent()
	}
	s.upstreamRequest.appendData(data, endStream)

	if s.upstreamProcessDone {
		s.cleanStream()
	}
}

func (s *downStream) ReceiveTrailers(trailers protocol.HeaderMap) {
	// if active stream finished the lifecycle, just ignore further data
	if s.processDone() {
		return
	}

	s.downstreamRecvDone = true

	s.doReceiveTrailers(nil, trailers)
}

func (s *downStream) doReceiveTrailers(filter *activeStreamReceiverFilter, trailers protocol.HeaderMap) {
	if s.runReceiveTrailersFilters(filter, trailers) {
		return
	}

	s.downstreamReqTrailers = trailers
	s.onUpstreamRequestSent()
	s.upstreamRequest.appendTrailers(trailers)

	// if upstream process done in the middle of receiving trailers, just end stream
	if s.upstreamProcessDone {
		s.cleanStream()
	}
}

func (s *downStream) initializeUpstreamConnectionPool() (connPool stream.ConnectionPool, err error) {
	upProtocol := s.getUpstreamProtocol()
	connPool = s.getUpstreamConnPool(upProtocol)
	if connPool == nil {
		err = errors.New("no health upstream connection found")
	}

	return
}

func (s *downStream) getDownstreamProtocol() (prot protocol.Protocol) {
	if s.proxy.serverStreamConn == nil {
		prot = protocol.Protocol(s.proxy.config.DownstreamProtocol)
	} else {
		prot = s.proxy.serverStreamConn.Protocol()
	}
	return
}

func (s *downStream) getUpstreamProtocol() (prot protocol.Protocol) {
	configProt := s.proxy.config.UpstreamProtocol

	// Auto means same as downstream protocol
	if configProt == string(protocol.Auto) {
		prot = s.getDownstreamProtocol()
	} else {
		prot = protocol.Protocol(configProt)
	}

	return
}

func (s *downStream) getUpstreamConnPool(prot protocol.Protocol) stream.ConnectionPool {
	var host upstream.Host
	if factory, ok := stream.ConnPoolFactories[prot]; ok {
		return factory(host)
	}
	return nil
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
	//dp, up := s.proxy.convertProtocol()

	//if dp != up {
	//	if convHeader, err := protocol.ConvertHeader(s.context, up, dp, headers); err == nil {
	//		return convHeader
	//	} else {
	//log.Println("convert header for:", up, dp)
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
	if !atomic.CompareAndSwapUint32(&s.downstreamReset, 0, 1) {
		return
	}

	workerPool.Offer(&event{
		id:  s.ID,
		dir: diDownstream,
		evt: reset,
		handle: func() {
			s.ResetStream(reason)
		},
	}, false)
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
	if s.upstreamRequest != nil && !s.upstreamProcessDone {
		s.upstreamProcessDone = true
		s.upstreamRequest.resetStream()
	}

	// clean up timers
	s.cleanUp()

	// todo:
	// tell filters it's time to destroy
	//for _, ef := range s.senderFilters {
	//	ef.filter.OnDestroy()
	//}

	for _, ef := range s.receiverFilters {
		ef.filter.OnDestroy()
	}

	// delete stream
	s.proxy.deleteActiveStream(s)

	// recycle if no reset events
	s.giveStream()
}

// case 1: downstream's lifecycle ends normally
// case 2: downstream got reset. See downStream.resetStream for more detail
func (s *downStream) endStream() {
	if s.responseSender != nil && !s.downstreamRecvDone {
		// not reuse buffer
		atomic.StoreUint32(&s.reuseBuffer, 0)
	}
	s.cleanStream()

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
		s.perRetryTimer.Stop()
		s.perRetryTimer = nil
	}

	// reset response timer
	if s.responseTimer != nil {
		s.responseTimer.Stop()
		s.responseTimer = nil
	}
}

func (s *downStream) AddStreamSenderFilter(filter stream.StreamSenderFilter) {
	sf := newActiveStreamSenderFilter(len(s.senderFilters), s, filter)
	s.senderFilters = append(s.senderFilters, sf)
}

func (s *downStream) AddStreamReceiverFilter(filter stream.StreamReceiverFilter) {
	sf := newActiveStreamReceiverFilter(len(s.receiverFilters), s, filter)
	s.receiverFilters = append(s.receiverFilters, sf)
}

func (s *downStream) resetStream() {
	if s.responseSender != nil && !s.upstreamProcessDone {
		// if downstream req received not done, or local proxy process not done by handle upstream response,
		// just mark it as done and reset stream as a failed case
		s.upstreamProcessDone = true

		// reset downstream will trigger a clean up, see OnResetStream
		s.responseSender.GetStream().ResetStream(stream.StreamLocalReset)
	}
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

func (s *downStream) onUpstreamRequestSent() {
	s.upstreamRequestSent = true

	// todo: implement timeout for upstream req
	if s.upstreamRequest != nil && s.timeout != nil {
		// setup per req timeout timer
		s.setupPerReqTimeout()

		// setup global timeout timer
		if s.timeout.GlobalTimeout > 0 {
			if s.responseTimer != nil {
				s.responseTimer.Stop()
			}

			id := s.ID
			s.responseTimer = utils.NewTimer(s.timeout.GlobalTimeout,
				func() {
					if atomic.LoadUint32(&s.downstreamCleaned) == 1 {
						return
					}
					if id != s.ID {
						return
					}
					s.onResponseTimeout()
				})
		}
	}
}

func (s *downStream) setupPerReqTimeout() {
	timeout := s.timeout

	if timeout.TryTimeout > 0 {
		if s.perRetryTimer != nil {
			s.perRetryTimer.Stop()
		}

		ID := s.ID
		s.perRetryTimer = utils.NewTimer(timeout.TryTimeout*time.Second,
			func() {
				if atomic.LoadUint32(&s.downstreamCleaned) == 1 {
					return
				}
				if ID != s.ID {
					return
				}
				s.onPerReqTimeout()
			})
	}
}

func (s *downStream) onResponseTimeout() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("onResponseTimeout() panic:", r)
		}
	}()
	s.responseTimer = nil

	if s.upstreamRequest != nil {
		//if s.upstreamRequest.host != nil {
		//	s.upstreamRequest.host.HostStats().UpstreamRequestTimeout.Inc(1)
		//}

		atomic.StoreUint32(&s.reuseBuffer, 0)
		s.upstreamRequest.resetStream()
		s.upstreamRequest.OnResetStream(stream.UpstreamGlobalTimeout)
	}
}

func (s *downStream) onPerReqTimeout() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("onPerReqTimeout() panic", r)
		}
	}()

	if !s.downstreamResponseStarted {
		// handle timeout on response not

		s.perRetryTimer = nil
		//s.cluster.Stats().UpstreamRequestTimeout.Inc(1)
		//
		//if s.upstreamRequest.host != nil {
		//	s.upstreamRequest.host.HostStats().UpstreamRequestTimeout.Inc(1)
		//}

		atomic.StoreUint32(&s.reuseBuffer, 0)
		s.upstreamRequest.resetStream()
		//s.requestInfo.SetResponseFlag(types.UpstreamRequestTimeout)
		s.upstreamRequest.OnResetStream(stream.UpstreamPerTryTimeout)
	} else {
		log.Println("Skip request timeout on getting upstream response")
	}
}

func (s *downStream) addEncodedData(filter *activeStreamSenderFilter, data buffer.IoBuffer, streaming bool) {
	if s.filterStage == 0 || s.filterStage&EncodeHeaders > 0 ||
		s.filterStage&EncodeData > 0 {

		filter.handleBufferData(data)
	} else if s.filterStage&EncodeTrailers > 0 {
		s.runAppendDataFilters(filter, data, false)
	}
}

func (s *downStream) runAppendDataFilters(filter *activeStreamSenderFilter, data buffer.IoBuffer, endStream bool) bool {
	var index int
	var f *activeStreamSenderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.senderFilters); index++ {
		f = s.senderFilters[index]

		s.filterStage |= EncodeData
		status := f.filter.AppendData(s.context, data, endStream)
		s.filterStage &= ^EncodeData
		if f.handleDataStatus(status, data) {
			return true
		}
	}

	return false
}

func (s *downStream) setBufferLimit(bufferLimit uint32) {
	s.bufferLimit = bufferLimit
}

func (s *downStream) addDecodedData(filter *activeStreamReceiverFilter, data buffer.IoBuffer, streaming bool) {
	if s.filterStage == 0 || s.filterStage&DecodeHeaders > 0 ||
		s.filterStage&DecodeData > 0 {

		filter.handleBufferData(data)
	} else if s.filterStage&EncodeTrailers > 0 {
		s.runReceiveDataFilters(filter, data, false)
	}
}

func (s *downStream) runReceiveHeadersFilters(filter *activeStreamReceiverFilter, headers protocol.HeaderMap, endStream bool) bool {
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

func (s *downStream) runReceiveDataFilters(filter *activeStreamReceiverFilter, data buffer.IoBuffer, endStream bool) bool {
	if s.upstreamProcessDone {
		return false
	}

	var index int
	var f *activeStreamReceiverFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.receiverFilters); index++ {
		f = s.receiverFilters[index]

		s.filterStage |= DecodeData
		status := f.filter.OnReceiveData(s.context, data, endStream)
		s.filterStage &= ^DecodeData
		if f.handleDataStatus(status, data) {
			// TODO: If it is the last filter, continue with
			// processing since we need to handle the case where a terminal filter wants to buffer, but
			// a previous filter has added body.
			return true
		}
	}

	return false
}

func (s *downStream) runReceiveTrailersFilters(filter *activeStreamReceiverFilter, trailers protocol.HeaderMap) bool {
	if s.upstreamProcessDone {
		return false
	}

	var index int
	var f *activeStreamReceiverFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.receiverFilters); index++ {
		f = s.receiverFilters[index]

		s.filterStage |= DecodeTrailers
		status := f.filter.OnReceiveTrailers(s.context, trailers)
		s.filterStage &= ^DecodeTrailers
		if f.handleTrailerStatus(status) {
			return true
		}
	}

	return false
}

func (s *downStream) reset() {
	s.ID = 0
	s.proxy = nil
	//s.route = nil
	//s.cluster = nil
	s.element = nil
	s.timeout = nil
	//s.retryState = nil
	//s.requestInfo = nil
	s.responseSender = nil
	s.upstreamRequest.downStream = nil
	s.upstreamRequest.requestSender = nil
	s.upstreamRequest.proxy = nil
	s.upstreamRequest.upstreamRespHeaders = nil
	s.upstreamRequest = nil
	s.perRetryTimer = nil
	s.responseTimer = nil
	s.downstreamRespHeaders = nil
	s.downstreamReqDataBuf = nil
	s.downstreamReqTrailers = nil
	s.downstreamRespHeaders = nil
	s.downstreamRespDataBuf = nil
	s.downstreamRespTrailers = nil
	s.senderFilters = s.senderFilters[:0]
	s.receiverFilters = s.receiverFilters[:0]
}

func (s *downStream) giveStream() {
	if atomic.LoadUint32(&s.reuseBuffer) != 1 {
		return
	}
	if atomic.LoadUint32(&s.upstreamReset) == 1 || atomic.LoadUint32(&s.downstreamReset) == 1 {
		return
	}
	// reset downstreamReqBuf
	if s.downstreamReqDataBuf != nil {
		if e := buffer.PutIoBuffer(s.downstreamReqDataBuf); e != nil {
			log.Println("PutIoBuffer error:", e)
		}
	}

	// Give buffers to bufferPool
	if ctx := buffer.PoolContext(s.context); ctx != nil {
		ctx.Give()
	}
}

func (s *downStream) processDone() bool {
	return s.upstreamProcessDone || atomic.LoadUint32(&s.downstreamReset) == 1
}
