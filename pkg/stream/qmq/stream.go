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
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
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
		context:      context,
		connection:   connection,
		protocol:     protocol.MQ,

		cscCallbacks: clientCallbacks,
		sscCallbacks: serverCallbacks,
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
	panic("implement me")
}

func (streamConnection) Protocol() protocol.Protocol {
	panic("implement me")
}
