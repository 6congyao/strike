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

package v1

import (
	"context"
	"errors"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"sync"
)

var (
	requestPool  sync.Pool
	responsePool sync.Pool
)

func AcquireRequest() *Request {
	v := requestPool.Get()
	if v == nil {
		return &Request{}
	}
	return v.(*Request)
}

func ReleaseRequest(req *Request) {
	req.Reset()
	requestPool.Put(req)
}

func AcquireResponse() *Response {
	v := responsePool.Get()
	if v == nil {
		return &Response{}
	}
	return v.(*Response)
}

func ReleaseResponse(resp *Response) {
	resp.Reset()
	responsePool.Put(resp)
}

var ErrBodyTooLarge = errors.New("body size exceeds the given limit")

type codec struct {
}

func NewCodec() protocol.Codec {
	return &codec{}
}

func (c *codec) Encode(ctx context.Context, model interface{}) (buffer.IoBuffer, error) {
	panic("implement me")
}

func (c *codec) Decode(ctx context.Context, data buffer.IoBuffer, filter protocol.DecodeFilter) {
	// todo: loop then data.Drain()
	if data.Len() > 0 {
		req := AcquireRequest()

		err := readLimitBody(data.Bytes(), req, 40000)
		if err != nil {
			filter.OnDecodeError(err, nil)
			return
		}
		streamID := protocol.GenerateIDString()
		// notify
		filter.OnDecodeDone(streamID, req)
	}
}

func readLimitBody(data []byte, req *Request, maxRequestBodySize int) error {
	n, err := parseHeader(data, &req.Header)
	if err != nil {
		return err
	}

	if req.Header.NoBody() {
		return nil
	}

	return continueReadBody(data, n, req, maxRequestBodySize)
}

func parseHeader(data []byte, header *RequestHeader) (int, error) {
	header.ResetSkipNormalize()

	return header.Parse(data)
}

func continueReadBody(data []byte, offset int, req *Request, maxRequestBodySize int) error {
	data = data[offset:]
	contentLength := req.Header.ContentLength()
	if contentLength > 0 {
		if maxRequestBodySize > 0 && contentLength > maxRequestBodySize {
			return ErrBodyTooLarge
		}
	}

	if contentLength == -2 {
		// identity body has no sense for http requests, since
		// the end of body is determined by connection close.
		// So just ignore request body for requests without
		// 'Content-Length' and 'Transfer-Encoding' headers.
		req.Header.SetContentLength(0)
		return nil
	}

	var err error
	bodyBuf := req.bodyBuffer()
	bodyBuf.Reset()
	bodyBuf.B, err = readBody(data, contentLength, bodyBuf.B)
	return err
}

func readBody(data []byte, contentLength int, dst []byte) ([]byte, error) {
	dst = dst[:0]
	if contentLength >= 0 {
		return appendBodyFixedSize(data, dst, contentLength)
	}
	if contentLength == -1 {
		//return readBodyChunked(r, maxBodySize, dst)
	}
	return dst, nil
}

func appendBodyFixedSize(data []byte, dst []byte, n int) ([]byte, error) {
	if n == 0 {
		return dst, nil
	}

	offset := len(dst)
	dstLen := offset + n
	if cap(dst) < dstLen {
		b := make([]byte, round2(dstLen))
		copy(b, dst)
		dst = b
	}
	dst = dst[:dstLen]

	return append(dst[:0], data[:n]...), nil
}

func round2(n int) int {
	if n <= 0 {
		return 0
	}
	n--
	x := uint(0)
	for n > 0 {
		n >>= 1
		x++
	}
	return 1 << x
}
