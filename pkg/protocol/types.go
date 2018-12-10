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

package protocol

import (
	"context"
	"strike/pkg/buffer"
	"strike/pkg/network"
)

type Protocol string

// Protocol type definition
const (
	MQ        Protocol = "MQ"
	HTTP1     Protocol = "Http1"
	HTTP2     Protocol = "Http2"
	Xprotocol Protocol = "X"
)

// Host key for routing in Header
const (
	StrikeHeaderHostKey         = "x-strike-host"
	StrikeHeaderPathKey         = "x-strike-path"
	StrikeHeaderQueryStringKey  = "x-strike-querystring"
	StrikeHeaderMethod          = "x-strike-method"
	StrikeOriginalHeaderPathKey = "x-strike-original-path"
	StrikeResponseStatusCode    = "x-strike-response-code"
	IstioHeaderHostKey          = "authority"
)

// CommonHeader wrapper for map[string]string
type CommonHeader map[string]string

// Get value of key
func (h CommonHeader) Get(key string) (value string, ok bool) {
	value, ok = h[key]
	return
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (h CommonHeader) Set(key string, value string) {
	h[key] = value
}

// Del delete pair of specified key
func (h CommonHeader) Del(key string) {
	delete(h, key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h CommonHeader) Range(f func(key, value string) bool) {
	for k, v := range h {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

func (h CommonHeader) ByteSize() uint64 {
	var size uint64

	for k, v := range h {
		size += uint64(len(k) + len(v))
	}
	return size
}

// HeaderMap is a interface to provide operation facade with user-value headers
type HeaderMap interface {
	// Get value of key
	Get(key string) (string, bool)

	// Set key-value pair in header map, the previous pair will be replaced if exists
	Set(key, value string)

	// Del delete pair of specified key
	Del(key string)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(key, value string) bool)

	// ByteSize return size of HeaderMap
	ByteSize() uint64
}

// Codec is a protocols' facade used by Stream
type Codec interface {
	// Encoder is a encoder interface to extend various of protocols
	Encoder
	// Decode decodes data to headers-data-trailers by Stream
	// Stream register a DecodeFilter to receive decode event
	Decode(ctx context.Context, data buffer.IoBuffer, filter DecodeFilter)
}

// DecodeFilter is a filter used by Stream to receive decode events
type DecodeFilter interface {
	// OnDecodeHeader is called on headers decoded
	OnDecodeHeader(streamID string, headers HeaderMap, endStream bool) network.FilterStatus

	// OnDecodeData is called on data decoded
	OnDecodeData(streamID string, data buffer.IoBuffer, endStream bool) network.FilterStatus

	// OnDecodeTrailer is called on trailers decoded
	OnDecodeTrailer(streamID string, trailers HeaderMap) network.FilterStatus

	// OnDecodeError is called when error occurs
	// When error occurring, filter status = stop
	OnDecodeError(err error, headers HeaderMap)

	// OnDecodeDone is called on header+body+trailer decoded
	OnDecodeDone(streamID string, result interface{}) network.FilterStatus
}

// Encoder is a encoder interface to extend various of protocols
type Encoder interface {
	// EncodeHeaders encodes the headers based on it's protocol
	EncodeHeaders(ctx context.Context, headers HeaderMap) (buffer.IoBuffer, error)

	// EncodeData encodes the data based on it's protocol
	EncodeData(ctx context.Context, data buffer.IoBuffer) buffer.IoBuffer

	// EncodeTrailers encodes the trailers based on it's protocol
	EncodeTrailers(ctx context.Context, trailers HeaderMap) buffer.IoBuffer
}
