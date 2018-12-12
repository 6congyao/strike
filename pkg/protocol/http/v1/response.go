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
	"io"
	"strike/utils"
)

var responseBodyPool Pool
type Response struct {
	noCopy utils.NoCopy

	// Response header
	//
	// Copying Header by value is forbidden. Use pointer to Header instead.
	Header ResponseHeader

	bodyStream io.Reader
	w          responseBodyWriter
	body       *ByteBuffer

	// Response.Read() skips reading body if set to true.
	// Use it for reading HEAD responses.
	//
	// Response.Write() skips writing body if set to true.
	// Use it for writing HEAD responses.
	SkipBody bool

	keepBodyBuffer bool
}

func (resp *Response) AppendBody(p []byte) {
	resp.AppendBodyString(utils.B2s(p))
}

// AppendBodyString appends s to response body.
func (resp *Response) AppendBodyString(s string) {
	resp.closeBodyStream()
	resp.bodyBuffer().WriteString(s)
}

func (resp *Response) closeBodyStream() error {
	if resp.bodyStream == nil {
		return nil
	}
	var err error
	if bsc, ok := resp.bodyStream.(io.Closer); ok {
		err = bsc.Close()
	}
	resp.bodyStream = nil
	return err
}

func (resp *Response) bodyBuffer() *ByteBuffer {
	if resp.body == nil {
		resp.body = responseBodyPool.Get()
	}
	return resp.body
}

type responseBodyWriter struct {
	r *Response
}

func (w *responseBodyWriter) Write(p []byte) (int, error) {
	w.r.AppendBody(p)
	return len(p), nil
}