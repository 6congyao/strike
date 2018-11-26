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
	"strike/pkg/network"
	"strike/pkg/protocol"
)

// StreamResetReason defines the reason why stream reset
type StreamResetReason string

// Group of stream reset reasons
const (
	StreamConnectionTermination StreamResetReason = "ConnectionTermination"
	StreamConnectionFailed      StreamResetReason = "ConnectionFailed"
	StreamLocalReset            StreamResetReason = "StreamLocalReset"
	StreamOverflow              StreamResetReason = "StreamOverflow"
	StreamRemoteReset           StreamResetReason = "StreamRemoteReset"
)

// StreamHeadersFilterStatus type
type StreamHeadersFilterStatus string

// StreamConnection is a connection runs multiple streams
type StreamConnection interface {
	// Dispatch incoming data
	// On data read scenario, it connects connection and stream by dispatching read buffer to stream,
	// stream uses protocol decode data, and popup event to controller
	Dispatch(buffer []byte)

	// Protocol on the connection
	Protocol() protocol.Protocol

	// GoAway sends go away to remote for graceful shutdown
	GoAway()
}

// StreamConnectionEventListener is a stream connection event listener
type StreamConnectionEventListener interface {
	// OnGoAway is called on remote sends 'go away'
	OnGoAway()
}

// StreamReceiver handles request on server scenario, handles response on client scenario.
// Listeners called on decode stream event
type StreamReceiver interface {
	// OnReceiveHeaders is called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnReceiveHeaders(ctx context.Context, headers protocol.HeaderMap, endOfStream bool)

	// OnReceiveData is called with a decoded data
	// endStream supplies whether this is the last data
	OnReceiveData(ctx context.Context, data []byte, endOfStream bool)

	// OnReceiveTrailers is called with a decoded trailers frame, implicitly ends the stream.
	OnReceiveTrailers(ctx context.Context, trailers protocol.HeaderMap)

	// OnDecodeError is called with when exception occurs
	OnDecodeError(ctx context.Context, err error, headers protocol.HeaderMap)
}

// StreamSender encodes protocol stream
// On server scenario, StreamSender handles response
// On client scenario, StreamSender handles request
type StreamSender interface {
	// Append headers
	// endStream supplies whether this is a header only request/response
	AppendHeaders(ctx context.Context, headers protocol.HeaderMap, endStream bool) error

	// Append data
	// endStream supplies whether this is the last data frame
	AppendData(ctx context.Context, data []byte, endStream bool) error

	// Append trailers, implicitly ends the stream.
	AppendTrailers(ctx context.Context, trailers protocol.HeaderMap) error

	// Get related stream
	GetStream() Stream
}

// Stream is a generic protocol stream, it is the core model in stream layer
type Stream interface {
	// AddEventListener adds stream event listener
	AddEventListener(streamEventListener StreamEventListener)

	// RemoveEventListener removes stream event listener
	RemoveEventListener(streamEventListener StreamEventListener)

	// ResetStream rests stream
	// Any registered StreamEventListener.OnResetStream should be called.
	ResetStream(reason StreamResetReason)

	// ReadDisable enable/disable further stream data
	ReadDisable(disable bool)
}

// StreamEventListener is a stream event listener
type StreamEventListener interface {
	// OnResetStream is called on a stream is been reset
	OnResetStream(reason StreamResetReason)
}

// ClientStreamConnection is a client side stream connection.
type ClientStreamConnection interface {
	StreamConnection

	// NewStream creates a new outgoing request stream
	// responseDecoder supplies the decoder listeners on decode event
	// StreamSender supplies the encoder to write the request
	NewStream(ctx context.Context, streamID string, responseDecoder StreamReceiver) StreamSender
}

// ServerStreamConnection is a server side stream connection.
type ServerStreamConnection interface {
	StreamConnection
}

// ServerStreamConnectionEventListener is a stream connection event listener for server connection
type ServerStreamConnectionEventListener interface {
	StreamConnectionEventListener

	// NewStream returns request stream decoder
	NewStream(context context.Context, streamID string, responseEncoder StreamSender) StreamReceiver
}

type ProtocolStreamFactory interface {
	CreateClientStream(context context.Context, connection network.ClientConnection,
		streamConnCallbacks StreamConnectionEventListener,
		callbacks network.ConnectionEventListener) ClientStreamConnection

	CreateServerStream(context context.Context, connection network.Connection,
		callbacks ServerStreamConnectionEventListener) ServerStreamConnection

	CreateBiDirectStream(context context.Context, connection network.ClientConnection,
		clientCallbacks StreamConnectionEventListener,
		serverCallbacks ServerStreamConnectionEventListener) ClientStreamConnection
}

type StreamFilterBase interface {
	OnDestroy()
}
