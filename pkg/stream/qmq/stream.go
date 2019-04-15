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
	"strike/pkg/protocol/mqtt"
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
	csc := &clientStreamConnection{
		streamConnection: streamConnection{
			context:    context,
			connection: connection,
			protocol:   protocol.MQ,

			cscCallbacks: clientCallbacks,
			sscCallbacks: serverCallbacks,
		},
	}
	if connection != nil {
		connection.AddConnectionEventListener(csc)
	}

	return csc
}

// stream.ServerStreamConnection
// stream.StreamConnection
// network.ConnectionEventListener
// proxy.DownstreamCallbacks
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

func (sc *streamConnection) Dispatch(buf buffer.IoBuffer) {
	panic("implement me")
}

func (sc *streamConnection) GoAway() {
	panic("implement me")
}

func (sc *streamConnection) Protocol() protocol.Protocol {
	return protocol.MQ
}

func (sc *streamConnection) OnEvent(event network.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		// clear
	}
}

func (sc *streamConnection) ActiveStreamsNum() int {
	return 0
}

func (sc *streamConnection) Reset(reason stream.StreamResetReason) {

}

// stream.ClientStreamConnection
type clientStreamConnection struct {
	streamConnection
}

func (csc *clientStreamConnection) NewStream(ctx context.Context, receiver stream.StreamReceiveListener) stream.StreamSender {
	cs := &clientStream{
		streamBase: streamBase{
			id: protocol.GenerateID(),

			context:  ctx,
			receiver: receiver,
		},
		processor: mqtt.NewProcessor(),
	}
	return cs
}

// stream.Stream
// stream.StreamSender
type streamBase struct {
	stream.BaseStream
	id      uint64
	context context.Context
	header  protocol.HeaderMap

	topic string
	msg   []byte

	receiver stream.StreamReceiveListener
}

func (sb *streamBase) ID() uint64 {
	return sb.id
}

// stream.Stream
// stream.StreamSender
type clientStream struct {
	streamBase
	processor *mqtt.Processor
}

func (cs *clientStream) AppendData(ctx context.Context, data buffer.IoBuffer, endStream bool) error {
	cs.msg = data.Bytes()

	if endStream {
		cs.endStream()
	}
	return nil
}

func (cs *clientStream) AppendHeaders(ctx context.Context, headers protocol.HeaderMap, endStream bool) error {
	cs.header = headers
	if path, ok := headers.Get(protocol.StrikeHeaderPathKey); ok {
		cs.topic = strings.TrimLeft(path, "/")
	}

	if endStream {
		cs.endStream()
	}

	return nil
}

func (cs *clientStream) AppendTrailers(ctx context.Context, trailers protocol.HeaderMap) error {
	panic("implement me")
}

func (cs *clientStream) GetStream() stream.Stream {
	return cs
}

func (cs *clientStream) endStream() {
	//if cs.msg != nil {
	// todo: handle MDMP process here
	fmt.Println("client stream handled:", cs.id)
	cs.handleSuccess()
	//}
}

func (cs *clientStream) handleSuccess() {
	raw := make(map[string]string, 5)
	headers := protocol.CommonHeader(raw)
	headers.Set(types.HeaderStatus, strconv.Itoa(types.SuccessCode))
	if method, ok := cs.header.Get(types.HeaderMethod); ok {
		headers.Set(types.HeaderMethod, method)
	}
	if strPacketId, ok := cs.header.Get(types.HeaderPacketID); ok {
		headers.Set(types.HeaderPacketID, strPacketId)
	}
	if username, ok := cs.header.Get(types.HeaderUsername); ok {
		headers.Set(types.HeaderUsername, username)
	}
	cs.receiver.OnReceiveHeaders(cs.context, headers, true)
}

func (cs *clientStream) handleFailure() {
	raw := make(map[string]string, 5)
	headers := protocol.CommonHeader(raw)
	headers.Set(types.HeaderStatus, strconv.Itoa(types.RouterUnavailableCode))
	if method, ok := cs.header.Get(types.HeaderMethod); ok {
		headers.Set(types.HeaderMethod, method)
	}
	cs.receiver.OnReceiveHeaders(cs.context, headers, true)
}
