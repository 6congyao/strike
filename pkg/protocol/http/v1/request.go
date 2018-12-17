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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"strike/utils"
	"sync"
)

var requestBodyPool Pool

var copyBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// Request represents HTTP request.
//
// It is forbidden copying Request instances. Create new instances
// and use CopyTo instead.
//
// Request instance MUST NOT be used from concurrently running goroutines.
type Request struct {
	noCopy utils.NoCopy

	// Request header
	//
	// Copying Header by value is forbidden. Use pointer to Header instead.
	Header RequestHeader

	uri      URI
	postArgs utils.Args

	bodyStream io.Reader
	w          requestBodyWriter
	body       *ByteBuffer

	multipartForm         *multipart.Form
	multipartFormBoundary string

	// Group bool members in order to reduce Request object size.
	parsedURI      bool
	parsedPostArgs bool

	keepBodyBuffer bool

	isTLS bool
}

// URI returns request URI
func (req *Request) URI() *URI {
	req.parseURI()
	return &req.uri
}

func (req *Request) parseURI() {
	if req.parsedURI {
		return
	}
	req.parsedURI = true

	req.uri.parseQuick(req.Header.RequestURI(), &req.Header, req.isTLS)
}

func (req *Request) PostArgs() *utils.Args {
	req.parsePostArgs()
	return &req.postArgs
}

func (req *Request) parsePostArgs() {
	if req.parsedPostArgs {
		return
	}
	req.parsedPostArgs = true

	if !bytes.HasPrefix(req.Header.ContentType(), strPostArgsContentType) {
		return
	}
	req.postArgs.ParseBytes(req.bodyBytes())
}

func (req *Request) Reset() {
	req.Header.Reset()
	req.resetSkipHeader()
}

func (req *Request) resetSkipHeader() {
	req.ResetBody()
	req.uri.Reset()
	req.parsedURI = false
	req.postArgs.Reset()
	req.parsedPostArgs = false
	req.isTLS = false
}

func (req *Request) Body() []byte {
	if req.bodyStream != nil {
		bodyBuf := req.bodyBuffer()
		bodyBuf.Reset()
		_, err := copyZeroAlloc(bodyBuf, req.bodyStream)
		req.closeBodyStream()
		if err != nil {
			bodyBuf.SetString(err.Error())
		}
	} else if req.onlyMultipartForm() {
		body, err := marshalMultipartForm(req.multipartForm, req.multipartFormBoundary)
		if err != nil {
			return []byte(err.Error())
		}
		return body
	}
	return req.bodyBytes()
}

func (req *Request) onlyMultipartForm() bool {
	return req.multipartForm != nil && (req.body == nil || len(req.body.B) == 0)
}

func marshalMultipartForm(f *multipart.Form, boundary string) ([]byte, error) {
	var buf ByteBuffer
	if err := WriteMultipartForm(&buf, f, boundary); err != nil {
		return nil, err
	}
	return buf.B, nil
}

func WriteMultipartForm(w io.Writer, f *multipart.Form, boundary string) error {
	// Do not care about memory allocations here, since multipart
	// form processing is slooow.
	if len(boundary) == 0 {
		panic("BUG: form boundary cannot be empty")
	}

	mw := multipart.NewWriter(w)
	if err := mw.SetBoundary(boundary); err != nil {
		return fmt.Errorf("cannot use form boundary %q: %s", boundary, err)
	}

	// marshal values
	for k, vv := range f.Value {
		for _, v := range vv {
			if err := mw.WriteField(k, v); err != nil {
				return fmt.Errorf("cannot write form field %q value %q: %s", k, v, err)
			}
		}
	}

	// marshal files
	for k, fvv := range f.File {
		for _, fv := range fvv {
			vw, err := mw.CreateFormFile(k, fv.Filename)
			if err != nil {
				return fmt.Errorf("cannot create form file %q (%q): %s", k, fv.Filename, err)
			}
			fh, err := fv.Open()
			if err != nil {
				return fmt.Errorf("cannot open form file %q (%q): %s", k, fv.Filename, err)
			}
			if _, err = copyZeroAlloc(vw, fh); err != nil {
				return fmt.Errorf("error when copying form file %q (%q): %s", k, fv.Filename, err)
			}
			if err = fh.Close(); err != nil {
				return fmt.Errorf("cannot close form file %q (%q): %s", k, fv.Filename, err)
			}
		}
	}

	if err := mw.Close(); err != nil {
		return fmt.Errorf("error when closing multipart form writer: %s", err)
	}

	return nil
}

// ResetBody resets request body.
func (req *Request) ResetBody() {
	req.RemoveMultipartFormFiles()
	req.closeBodyStream()
	if req.body != nil {
		if req.keepBodyBuffer {
			req.body.Reset()
		} else {
			requestBodyPool.Put(req.body)
			req.body = nil
		}
	}
}

func (req *Request) RemoveMultipartFormFiles() {
	if req.multipartForm != nil {
		// Do not check for error, since these files may be deleted or moved
		// to new places by user code.
		req.multipartForm.RemoveAll()
		req.multipartForm = nil
	}
	req.multipartFormBoundary = ""
}

//func (req *Request) MultipartForm() (*multipart.Form, error) {
//	if req.multipartForm != nil {
//		return req.multipartForm, nil
//	}
//
//	req.multipartFormBoundary = string(req.Header.MultipartFormBoundary())
//	if len(req.multipartFormBoundary) == 0 {
//		return nil, ErrNoMultipartForm
//	}
//
//	ce := req.Header.peek(strContentEncoding)
//	body := req.bodyBytes()
//	if bytes.Equal(ce, strGzip) {
//		// Do not care about memory usage here.
//		var err error
//		if body, err = AppendGunzipBytes(nil, body); err != nil {
//			return nil, fmt.Errorf("cannot gunzip request body: %s", err)
//		}
//	} else if len(ce) > 0 {
//		return nil, fmt.Errorf("unsupported Content-Encoding: %q", ce)
//	}
//
//	f, err := readMultipartForm(bytes.NewReader(body), req.multipartFormBoundary, len(body), len(body))
//	if err != nil {
//		return nil, err
//	}
//	req.multipartForm = f
//	return f, nil
//}

// SetHost sets host for the request.
func (req *Request) SetHost(host string) {
	req.URI().SetHost(host)
}

// SetHostBytes sets host for the request.
func (req *Request) SetHostBytes(host []byte) {
	req.URI().SetHostBytes(host)
}

// Host returns the host for the given request.
func (req *Request) Host() []byte {
	return req.URI().Host()
}

// SetRequestURI sets RequestURI.
func (req *Request) SetRequestURI(requestURI string) {
	req.Header.SetRequestURI(requestURI)
	req.parsedURI = false
}

// SetRequestURIBytes sets RequestURI.
func (req *Request) SetRequestURIBytes(requestURI []byte) {
	req.Header.SetRequestURIBytes(requestURI)
	req.parsedURI = false
}

// RequestURI returns request's URI.
func (req *Request) RequestURI() []byte {
	if req.parsedURI {
		requestURI := req.uri.RequestURI()
		req.SetRequestURIBytes(requestURI)
	}
	return req.Header.RequestURI()
}

// ConnectionClose returns true if 'Connection: close' header is set.
func (req *Request) ConnectionClose() bool {
	return req.Header.ConnectionClose()
}

// SetConnectionClose sets 'Connection: close' header.
func (req *Request) SetConnectionClose() {
	req.Header.SetConnectionClose()
}

func (req *Request) SetBodyStream(bodyStream io.Reader, bodySize int) {
	req.ResetBody()
	req.bodyStream = bodyStream
	req.Header.SetContentLength(bodySize)
}

func (req *Request) IsBodyStream() bool {
	return req.bodyStream != nil
}

func (req *Request) BodyWriter() io.Writer {
	req.w.r = req
	return &req.w
}

func (req *Request) bodyBytes() []byte {
	if req.body == nil {
		return nil
	}
	return req.body.B
}

func (req *Request) bodyBuffer() *ByteBuffer {
	if req.body == nil {
		req.body = requestBodyPool.Get()
	}
	return req.body
}

func (req *Request) closeBodyStream() error {
	if req.bodyStream == nil {
		return nil
	}
	var err error
	if bsc, ok := req.bodyStream.(io.Closer); ok {
		err = bsc.Close()
	}
	req.bodyStream = nil
	return err
}

func (req *Request) ReleaseBody(size int) {
	if cap(req.body.B) > size {
		req.closeBodyStream()
		req.body = nil
	}
}

func (req *Request) SwapBody(body []byte) []byte {
	bb := req.bodyBuffer()

	if req.bodyStream != nil {
		bb.Reset()
		_, err := copyZeroAlloc(bb, req.bodyStream)
		req.closeBodyStream()
		if err != nil {
			bb.Reset()
			bb.SetString(err.Error())
		}
	}

	oldBody := bb.B
	bb.B = body
	return oldBody
}

//func (req *Request) Body() []byte {
//	if req.bodyStream != nil {
//		bodyBuf := req.bodyBuffer()
//		bodyBuf.Reset()
//		_, err := copyZeroAlloc(bodyBuf, req.bodyStream)
//		req.closeBodyStream()
//		if err != nil {
//			bodyBuf.SetString(err.Error())
//		}
//	} else if req.onlyMultipartForm() {
//		body, err := marshalMultipartForm(req.multipartForm, req.multipartFormBoundary)
//		if err != nil {
//			return []byte(err.Error())
//		}
//		return body
//	}
//	return req.bodyBytes()
//}

//func (req *Request) AppendBody(p []byte) {
//	req.AppendBodyString(utils.B2s(p))
//}

//func (req *Request) AppendBodyString(s string) {
//	req.RemoveMultipartFormFiles()
//	req.closeBodyStream()
//	req.bodyBuffer().WriteString(s)
//}

// CopyTo copies req contents to dst except of body stream.
func (req *Request) CopyTo(dst *Request) {
	req.copyToSkipBody(dst)
	if req.body != nil {
		dst.bodyBuffer().Set(req.body.B)
	} else if dst.body != nil {
		dst.body.Reset()
	}
}

func (req *Request) copyToSkipBody(dst *Request) {
	dst.Reset()
	req.Header.CopyTo(&dst.Header)

	req.uri.CopyTo(&dst.uri)
	dst.parsedURI = req.parsedURI

	req.postArgs.CopyTo(&dst.postArgs)
	dst.parsedPostArgs = req.parsedPostArgs
	dst.isTLS = req.isTLS

	// do not copy multipartForm - it will be automatically
	// re-created on the first call to MultipartForm.
}

func (req *Request) AppendBody(p []byte) {
	req.AppendBodyString(utils.B2s(p))
}

// AppendBodyString appends s to request body.
func (req *Request) AppendBodyString(s string) {
	req.RemoveMultipartFormFiles()
	req.closeBodyStream()
	req.bodyBuffer().WriteString(s)
}

type requestBodyWriter struct {
	r *Request
}

func (w *requestBodyWriter) Write(p []byte) (int, error) {
	w.r.AppendBody(p)
	return len(p), nil
}

func copyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	vbuf := copyBufPool.Get()
	buf := vbuf.([]byte)
	n, err := io.CopyBuffer(w, r, buf)
	copyBufPool.Put(vbuf)
	return n, err
}
