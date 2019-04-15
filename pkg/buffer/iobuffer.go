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

package buffer

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const (
	FirstConnReadTimeout = 5 * time.Second
	ConnReadTimeout      = 10 * time.Millisecond
)

const (
	MinRead      = 1 << 9
	MaxRead      = 1 << 17
	ResetOffMark = -1
	DefaultSize  = 1 << 4
)

var nullByte []byte

var (
	EOF                  = errors.New("EOF")
	ErrTooLarge          = errors.New("io buffer: too large")
	ErrNegativeCount     = errors.New("io buffer: negative count")
	ErrInvalidWriteCount = errors.New("io buffer: invalid write count")
)

// IoBuffer
type ioBuffer struct {
	buf     []byte // contents: buf[off : len(buf)]
	off     int    // read from &buf[off], write to &buf[len(buf)]
	offMark int
	count   int32
	eof     bool

	b *[]byte
}

func (b *ioBuffer) Read(p []byte) (n int, err error) {
	if b.off >= len(b.buf) {
		b.Reset()

		if len(p) == 0 {
			return
		}

		return 0, io.EOF
	}

	n = copy(p, b.buf[b.off:])
	b.off += n

	return
}

func (b *ioBuffer) ReadOnce(r io.Reader) (n int64, err error) {
	var (
		m               int
		e               error
		zeroTime        time.Time
		conn            net.Conn
		loop, ok, first = true, true, true
	)

	if conn, ok = r.(net.Conn); !ok {
		loop = false
	}

	if b.off >= len(b.buf) {
		b.Reset()
	}

	if b.off > 0 && len(b.buf)-b.off < 4*MinRead {
		b.copy(0)
	}

	for {
		if first == false {
			if free := cap(b.buf) - len(b.buf); free < MinRead {
				// not enough space at end
				if b.off+free < MinRead {
					// not enough space using beginning of buffer;
					// double buffer capacity
					b.copy(MinRead)
				} else {
					b.copy(0)
				}
			}
		}

		l := cap(b.buf) - len(b.buf)

		if conn != nil {
			if first {
				// TODO: support configure
				conn.SetReadDeadline(time.Now().Add(FirstConnReadTimeout))
			} else {
				conn.SetReadDeadline(time.Now().Add(ConnReadTimeout))
			}

			m, e = r.Read(b.buf[len(b.buf):cap(b.buf)])

			// Reset read deadline
			conn.SetReadDeadline(zeroTime)

		} else {
			m, e = r.Read(b.buf[len(b.buf):cap(b.buf)])
		}

		if m > 0 {
			b.buf = b.buf[0 : len(b.buf)+m]
			n += int64(m)
		}

		if e != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() && !first {
				return n, nil
			}
			return n, e
		}

		if l != m {
			loop = false
		}

		if n > MaxRead {
			loop = false
		}

		if !loop {
			break
		}

		first = false
	}

	return n, nil
}

func (b *ioBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.off >= len(b.buf) {
		b.Reset()
	}

	for {
		if free := cap(b.buf) - len(b.buf); free < MinRead {
			// not enough space at end
			if b.off+free < MinRead {
				// not enough space using beginning of buffer;
				// double buffer capacity
				b.copy(MinRead)
			} else {
				b.copy(0)
			}
		}

		m, e := r.Read(b.buf[len(b.buf):cap(b.buf)])

		b.buf = b.buf[0 : len(b.buf)+m]
		n += int64(m)

		if e == io.EOF {
			break
		}

		if m == 0 {
			break
		}

		if e != nil {
			return n, e
		}
	}

	return
}

func (b *ioBuffer) Write(p []byte) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(p))

	if !ok {
		m = b.grow(len(p))
	}

	return copy(b.buf[m:], p), nil
}

func (b *ioBuffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); l+n <= cap(b.buf) {
		b.buf = b.buf[:l+n]

		return l, true
	}

	return 0, false
}

func (b *ioBuffer) grow(n int) int {
	m := b.Len()

	// If buffer is empty, reset to recover space.
	if m == 0 && b.off != 0 {
		b.Reset()
	}

	// Try to grow by means of a reslice.
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}

	if m+n <= cap(b.buf)/2 {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= cap(b.buf) to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		b.copy(0)
	} else {
		// Not enough space anywhere, we need to allocate.
		b.copy(n)
	}

	// Restore b.off and len(b.buf).
	b.off = 0
	b.buf = b.buf[:m+n]

	return m
}

func (b *ioBuffer) WriteTo(w io.Writer) (n int64, err error) {
	for b.off < len(b.buf) {
		nBytes := b.Len()
		m, e := w.Write(b.buf[b.off:])

		if m > nBytes {
			panic(ErrInvalidWriteCount)
		}

		b.off += m
		n += int64(m)

		if e != nil {
			return n, e
		}

		if m == 0 || m == nBytes {
			return n, nil
		}
	}

	return
}

func (b *ioBuffer) Append(data []byte) error {
	if b.off >= len(b.buf) {
		b.Reset()
	}

	dataLen := len(data)

	if free := cap(b.buf) - len(b.buf); free < dataLen {
		// not enough space at end
		if b.off+free < dataLen {
			// not enough space using beginning of buffer;
			// double buffer capacity
			b.copy(dataLen)
		} else {
			b.copy(0)
		}
	}

	m := copy(b.buf[len(b.buf):len(b.buf)+dataLen], data)
	b.buf = b.buf[0 : len(b.buf)+m]

	return nil
}

func (b *ioBuffer) AppendByte(data byte) error {
	return b.Append([]byte{data})
}

func (b *ioBuffer) Peek(n int) []byte {
	if len(b.buf)-b.off < n {
		return nil
	}

	return b.buf[b.off : b.off+n]
}

func (b *ioBuffer) Mark() {
	b.offMark = b.off
}

func (b *ioBuffer) Restore() {
	if b.offMark != ResetOffMark {
		b.off = b.offMark
		b.offMark = ResetOffMark
	}
}

func (b *ioBuffer) Bytes() []byte {
	return b.buf[b.off:]
}

func (b *ioBuffer) Cut(offset int) IoBuffer {
	if b.off+offset > len(b.buf) {
		return nil
	}

	buf := make([]byte, offset)

	copy(buf, b.buf[b.off:b.off+offset])
	b.off += offset
	b.offMark = ResetOffMark

	return &ioBuffer{
		buf: buf,
		off: 0,
	}
}

func (b *ioBuffer) Drain(offset int) {
	if b.off+offset > len(b.buf) {
		return
	}

	b.off += offset
	b.offMark = ResetOffMark
}

func (b *ioBuffer) String() string {
	return string(b.buf[b.off:])
}

func (b *ioBuffer) Len() int {
	return len(b.buf) - b.off
}

func (b *ioBuffer) Cap() int {
	return cap(b.buf)
}

func (b *ioBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
	b.offMark = ResetOffMark
	b.eof = false
}

func (b *ioBuffer) available() int {
	return len(b.buf) - b.off
}

func (b *ioBuffer) Clone() IoBuffer {
	buf := GetIoBuffer(b.Len())
	buf.Write(b.Bytes())

	buf.SetEOF(b.EOF())

	return buf
}

func (b *ioBuffer) Free() {
	b.Reset()
	b.giveSlice()
}

func (b *ioBuffer) Alloc(size int) {
	if b.buf != nil {
		b.Free()
	}
	if size <= 0 {
		size = DefaultSize
	}
	b.b = b.makeSlice(size)
	b.buf = *b.b
	b.buf = b.buf[:0]
}

func (b *ioBuffer) Count(count int32) int32 {
	return atomic.AddInt32(&b.count, count)
}

func (b *ioBuffer) EOF() bool {
	return b.eof
}

func (b *ioBuffer) SetEOF(eof bool) {
	b.eof = eof
}

func (b *ioBuffer) copy(expand int) {
	var newBuf []byte
	var bufp *[]byte

	if expand > 0 {
		bufp = b.makeSlice(2*cap(b.buf) + expand)
		newBuf = *bufp
		copy(newBuf, b.buf[b.off:])
		PutBytes(b.b)
		b.b = bufp
	} else {
		newBuf = b.buf
		copy(newBuf, b.buf[b.off:])
	}
	b.buf = newBuf[:len(b.buf)-b.off]
	b.off = 0
}

func (b *ioBuffer) makeSlice(n int) *[]byte {
	return GetBytes(n)
}

func (b *ioBuffer) giveSlice() {
	if b.b != nil {
		PutBytes(b.b)
		b.b = nil
		b.buf = nullByte
	}
}

func NewIoBuffer(capacity int) IoBuffer {
	buffer := &ioBuffer{
		offMark: ResetOffMark,
		count:   1,
	}
	if capacity <= 0 {
		capacity = DefaultSize
	}
	buffer.b = GetBytes(capacity)
	buffer.buf = (*buffer.b)[:0]
	return buffer
}

func NewIoBufferString(s string) IoBuffer {
	if s == "" {
		return NewIoBuffer(0)
	}
	return &ioBuffer{
		buf:     []byte(s),
		offMark: ResetOffMark,
		count:   1,
	}
}

func NewIoBufferBytes(bytes []byte) IoBuffer {
	if bytes == nil {
		return NewIoBuffer(0)
	}
	return &ioBuffer{
		buf:     bytes,
		offMark: ResetOffMark,
		count:   1,
	}
}

func NewIoBufferEOF() IoBuffer {
	buf := NewIoBuffer(0)
	buf.SetEOF(true)
	return buf
}

func (b *ioBuffer) ReadByte() (bt byte, err error) {
	if b.off >= len(b.buf) {
		b.Reset()
		err = io.EOF
		return
	}

	bt = b.buf[b.off]
	b.off++

	return
}

func (b *ioBuffer) WriteByte(p byte) (err error) {
	m, ok := b.tryGrowByReslice(1)

	if !ok {
		m = b.grow(1)
	}

	b.buf[m] = p
	return
}
