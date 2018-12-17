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
	"bufio"
	"fmt"
	"io"
	"strike/utils"
	"sync"
)

var (
	responseBodyPool Pool
	bufioWriterPool  sync.Pool
	statsWriterPool  sync.Pool
)

const maxSmallFileSize = 2 * 4096

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

func (resp *Response) Reset() {
	resp.Header.Reset()
	resp.resetSkipHeader()
	resp.SkipBody = false
}

func (resp *Response) resetSkipHeader() {
	resp.ResetBody()
}

func (resp *Response) ResetBody() {
	resp.closeBodyStream()
	if resp.body != nil {
		if resp.keepBodyBuffer {
			resp.body.Reset()
		} else {
			responseBodyPool.Put(resp.body)
			resp.body = nil
		}
	}
}

func (resp *Response) mustSkipBody() bool {
	return resp.SkipBody || resp.Header.mustSkipContentLength()
}

func (resp *Response) writeBodyStream(w *bufio.Writer, sendBody bool) error {
	var err error

	contentLength := resp.Header.ContentLength()
	if contentLength < 0 {
		lrSize := limitedReaderSize(resp.bodyStream)
		if lrSize >= 0 {
			contentLength = int(lrSize)
			if int64(contentLength) != lrSize {
				contentLength = -1
			}
			if contentLength >= 0 {
				resp.Header.SetContentLength(contentLength)
			}
		}
	}
	if contentLength >= 0 {
		if err = resp.Header.Write(w); err == nil && sendBody {
			err = writeBodyFixedSize(w, resp.bodyStream, int64(contentLength))
		}
	} else {
		resp.Header.SetContentLength(-1)
		if err = resp.Header.Write(w); err == nil && sendBody {
			err = writeBodyChunked(w, resp.bodyStream)
		}
	}
	err1 := resp.closeBodyStream()
	if err == nil {
		err = err1
	}
	return err
}

func (resp *Response) bodyBytes() []byte {
	if resp.body == nil {
		return nil
	}
	return resp.body.B
}

func (resp *Response) Write(w *bufio.Writer) error {
	sendBody := !resp.mustSkipBody()

	if resp.bodyStream != nil {
		return resp.writeBodyStream(w, sendBody)
	}

	body := resp.bodyBytes()
	bodyLen := len(body)
	if sendBody || bodyLen > 0 {
		resp.Header.SetContentLength(bodyLen)
	}
	if err := resp.Header.Write(w); err != nil {
		return err
	}
	if sendBody {
		if _, err := w.Write(body); err != nil {
			return err
		}
	}
	return nil
}

func (resp *Response) SetStatusCode(statusCode int) {
	resp.Header.SetStatusCode(statusCode)
}

func (resp *Response) WriteTo(w io.Writer) (int64, error) {
	return writeBufio(resp, w)
}

func writeBufio(hw httpWriter, w io.Writer) (int64, error) {
	sw := acquireStatsWriter(w)
	bw := acquireBufioWriter(sw)
	err1 := hw.Write(bw)
	err2 := bw.Flush()
	releaseBufioWriter(bw)
	n := sw.bytesWritten
	releaseStatsWriter(sw)

	err := err1
	if err == nil {
		err = err2
	}
	return n, err
}

type responseBodyWriter struct {
	r *Response
}

func (w *responseBodyWriter) Write(p []byte) (int, error) {
	w.r.AppendBody(p)
	return len(p), nil
}

type httpWriter interface {
	Write(w *bufio.Writer) error
}

func acquireStatsWriter(w io.Writer) *statsWriter {
	v := statsWriterPool.Get()
	if v == nil {
		return &statsWriter{
			w: w,
		}
	}
	sw := v.(*statsWriter)
	sw.w = w
	return sw
}

func releaseStatsWriter(sw *statsWriter) {
	sw.w = nil
	sw.bytesWritten = 0
	statsWriterPool.Put(sw)
}

func acquireBufioWriter(w io.Writer) *bufio.Writer {
	v := bufioWriterPool.Get()
	if v == nil {
		return bufio.NewWriter(w)
	}
	bw := v.(*bufio.Writer)
	bw.Reset(w)
	return bw
}

func releaseBufioWriter(bw *bufio.Writer) {
	bufioWriterPool.Put(bw)
}

type statsWriter struct {
	w            io.Writer
	bytesWritten int64
}

func (w *statsWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.bytesWritten += int64(n)
	return n, err
}

func limitedReaderSize(r io.Reader) int64 {
	lr, ok := r.(*io.LimitedReader)
	if !ok {
		return -1
	}
	return lr.N
}

func writeBodyChunked(w *bufio.Writer, r io.Reader) error {
	vbuf := copyBufPool.Get()
	buf := vbuf.([]byte)

	var err error
	var n int
	for {
		n, err = r.Read(buf)
		if n == 0 {
			if err == nil {
				panic("BUG: io.Reader returned 0, nil")
			}
			if err == io.EOF {
				if err = writeChunk(w, buf[:0]); err != nil {
					break
				}
				err = nil
			}
			break
		}
		if err = writeChunk(w, buf[:n]); err != nil {
			break
		}
	}

	copyBufPool.Put(vbuf)
	return err
}

func writeBodyFixedSize(w *bufio.Writer, r io.Reader, size int64) error {
	if size > maxSmallFileSize {
		// w buffer must be empty for triggering
		// sendfile path in bufio.Writer.ReadFrom.
		if err := w.Flush(); err != nil {
			return err
		}
	}

	// Unwrap a single limited reader for triggering sendfile path
	// in net.TCPConn.ReadFrom.
	lr, ok := r.(*io.LimitedReader)
	if ok {
		r = lr.R
	}

	n, err := copyZeroAlloc(w, r)

	if ok {
		lr.N -= n
	}

	if n != size && err == nil {
		err = fmt.Errorf("copied %d bytes from body stream instead of %d bytes", n, size)
	}
	return err
}

func writeChunk(w *bufio.Writer, b []byte) error {
	n := len(b)
	utils.WriteHexInt(w, n)
	w.Write(strCRLF)
	w.Write(b)
	_, err := w.Write(strCRLF)
	err1 := w.Flush()
	if err == nil {
		err = err1
	}
	return err
}
