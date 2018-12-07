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

// StreamConnection is a connection runs multiple streams
type StreamConnection interface {
	// Dispatch incoming data
	// On data read scenario, it connects connection and stream by dispatching read buffer to stream,
	// stream uses protocol decode data, and popup event to controller
	Dispatch(buf buffer.IoBuffer)

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
	OnReceiveData(ctx context.Context, data buffer.IoBuffer, endOfStream bool)

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
	AppendData(ctx context.Context, data buffer.IoBuffer, endStream bool) error

	// Append trailers, implicitly ends the stream.
	AppendTrailers(ctx context.Context, trailers protocol.HeaderMap) error

	// Get related stream
	GetStream() Stream
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

// StreamHeadersFilterStatus type
type StreamHeadersFilterStatus string

// StreamHeadersFilterStatus types
const (
	// Continue filter chain iteration.
	StreamHeadersFilterContinue StreamHeadersFilterStatus = "Continue"
	// Do not iterate to next iterator. Filter calls continueDecoding to continue.
	StreamHeadersFilterStop StreamHeadersFilterStatus = "Stop"
)

// StreamDataFilterStatus type
type StreamDataFilterStatus string

// StreamDataFilterStatus types
const (
	// Continue filter chain iteration
	StreamDataFilterContinue StreamDataFilterStatus = "Continue"
	// Do not iterate to next iterator, and buffer body data in controller for later use
	StreamDataFilterStop StreamDataFilterStatus = "Stop"
	// Do not iterate to next iterator, and buffer body data in controller for later use
	StreamDataFilterStopAndBuffer StreamDataFilterStatus = "StopAndBuffer"
)

// StreamTrailersFilterStatus type
type StreamTrailersFilterStatus string

// StreamTrailersFilterStatus types
const (
	// Continue filter chain iteration
	StreamTrailersFilterContinue StreamTrailersFilterStatus = "Continue"
	// Do not iterate to next iterator. Filter calls continueDecoding to continue.
	StreamTrailersFilterStop StreamTrailersFilterStatus = "Stop"
)

// StreamFilterChainFactory adds filter into callbacks
type StreamFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks StreamFilterChainFactoryCallbacks)
}

// StreamFilterChainFactoryCallbacks is called in StreamFilterChainFactory
type StreamFilterChainFactoryCallbacks interface {
	AddStreamSenderFilter(filter StreamSenderFilter)

	AddStreamReceiverFilter(filter StreamReceiverFilter)
}

// StreamSenderFilter is a stream sender filter
type StreamSenderFilter interface {
	StreamFilterBase

	// AppendHeaders encodes headers
	// endStream supplies whether this is a header only request/response
	AppendHeaders(headers protocol.HeaderMap, endStream bool) StreamHeadersFilterStatus

	// AppendData encodes data
	// endStream supplies whether this is the last data
	AppendData(buf buffer.IoBuffer, endStream bool) StreamDataFilterStatus

	// AppendTrailers encodes trailers, implicitly ending the stream
	AppendTrailers(trailers protocol.HeaderMap) StreamTrailersFilterStatus

	// SetEncoderFilterCallbacks sets the StreamSenderFilterCallbacks
	SetEncoderFilterCallbacks(cb StreamSenderFilterCallbacks)
}

// StreamReceiverFilter is a StreamFilterBase wrapper
type StreamReceiverFilter interface {
	StreamFilterBase

	// OnDecodeHeaders is called with decoded headers
	// endStream supplies whether this is a header only request/response
	OnDecodeHeaders(headers protocol.HeaderMap, endStream bool) StreamHeadersFilterStatus

	// OnDecodeData is called with a decoded data
	// endStream supplies whether this is the last data
	OnDecodeData(buf buffer.IoBuffer, endStream bool) StreamDataFilterStatus

	// OnDecodeTrailers is called with decoded trailers, implicitly ending the stream
	OnDecodeTrailers(trailers protocol.HeaderMap) StreamTrailersFilterStatus

	// SetDecoderFilterCallbacks sets decoder filter callbacks
	SetDecoderFilterCallbacks(cb StreamReceiverFilterCallbacks)
}

// StreamFilterCallbacks is called by stream filter to interact with underlying stream
type StreamFilterCallbacks interface {
	// Connection returns the originating connection
	Connection() network.Connection

	// ResetStream resets the underlying stream
	ResetStream()

	// Route returns a route for current stream
	//Route() Route

	// StreamID returns stream id
	StreamID() string

	// RequestInfo returns request info related to the stream
	//RequestInfo() RequestInfo
}

// StreamSenderFilterCallbacks is a StreamFilterCallbacks wrapper
type StreamSenderFilterCallbacks interface {
	StreamFilterCallbacks

	// ContinueEncoding continue iterating through the filter chain with buffered headers and body data
	ContinueEncoding()

	// EncodingBuffer returns data buffered by this filter or previous ones in the filter chain
	EncodingBuffer() buffer.IoBuffer

	// AddEncodedData adds buffered body data
	AddEncodedData(buf buffer.IoBuffer, streamingFilter bool)

	// SetEncoderBufferLimit sets the buffer limit
	SetEncoderBufferLimit(limit uint32)

	// EncoderBufferLimit returns buffer limit
	EncoderBufferLimit() uint32
}

// StreamReceiverFilterCallbacks add additional callbacks that allow a decoding filter to restart
// decoding if they decide to hold data
type StreamReceiverFilterCallbacks interface {
	StreamFilterCallbacks

	// ContinueDecoding continue iterating through the filter chain with buffered headers and body data
	// It can only be called if decode process has been stopped by current filter, using StopIteration from decodeHeaders() or StopIterationAndBuffer or StopIterationNoBuffer from decodeData()
	// The controller will dispatch headers and any buffered body data to the next filter in the chain.
	ContinueDecoding()

	// DecodingBuffer returns data buffered by this filter or previous ones in the filter chain,
	// if nothing has been buffered, returns nil
	DecodingBuffer() buffer.IoBuffer

	// AddDecodedData add s buffered body data
	AddDecodedData(buf buffer.IoBuffer, streamingFilter bool)

	// AppendHeaders is called with headers to be encoded, optionally indicating end of stream
	// Filter uses this function to send out request/response headers of the stream
	// endStream supplies whether this is a header only request/response
	AppendHeaders(headers protocol.HeaderMap, endStream bool)

	// AppendData is called with data to be encoded, optionally indicating end of stream.
	// Filter uses this function to send out request/response data of the stream
	// endStream supplies whether this is the last data
	AppendData(buf buffer.IoBuffer, endStream bool)

	// AppendTrailers is called with trailers to be encoded, implicitly ends the stream.
	// Filter uses this function to send out request/response trailers of the stream
	AppendTrailers(trailers protocol.HeaderMap)

	// SetDecoderBufferLimit sets the buffer limit for decoder filters
	SetDecoderBufferLimit(limit uint32)

	// DecoderBufferLimit returns the decoder buffer limit
	DecoderBufferLimit() uint32
	// SendHijackReply is called when the filter will response directly
	SendHijackReply(code int, headers protocol.HeaderMap)
}

// PoolFailureReason type
type PoolFailureReason string

// PoolFailureReason types
const (
	Overflow          PoolFailureReason = "Overflow"
	ConnectionFailure PoolFailureReason = "ConnectionFailure"
)

//  ConnectionPool is a connection pool interface to extend various of protocols
type ConnectionPool interface {
	Protocol() protocol.Protocol

	NewStream(ctx context.Context, streamID string, responseDecoder StreamReceiver, cb PoolEventListener) Cancellable

	Close()
}

//todo
type PoolEventListener interface {
	OnFailure(streamID string, reason PoolFailureReason, host interface{})

	OnReady(streamID string, requestEncoder StreamSender, host interface{})
}

type Cancellable interface {
	Cancel()
}
