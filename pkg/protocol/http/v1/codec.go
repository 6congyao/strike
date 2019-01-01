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
	"fmt"
	"io"
	"mime/multipart"
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
const defaultMaxInMemoryFileSize = 16 * 1024 * 1024

type codec struct {
}

func NewCodec() protocol.Codec {
	return &codec{}
}

func (c *codec) Encode(ctx context.Context, model interface{}) (buffer.IoBuffer, error) {
	panic("implement me")
}

func (c *codec) Decode(ctx context.Context, data buffer.IoBuffer, filter protocol.DecodeFilter) {
	for data.Len() > 0 {
		req, err := decodeHttpReq(data)
		if err != nil {
			filter.OnDecodeError(err, nil)
			return
		}
		streamID := protocol.GenerateIDString()
		// notify
		filter.OnDecodeDone(streamID, req)
	}
}

func decodeHttpReq(data buffer.IoBuffer) (req *Request, err error) {
	req = AcquireRequest()
	err = readLimitBody(data, req, 40000)
	return
}

func readLimitBody(r buffer.IoBuffer, req *Request, maxBodySize int) error {
	req.resetSkipHeader()
	err := parseHeader(r, &req.Header)
	if err != nil {
		return err
	}
	if req.Header.NoBody() {
		return nil
	}
	if req.MayContinue() {
		// 'Expect: 100-continue' header found. Let the caller deciding
		// whether to read request body or
		// to return StatusExpectationFailed.
		return nil
	}

	return continueReadBody(r, req, maxBodySize)
}

func parseHeader(r buffer.IoBuffer, h *RequestHeader) error {
	//n := 1
	//for {
	//	err := h.doParse(r, n)
	//	if err == nil {
	//		return nil
	//	}
	//	if err != errNeedMore {
	//		h.ResetSkipNormalize()
	//		return err
	//	}
	//	n = r.Len() + 1
	//}

	err := h.doParse(r, 1)

	if err != nil {
		if err != errNeedMore {
			h.ResetSkipNormalize()
		}
		return err
	}
	return nil
}

func continueReadBody(r buffer.IoBuffer, req *Request, maxBodySize int) (err error) {
	contentLength := req.Header.ContentLength()
	if contentLength > 0 {
		if maxBodySize > 0 && contentLength > maxBodySize {
			return ErrBodyTooLarge
		}

		// Pre-read multipart form data of known length.
		// This way we limit memory usage for large file uploads, since their contents
		// is streamed into temporary files if file size exceeds defaultMaxInMemoryFileSize.
		req.multipartFormBoundary = string(req.Header.MultipartFormBoundary())
		if len(req.multipartFormBoundary) > 0 && len(req.Header.peek(strContentEncoding)) == 0 {
			req.multipartForm, err = readMultipartForm(r, req.multipartFormBoundary, contentLength, defaultMaxInMemoryFileSize)
			if err != nil {
				req.Reset()
			}
			return err
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

	bodyBuf := req.bodyBuffer()
	bodyBuf.Reset()
	bodyBuf.B, err = readBody(r, contentLength, maxBodySize, bodyBuf.B)
	if err != nil {
		req.Reset()
		return err
	}
	req.Header.SetContentLength(len(bodyBuf.B))
	return nil
}

func readBody(r buffer.IoBuffer, contentLength, maxBodySize int, dst []byte) ([]byte, error) {
	dst = dst[:0]
	if contentLength >= 0 {
		if maxBodySize > 0 && contentLength > maxBodySize {
			return dst, ErrBodyTooLarge
		}
		return appendBodyFixedSize(r, dst, contentLength)
	}

	if contentLength == -1 {
		//return readBodyChunked(r, maxBodySize, dst)
	}
	//return readBodyIdentity(r, maxBodySize, dst)
	return nil, nil
}

func appendBodyFixedSize(r buffer.IoBuffer, dst []byte, n int) ([]byte, error) {
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

	nn, err := r.Read(dst[offset:])
	if nn <= 0 {
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return dst[:offset], err
		}
		panic(fmt.Sprintf("BUG: bufio.Read() returned (%d, nil)", nn))
	}
	offset += nn
	if offset == dstLen {
		return dst, nil
	}
	return dst, nil
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

func readMultipartForm(r io.Reader, boundary string, size, maxInMemoryFileSize int) (*multipart.Form, error) {
	// Do not care about memory allocations here, since they are tiny
	// compared to multipart data (aka multi-MB files) usually sent
	// in multipart/form-data requests.

	if size <= 0 {
		panic(fmt.Sprintf("BUG: form size must be greater than 0. Given %d", size))
	}
	lr := io.LimitReader(r, int64(size))
	mr := multipart.NewReader(lr, boundary)
	f, err := mr.ReadForm(int64(maxInMemoryFileSize))
	if err != nil {
		return nil, fmt.Errorf("cannot read multipart/form-data body: %s", err)
	}
	return f, nil
}