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

package qmq

import (
	"context"
	"fmt"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	"strings"
)

func init() {
	stream.Register(protocol.MQ, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStreamConnection(context context.Context, connection network.ClientConnection,
	streamConnCallbacks stream.StreamConnectionEventListener,
	callbacks network.ConnectionEventListener) stream.ClientStreamConnection {
	return newStreamConnection(context, connection, streamConnCallbacks, nil)
}

func (f *streamConnFactory) CreateServerStreamConnection(context context.Context, connection network.Connection,
	callbacks stream.ServerStreamConnectionEventListener) stream.ServerStreamConnection {
	return nil
}

func (f *streamConnFactory) CreateBiDirectStreamConnection(context context.Context, connection network.ClientConnection,
	clientCallbacks stream.StreamConnectionEventListener,
	serverCallbacks stream.ServerStreamConnectionEventListener) stream.ClientStreamConnection {
	return nil
}

func newStreamConnection(context context.Context, connection network.Connection, clientCallbacks stream.StreamConnectionEventListener,
	serverCallbacks stream.ServerStreamConnectionEventListener) stream.ClientStreamConnection {
	sc := &streamConnection{
		context:    context,
		connection: connection,
		protocol:   protocol.MQ,

		cscCallbacks: clientCallbacks,
		sscCallbacks: serverCallbacks,
	}
	if connection != nil {
		connection.AddConnectionEventListener(sc)
	}

	return sc
}

// stream.ClientStreamConnection
// stream.StreamConnection
type streamConnection struct {
	context       context.Context
	protocol      protocol.Protocol
	connection    network.Connection
	connCallbacks network.ConnectionEventListener
	// Client Stream Conn Callbacks
	cscCallbacks stream.StreamConnectionEventListener
	// Server Stream Conn Callbacks
	sscCallbacks stream.ServerStreamConnectionEventListener
}

func (streamConnection) Dispatch(buf buffer.IoBuffer) {
	panic("implement me")
}

func (streamConnection) GoAway() {
	panic("implement me")
}

func (streamConnection) NewStream(ctx context.Context, receiver stream.StreamReceiver) stream.StreamSender {
	s := &streamBase{
		id:        protocol.GenerateID(),
		processor: nil,
		context:   ctx,
		receiver:  receiver,
	}
	return s
}

func (streamConnection) Protocol() protocol.Protocol {
	return protocol.MQ
}

func (sc *streamConnection) OnEvent(event network.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		// clear
	}
}

// stream.Stream
// stream.StreamSender
type streamBase struct {
	id        uint64
	processor interface{}
	context   context.Context

	topic string
	msg   []byte

	receiver  stream.StreamReceiver
	streamCbs []stream.StreamEventListener
}

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

func (s *streamBase) ReadDisable(disable bool) {
	panic("unsupported")
}

func (s *streamBase) AppendData(ctx context.Context, data buffer.IoBuffer, endStream bool) error {
	s.msg = data.Bytes()

	if endStream {
		s.endStream()
	}
	return nil
}

func (s *streamBase) AppendHeaders(ctx context.Context, headers protocol.HeaderMap, endStream bool) error {
	path, _ := headers.Get(protocol.StrikeHeaderPathKey)
	s.topic = strings.TrimLeft(path, "/")
	return nil
}

func (s *streamBase) AppendTrailers(ctx context.Context, trailers protocol.HeaderMap) error {
	panic("implement me")
}

func (s *streamBase) GetStream() stream.Stream {
	return s
}

func (s *streamBase) endStream() {
	if s.msg != nil {
		// todo: mq send and give response
		fmt.Println("mq send msg")
		s.handleFailure()
	}
}

func (s *streamBase) handleSuccess() {
	raw := make(map[string]string, 5)
	headers := protocol.CommonHeader(raw)
	headers.Set(types.HeaderStatus, strconv.Itoa(types.SuccessCode))
	s.receiver.OnReceiveHeaders(s.context, headers, true)
}

func (s *streamBase) handleFailure() {
	raw := make(map[string]string, 5)
	headers := protocol.CommonHeader(raw)
	headers.Set(types.HeaderStatus, strconv.Itoa(types.RouterUnavailableCode))
	s.receiver.OnReceiveHeaders(s.context, headers, true)
}