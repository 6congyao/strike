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
	AUTO      Protocol = "Auto"
	MQ        Protocol = "MQ"
	HTTP1     Protocol = "Http1"
	HTTP2     Protocol = "Http2"
	MQTT      Protocol = "Mqtt"
	Xprotocol Protocol = "X"
)

// header direction definition
const (
	Request  = "Request"
	Response = "Response"
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

// Clone used to deep copy header's map
func (h CommonHeader) Clone() HeaderMap {
	copy := make(map[string]string)

	for k, v := range h {
		copy[k] = v
	}

	return CommonHeader(copy)
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
	// Decoder decodes data to model
	Decoder
}

// DecodeFilter is a filter used by Stream to receive decode events
type DecodeFilter interface {
	// OnDecodeHeader is called on headers decoded
	OnDecodeHeader(streamID uint64, headers HeaderMap, endStream bool) network.FilterStatus

	// OnDecodeData is called on data decoded
	OnDecodeData(streamID uint64, data buffer.IoBuffer, endStream bool) network.FilterStatus

	// OnDecodeTrailer is called on trailers decoded
	OnDecodeTrailer(streamID uint64, trailers HeaderMap) network.FilterStatus

	// OnDecodeError is called when error occurs
	// When error occurring, filter status = stop
	OnDecodeError(err error, headers HeaderMap)

	// OnDecodeDone is called on header+body+trailer decoded
	OnDecodeDone(streamID uint64, result interface{}) network.FilterStatus
}

// Encoder is a encoder interface to extend various of protocols
type Encoder interface {
	// Encode encodes a model to binary data
	// return 1. encoded bytes 2. encode error
	Encode(ctx context.Context, model interface{}) (buffer.IoBuffer, error)

	// EncodeTo encodes a model to binary data, and append into the given buffer
	// This method should be used in term of performance
	// return 1. encoded bytes number 2. encode error
	//EncodeTo(ctx context.Context, model interface{}, buf IoBuffer) (int, error)
}

// Decoder is a decoder interface to extend various of protocols
type Decoder interface {
	// Decode decodes binary data to a model
	// pass sub protocol type to identify protocol format
	// return 1. decoded model(nil if no enough data) 2. decode error
	Decode(ctx context.Context, data buffer.IoBuffer, cb DecodeFilter)
}