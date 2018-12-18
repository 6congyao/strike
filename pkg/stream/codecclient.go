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
	"container/list"
	"context"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/upstream"
	"sync"
	"sync/atomic"
)

// stream.CodecClient
// network.ReadFilter
// stream.StreamConnectionEventListener
type codecClient struct {
	Protocol                  protocol.Protocol
	Connection                network.ClientConnection
	Host                      upstream.HostInfo
	Codec                     ClientStreamConnection
	ActiveRequests            *list.List
	AcrMux                    sync.RWMutex
	CodecClientCallbacks      CodecClientCallbacks
	StreamConnectionCallbacks StreamConnectionEventListener
	ConnectedFlag             bool
	RemoteCloseFlag           bool
}

// NewCodecClient
// Create a codecclient used as a client to send/receive stream in a connection
func NewCodecClient(ctx context.Context, prot protocol.Protocol, connection network.ClientConnection, host upstream.HostInfo) CodecClient {
	codecClient := &codecClient{
		Protocol:       prot,
		Connection:     connection,
		Host:           host,
		ActiveRequests: list.New(),
	}

	if factory, ok := streamFactories[prot]; ok {
		codecClient.Codec = factory.CreateClientStreamConnection(ctx, connection, codecClient, codecClient)
	} else {
		return nil
	}

	connection.AddConnectionEventListener(codecClient)
	connection.FilterManager().AddReadFilter(codecClient)
	connection.SetNoDelay(true)

	return codecClient
}

func (c *codecClient) ID() uint64 {
	return c.Connection.ID()
}

func (c *codecClient) AddConnectionCallbacks(cb network.ConnectionEventListener) {
	c.Connection.AddConnectionEventListener(cb)
}

func (c *codecClient) ActiveRequestsNum() int {
	c.AcrMux.RLock()
	defer c.AcrMux.RUnlock()

	return c.ActiveRequests.Len()
}

func (c *codecClient) SetCodecClientCallbacks(cb CodecClientCallbacks) {
	c.CodecClientCallbacks = cb
}

func (c *codecClient) SetCodecConnectionCallbacks(cb StreamConnectionEventListener) {
	c.StreamConnectionCallbacks = cb
}

func (c *codecClient) RemoteClose() bool {
	return c.RemoteCloseFlag
}

func (c *codecClient) NewStream(context context.Context, respDecoder StreamReceiver) StreamSender {
	ar := newActiveRequest(c, respDecoder)
	ar.requestSender = c.Codec.NewStream(context, ar)
	ar.requestSender.GetStream().AddEventListener(ar)

	c.AcrMux.Lock()
	ar.element = c.ActiveRequests.PushBack(ar)
	c.AcrMux.Unlock()

	return ar.requestSender
}

func (c *codecClient) Close() {
	c.Connection.Close(network.NoFlush, network.LocalClose)
}

// types.StreamConnectionEventListener
func (c *codecClient) OnGoAway() {
	c.StreamConnectionCallbacks.OnGoAway()
}

// conn callbacks
func (c *codecClient) OnEvent(event network.ConnectionEvent) {
	switch event {
	case network.Connected:
		c.ConnectedFlag = true
	case network.RemoteClose:
		c.RemoteCloseFlag = true
	}

	if event.IsClose() || event.ConnectFailure() {
		var arNext *list.Element

		c.AcrMux.RLock()
		acReqs := make([]*activeRequest, 0, c.ActiveRequests.Len())
		for ar := c.ActiveRequests.Front(); ar != nil; ar = arNext {
			arNext = ar.Next()
			acReqs = append(acReqs, ar.Value.(*activeRequest))
		}
		c.AcrMux.RUnlock()

		for _, ac := range acReqs {
			reason := StreamConnectionFailed

			if c.ConnectedFlag {
				reason = StreamConnectionTermination
			}
			ac.requestSender.GetStream().ResetStream(reason)
		}
	}
}

// read filter, recv upstream data
func (c *codecClient) OnData(buffer buffer.IoBuffer) network.FilterStatus {
	c.Codec.Dispatch(buffer)

	return network.Stop
}

func (c *codecClient) OnNewConnection() network.FilterStatus {
	return network.Continue
}

func (c *codecClient) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {}

func (c *codecClient) onReset(request *activeRequest, reason StreamResetReason) {
	if c.CodecClientCallbacks != nil {
		c.CodecClientCallbacks.OnStreamReset(reason)
	}

	c.deleteRequest(request)
}

func (c *codecClient) responseDecodeComplete(request *activeRequest) {
	c.deleteRequest(request)
	request.requestSender.GetStream().RemoveEventListener(request)
}

func (c *codecClient) deleteRequest(request *activeRequest) {
	if !atomic.CompareAndSwapUint32(&request.deleted, 0, 1) {
		return
	}

	c.AcrMux.Lock()
	defer c.AcrMux.Unlock()

	c.ActiveRequests.Remove(request.element)

	if c.CodecClientCallbacks != nil {
		c.CodecClientCallbacks.OnStreamDestroy()
	}
}

// StreamEventListener
// StreamDecoderWrapper
type activeRequest struct {
	codecClient      *codecClient
	responseReceiver StreamReceiver
	requestSender    StreamSender
	element          *list.Element
	deleted          uint32
}

func newActiveRequest(codecClient *codecClient, streamDecoder StreamReceiver) *activeRequest {
	return &activeRequest{
		codecClient:      codecClient,
		responseReceiver: streamDecoder,
	}
}

func (r *activeRequest) OnResetStream(reason StreamResetReason) {
	r.codecClient.onReset(r, reason)
}

func (r *activeRequest) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseReceiver.OnReceiveHeaders(context, headers, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) OnReceiveData(context context.Context, data buffer.IoBuffer, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseReceiver.OnReceiveData(context, data, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) {
	r.onPreDecodeComplete()
	r.responseReceiver.OnReceiveTrailers(context, trailers)
	r.onDecodeComplete()
}

func (r *activeRequest) OnDecodeError(context context.Context, err error, headers protocol.HeaderMap) {
}

func (r *activeRequest) onPreDecodeComplete() {
	r.codecClient.responseDecodeComplete(r)
}

func (r *activeRequest) onDecodeComplete() {}