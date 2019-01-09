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

package v1

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/protocol/http/v1"
	"strike/pkg/stream"
	"strike/pkg/types"
	"sync/atomic"
)

var errConnClose = errors.New("connection closed")

func init() {
	stream.Register(protocol.HTTP1, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStreamConnection(context context.Context, connection network.ClientConnection,
	streamConnCallbacks stream.StreamConnectionEventListener,
	callbacks network.ConnectionEventListener) stream.ClientStreamConnection {
	return nil
}

func (f *streamConnFactory) CreateServerStreamConnection(context context.Context, connection network.Connection,
	callbacks stream.ServerStreamConnectionEventListener) stream.ServerStreamConnection {
	return newStreamConnection(context, connection, nil, callbacks)
}

func (f *streamConnFactory) CreateBiDirectStreamConnection(context context.Context, connection network.ClientConnection,
	clientCallbacks stream.StreamConnectionEventListener,
	serverCallbacks stream.ServerStreamConnectionEventListener) stream.ClientStreamConnection {
	return nil
}

func newStreamConnection(context context.Context, connection network.Connection, clientCallbacks stream.StreamConnectionEventListener,
	serverCallbacks stream.ServerStreamConnectionEventListener) stream.ServerStreamConnection {
	sc := &streamConnection{
		context:      context,
		connection:   connection,
		protocol:     protocol.HTTP1,
		codec:        v1.NewCodec(),
		cscCallbacks: clientCallbacks,
		sscCallbacks: serverCallbacks,
	}

	connection.AddConnectionEventListener(sc)
	return sc
}

// protocol.DecodeFilter
// stream.ServerStreamConnection
// stream.StreamConnection
// network.ConnectionEventListener
type streamConnection struct {
	context       context.Context
	protocol      protocol.Protocol
	codec         protocol.Codec
	connection    network.Connection
	connCallbacks network.ConnectionEventListener
	// Client Stream Conn Callbacks
	cscCallbacks stream.StreamConnectionEventListener
	// Server Stream Conn Callbacks
	sscCallbacks stream.ServerStreamConnectionEventListener
}

func (sc *streamConnection) OnDecodeHeader(streamID uint64, headers protocol.HeaderMap, endStream bool) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeData(streamID uint64, data buffer.IoBuffer, endStream bool) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeTrailer(streamID uint64, trailers protocol.HeaderMap) network.FilterStatus {
	return network.Continue
}

// http v1 decode filter use this cb to handle the request
func (sc *streamConnection) OnDecodeDone(streamID uint64, result interface{}) network.FilterStatus {
	if req, ok := result.(*v1.Request); ok {
		srvStream := &serverStream{
			streamBase: streamBase{
				id:      streamID,
				req:     req,
				context: context.WithValue(sc.context, types.ContextKeyStreamID, streamID),
			},
			connection:       sc,
			responseDoneChan: make(chan struct{}),
		}
		srvStream.receiver = sc.sscCallbacks.NewStreamDetect(sc.context, streamID, srvStream)

		if atomic.LoadInt32(&srvStream.readDisableCount) <= 0 {
			srvStream.handleRequest()
		}

		<-srvStream.responseDoneChan
	}

	return network.Continue
}

func (sc *streamConnection) OnDecodeError(err error, headers protocol.HeaderMap) {
	log.Println("decode error:", err)
}

func (sc *streamConnection) Dispatch(buf buffer.IoBuffer) {
	sc.codec.Decode(sc.context, buf, sc)
}

func (sc *streamConnection) Protocol() protocol.Protocol {
	return sc.protocol
}

func (sc *streamConnection) GoAway() {
	panic("implement me")
}

func (sc *streamConnection) Read(p []byte) (n int, err error) {
	//data, ok := <-sc.bufChan
	//
	//// Connection close
	//if !ok {
	//	err = errConnClose
	//	return
	//}
	//
	//n = copy(p, data.Bytes())
	//data.Drain(n)
	//sc.bufChan <- nil
	return
}

func (sc *streamConnection) Write(p []byte) (n int, err error) {
	n = len(p)

	// TODO avoid copy
	buf := buffer.GetIoBuffer(n)
	buf.Write(p)

	err = sc.connection.Write(buf)
	return
}

// connection callbacks
func (sc *streamConnection) OnEvent(event network.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		// clear
	}
}

// stream.Stream
// stream.StreamSender
type streamBase struct {
	id               uint64
	req              *v1.Request
	res              *v1.Response
	context          context.Context
	readDisableCount int32
	receiver         stream.StreamReceiver
	streamCbs        []stream.StreamEventListener
}

// stream.Stream
func (s *streamBase) ID() uint64 {
	return s.id
}

func (s *streamBase) AddEventListener(streamCb stream.StreamEventListener) {
	s.streamCbs = append(s.streamCbs, streamCb)
}

func (s *streamBase) RemoveEventListener(streamCb stream.StreamEventListener) {
	cbIdx := -1

	for i, streamCb := range s.streamCbs {
		if streamCb == streamCb {
			cbIdx = i
			break
		}
	}

	if cbIdx > -1 {
		s.streamCbs = append(s.streamCbs[:cbIdx], s.streamCbs[cbIdx+1:]...)
	}
}

func (s *streamBase) ResetStream(reason stream.StreamResetReason) {
	for _, cb := range s.streamCbs {
		cb.OnResetStream(reason)
	}
}

// stream.StreamSender
// stream.Stream
type serverStream struct {
	streamBase

	connection       *streamConnection
	responseDoneChan chan struct{}
}

// stream.StreamSender
func (s *serverStream) AppendHeaders(context context.Context, headerIn protocol.HeaderMap, endStream bool) error {
	if s.res == nil {
		s.res = v1.AcquireResponse()
	}
	if status, ok := headerIn.Get(types.HeaderStatus); ok {
		headerIn.Del(types.HeaderStatus)

		statusCode, _ := strconv.Atoi(status)
		s.res.SetStatusCode(statusCode)
	}
	if endStream {
		s.endStream()
	}
	return nil
}

func (s *serverStream) AppendData(context context.Context, data buffer.IoBuffer, endStream bool) error {

	return nil
}

func (s *serverStream) AppendTrailers(context context.Context, trailers protocol.HeaderMap) error {
	s.endStream()
	return nil
}

func (s *serverStream) GetStream() stream.Stream {
	return s
}

func (s *serverStream) ReadDisable(disable bool) {
	if disable {
		atomic.AddInt32(&s.readDisableCount, 1)
	} else {
		newCount := atomic.AddInt32(&s.readDisableCount, -1)

		if newCount <= 0 {
			s.handleRequest()
		}
	}
}

func (s *serverStream) handleRequest() {
	if s.req == nil {
		return
	}
	header := decodeReqHeader(s.req.Header)
	header[protocol.StrikeHeaderHostKey] = string(s.req.Header.Host())
	header[protocol.IstioHeaderHostKey] = string(s.req.Header.Host())
	header[protocol.StrikeHeaderMethod] = string(s.req.Header.Method())
	header[protocol.StrikeHeaderPathKey] = string(s.req.RequestURI())

	noBody := s.req.Header.NoBody()

	s.receiver.OnReceiveHeaders(s.context, protocol.CommonHeader(header), noBody)

	if !noBody {
		buf := buffer.NewIoBufferBytes(s.req.Body())
		s.receiver.OnReceiveData(s.context, buf, true)
	}
}

func decodeReqHeader(in v1.RequestHeader) (out map[string]string) {
	out = make(map[string]string, in.Len())

	in.VisitAll(func(key, value []byte) {
		out[string(key)] = string(value)
	})

	return
}

func (s *serverStream) endStream() {
	s.doSend()
	close(s.responseDoneChan)

	if s.req != nil {
		v1.ReleaseRequest(s.req)
	}
	if s.res != nil {
		v1.ReleaseResponse(s.res)
	}
}

func (s *serverStream) doSend() {
	s.res.WriteTo(s.connection)
}

// stream.ClientStreamConnection
type clientStreamConnection struct {
	streamConnection

	stream              *clientStream
	connCallbacks       network.ConnectionEventListener
	streamConnCallbacks stream.StreamConnectionEventListener
}

type clientStream struct {
	streamBase

	connection *clientStreamConnection
}
