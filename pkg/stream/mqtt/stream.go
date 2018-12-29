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

package mqtt

import (
	"context"
	"log"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/protocol/mqtt"
	"strike/pkg/protocol/mqtt/message"
	"strike/pkg/stream"
)

func init() {
	stream.Register(protocol.MQTT, &streamConnFactory{})
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
	return &streamConnection{
		context:      context,
		connection:   connection,
		protocol:     protocol.MQTT,
		codec:        message.NewCodec(),
		processor:    mqtt.NewProcessor(),
		cscCallbacks: clientCallbacks,
		sscCallbacks: serverCallbacks,
	}
}

// protocol.DecodeFilter
// stream.ServerStreamConnection
// stream.StreamConnection
type streamConnection struct {
	context       context.Context
	protocol      protocol.Protocol
	codec         protocol.Codec
	processor     *mqtt.Processor
	connection    network.Connection
	connCallbacks network.ConnectionEventListener
	// Client Stream Conn Callbacks
	cscCallbacks stream.StreamConnectionEventListener
	// Server Stream Conn Callbacks
	sscCallbacks stream.ServerStreamConnectionEventListener
}

func (sc *streamConnection) OnDecodeHeader(streamID string, headers protocol.HeaderMap, endStream bool) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeData(streamID string, data buffer.IoBuffer, endStream bool) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeTrailer(streamID string, trailers protocol.HeaderMap) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeDone(streamID string, result interface{}) network.FilterStatus {
	if msg, ok := result.(message.Message); ok {
		sc.handleMessage(msg)
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

func (sc *streamConnection) handleMessage(msg message.Message) {
	//todo: doing business with processor
	sc.processor.Process(sc.context, msg)
}