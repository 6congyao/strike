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

package stream

import (
	"context"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/upstream"
)

// stream.Client
// network.ReadFilter
// stream.StreamConnectionEventListener
type client struct {
	Protocol                      protocol.Protocol
	Connection                    network.ClientConnection
	Host                          upstream.HostInfo
	ClientStreamConnection        ClientStreamConnection
	StreamConnectionEventListener StreamConnectionEventListener
	ConnectedFlag                 bool
}

// Create a StreamClient used as a client to send/receive stream in a connection
func NewStreamClient(ctx context.Context, prot protocol.Protocol, connection network.ClientConnection, host upstream.HostInfo) Client {
	client := &client{
		Protocol:   prot,
		Connection: connection,
		Host:       host,
	}

	if factory, ok := streamFactories[prot]; ok {
		client.ClientStreamConnection = factory.CreateClientStreamConnection(ctx, connection, client, client)
	} else {
		return nil
	}

	if connection != nil {
		connection.AddConnectionEventListener(client)
		connection.FilterManager().AddReadFilter(client)
		connection.SetNoDelay(true)
	}

	return client
}

func (c *client) ConnID() uint64 {
	return c.Connection.ID()
}

func (c *client) Connect(ioEnabled bool) error {
	return c.Connection.Connect(ioEnabled)
}

func (c *client) AddConnectionEventListener(l network.ConnectionEventListener) {
	c.Connection.AddConnectionEventListener(l)
}

func (c *client) ActiveRequestsNum() int {
	return c.ClientStreamConnection.ActiveStreamsNum()
}

func (c *client) SetStreamConnectionEventListener(l StreamConnectionEventListener) {
	c.StreamConnectionEventListener = l
}

func (c *client) NewStream(context context.Context, respReceiver StreamReceiveListener) StreamSender {
	wrapper := &clientStreamReceiverWrapper{
		streamReceiver: respReceiver,
	}

	streamSender := c.ClientStreamConnection.NewStream(context, wrapper)
	wrapper.stream = streamSender.GetStream()

	return streamSender
}

func (c *client) Close() {
	c.Connection.Close(network.NoFlush, network.LocalClose)
}

// types.StreamConnectionEventListener
func (c *client) OnGoAway() {
	c.StreamConnectionEventListener.OnGoAway()
}

// conn callbacks
func (c *client) OnEvent(event network.ConnectionEvent) {
	switch event {
	case network.Connected:
		c.ConnectedFlag = true
	}

	if event.IsClose() || event.ConnectFailure() {
		reason := StreamConnectionFailed

		if c.ConnectedFlag {
			reason = StreamConnectionTermination
		}

		c.ClientStreamConnection.Reset(reason)
	}
}

// network.ReadFilter
// read filter, recv upstream data
func (c *client) OnData(buffer buffer.IoBuffer) network.FilterStatus {
	c.ClientStreamConnection.Dispatch(buffer)

	return network.Stop
}

func (c *client) OnNewConnection() network.FilterStatus {
	return network.Continue
}

func (c *client) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {}

// uniform wrapper to destroy stream at client side
type clientStreamReceiverWrapper struct {
	stream         Stream
	streamReceiver StreamReceiveListener
}

func (w *clientStreamReceiverWrapper) OnReceiveHeaders(ctx context.Context, headers protocol.HeaderMap, endOfStream bool) {
	if endOfStream {
		w.stream.DestroyStream()
	}

	w.streamReceiver.OnReceiveHeaders(ctx, headers, endOfStream)
}

func (w *clientStreamReceiverWrapper) OnReceiveData(ctx context.Context, data buffer.IoBuffer, endOfStream bool) {
	if endOfStream {
		w.stream.DestroyStream()
	}

	w.streamReceiver.OnReceiveData(ctx, data, endOfStream)
}

func (w *clientStreamReceiverWrapper) OnReceiveTrailers(ctx context.Context, trailers protocol.HeaderMap) {
	w.stream.DestroyStream()

	w.streamReceiver.OnReceiveTrailers(ctx, trailers)
}

func (w *clientStreamReceiverWrapper) OnDecodeError(ctx context.Context, err error, headers protocol.HeaderMap) {
	w.streamReceiver.OnDecodeError(ctx, err, headers)
}
