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

package network

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"runtime"
	"runtime/debug"
	"strike/pkg/admin"
	"strike/pkg/buffer"
	strikesync "strike/pkg/sync"
	"strike/pkg/types"
	"sync"
	"sync/atomic"
)

var globalSessionId uint64 = 0
var PipelineReaderPool sync.Pool

const DefaultBufferReadCapacity = 1 << 0

// network.Connection
// admin.Emitter
// admin.EmitterManager
type Session struct {
	id         uint64
	remoteAddr net.Addr
	closeFlag  int32

	closeWithFlush bool
	filterManager FilterManager
	rawc          net.Conn
	connCallbacks []ConnectionEventListener
	emitters      []admin.Emitter

	readBuffer      buffer.IoBuffer
	readTimeout     int64
	bufferLimit     uint32
	//stopChan        chan struct{}
	writeBuffers    net.Buffers
	ioBuffers       []buffer.IoBuffer
	writeBufferChan chan *[]buffer.IoBuffer

	// readLoop/writeLoop goroutine fields:
	internalLoopStarted bool
	internalStopChan    chan struct{}

	startOnce sync.Once
}

func NewSession(ctx context.Context, rawc net.Conn) Connection {
	s := &Session{
		id:               atomic.AddUint64(&globalSessionId, 1),
		remoteAddr:       rawc.RemoteAddr(),
		rawc:             rawc,
		internalStopChan: make(chan struct{}),
		writeBufferChan:  make(chan *[]buffer.IoBuffer, 32),
	}

	s.filterManager = newFilterManager(s)
	return s
}

// start read loop on std net conn
// do nothing if edge conn
func (s *Session) Start(ctx context.Context) {
	if !s.IsClosed() {
		if _, ok := s.RawConn().(net.Conn); ok {
			s.startOnce.Do(func() {
				s.startRWLoop(ctx)
			})
		}
	}
}

func (s *Session) Close(ccType ConnectionCloseType, eventType ConnectionEvent) error {
	if !atomic.CompareAndSwapInt32(&s.closeFlag, 0, 1) {
		return nil
	}

	if reflect.ValueOf(s.RawConn()).IsNil() {
		return nil
	}

	rawc := s.RawConn()

	// shutdown read first
	if conn, ok := rawc.(*net.TCPConn); ok {
		conn.CloseRead()
	}

	if ccType == FlushWrite {
		if s.writeBufLen() > 0 {
			s.closeWithFlush = true

			for {
				bytesSent, err := s.doWrite()

				if err != nil {
					if te, ok := err.(net.Error); !(ok && te.Timeout()) {
						break
					}
				}

				if bytesSent == 0 {
					break
				}
			}
		}
	}

	if s.internalLoopStarted {
		// because close function must be called by one io loop thread, notify another loop here
		close(s.internalStopChan)
		close(s.writeBufferChan)
	}

	if conn, ok := s.RawConn().(net.Conn); ok {
		conn.Close()
	}

	for _, cb := range s.connCallbacks {
		cb.OnEvent(eventType)
	}

	s.emitters = s.emitters[0:0]

	return nil
}

func (c *Session) writeBufLen() (bufLen int) {
	for _, buf := range c.writeBuffers {
		bufLen += len(buf)
	}
	return
}

func (s *Session) IsClosed() bool {
	return atomic.LoadInt32(&s.closeFlag) == 1
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) RemoteAddr() net.Addr {
	return s.remoteAddr
}

func (s *Session) Write(bufs ...buffer.IoBuffer) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Write panic:", r)
		}
	}()

	fs := s.filterManager.OnWrite(bufs)

	if fs == Stop {
		return nil
	}

	select {
	case s.writeBufferChan <- &bufs:
	default:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Write panic:", r)
				}
			}()
			s.writeBufferChan <- &bufs
		}()
	}

	return nil
}

func (s *Session) SetRemoteAddr(addr net.Addr) {
	s.remoteAddr = addr
}

func (s *Session) SetBufferLimit(limit uint32) {
	if limit > 0 {
		s.bufferLimit = limit
	}
}

func (s *Session) BufferLimit() uint32 {
	return s.bufferLimit
}

func (s *Session) AddConnectionEventListener(cb ConnectionEventListener) {
	s.connCallbacks = append(s.connCallbacks, cb)
}

func (s *Session) GetReadBuffer() buffer.IoBuffer {
	return s.readBuffer
}

func (s *Session) FilterManager() FilterManager {
	return s.filterManager
}

func (s *Session) SetNoDelay(enable bool) {
	if s.rawc != nil {
		if rawc, ok := s.rawc.(*net.TCPConn); ok {
			rawc.SetNoDelay(enable)
		}
	}
}

func (s *Session) RawConn() net.Conn {
	return s.rawc
}

func (s *Session) SetReadTimeout(duration int64) {
	s.readTimeout = duration
}

func (s *Session) Emit(topic string, args ...interface{}) error {
	for i, _ := range s.emitters {
		err := s.emitters[i].Emit(topic, args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) AddEmitter(e admin.Emitter) {
	s.emitters = append(s.emitters, e)
}

func (s *Session) doReadConn() (err error) {
	if conn, ok := s.rawc.(net.Conn); ok {
		if s.readBuffer == nil {
			s.readBuffer = buffer.GetIoBuffer(DefaultBufferReadCapacity)
		}

		var bytesRead int64
		bytesRead, err = s.readBuffer.ReadOnce(conn, s.readTimeout)

		if err != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() {
				// run read timeout callback, for keep alive if configured
				for _, cb := range s.connCallbacks {
					cb.OnEvent(OnReadTimeout)
				}
				if bytesRead == 0 {
					return err
				}
			} else if err != io.EOF {
				return err
			}
		}

		s.onRead(bytesRead)
	}
	return
}

func (s *Session) doRead(b []byte) {
	if s.readBuffer == nil {
		s.readBuffer = buffer.GetIoBuffer(DefaultBufferReadCapacity)
	}
	n, _ := s.readBuffer.ReadFrom(bytes.NewBuffer(b))

	//p := s.In.Begin(b)
	//// lazy acquire
	//s.Pr = AcquirePipelineReader(s)
	//pr := s.Pr
	//rbuf := bytes.NewBuffer(p)
	//pr.Rd = rbuf
	//pr.Wr = s

	s.onRead(n)

	//p = p[len(p)-rbuf.Len():]
	//s.In.End(p)
	return
}

func (s *Session) onRead(bytesRead int64) {
	if bytesRead == 0 {
		return
	}
	s.filterManager.OnRead()
}

func (s *Session) startRWLoop(ctx context.Context) {
	s.internalLoopStarted = true

	// async
	// worker pool serves the read loop and the write loop
	wp := ctx.Value(types.ContextKeyWorkerPoolRef).(strikesync.WorkerPool)
	if wp == nil {
		log.Fatalln("Bug: null WorkerPool")
		return
	}

	wp.Serve(func() {
		defer func() {
			if p := recover(); p != nil {
				log.Println("panic:", p)

				debug.PrintStack()

				s.startReadLoop()
			}
		}()

		s.startReadLoop()
		fmt.Println("^^ read loop exit")
	})

	defer func() {
		if p := recover(); p != nil {
			log.Println("panic:", p)

			debug.PrintStack()

			s.startWriteLoop()
		}
	}()

	s.startWriteLoop()
	fmt.Println("^^ write loop exit")
}

func (s *Session) startReadLoop() {
	for {
		// exit loop asap. one receive & one default block will be optimized by go compiler
		select {
		case <-s.internalStopChan:
			return
		default:
		}

		select {
		case <-s.internalStopChan:
			return
		default:
			err := s.doReadConn()
			if err != nil {
				if te, ok := err.(net.Error); ok && te.Timeout() {
					if s.readBuffer != nil && s.readBuffer.Len() == 0 {
						s.readBuffer.Free()
						s.readBuffer.Alloc(DefaultBufferReadCapacity)
					}
					continue
				}
				if err == io.EOF {
					s.Close(NoFlush, RemoteClose)
				} else {
					s.Close(NoFlush, OnReadErrClose)
				}

				log.Println("Error on read:", s.id, err)
				return
			}
			runtime.Gosched()
		}
	}
	//buf := make([]byte, 0xFFFF)
	//
	//for {
	//	var close bool
	//	n, err := conn.Read(buf)
	//	if err != nil {
	//		log.Println("conn read error: ", err)
	//		return
	//	}
	//	in := buf[:n]
	//	s.doRead(in)
	//
	//	if close {
	//		break
	//	}
	//}
}

func (s *Session) startWriteLoop() {
	var err error
	for {
		// exit loop asap. one receive & one default block will be optimized by go compiler
		select {
		case <-s.internalStopChan:
			return
		default:
		}

		select {
		case <-s.internalStopChan:
			return

		case buf, ok := <-s.writeBufferChan:
			// check if chan closed
			if ok == false {
				return
			}
			s.appendBuffer(buf)

			//todo: dynamic set loop nums
			for i := 0; i < 10; i++ {
				select {
				case buf, ok := <-s.writeBufferChan:
					// check if chan closed
					if ok == false {
						s.resetBuffer()
						return
					}
					s.appendBuffer(buf)
				default:
				}
			}
			_, err = s.doWrite()
		}

		if err != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() {
				continue
			}

			if err == buffer.EOF {
				s.Close(NoFlush, LocalClose)
			} else if err == io.EOF {
				// remote conn closed
				s.Close(NoFlush, RemoteClose)
			} else {
				// on non-timeout error
				s.Close(NoFlush, OnWriteErrClose)
			}
			fmt.Println("Error on write:", s.id, err)

			return
		}
		runtime.Gosched()
	}
}

func (s *Session) appendBuffer(iobuffers *[]buffer.IoBuffer) {
	if iobuffers == nil {
		return
	}
	for _, buf := range *iobuffers {
		if buf == nil {
			continue
		}
		s.ioBuffers = append(s.ioBuffers, buf)
		s.writeBuffers = append(s.writeBuffers, buf.Bytes())
	}
}

func (s *Session) resetBuffer() {
	s.ioBuffers = s.ioBuffers[:0]
	s.writeBuffers = s.writeBuffers[:0]
}

func (s *Session) doWrite() (int64, error) {
	bytesSent, err := s.doWriteIo()

	//for _, cb := range s.bytesSendCallbacks {
	//	cb(uint64(bytesSent))
	//}

	return bytesSent, err
}

func (s *Session) doWriteIo() (bytesSent int64, err error) {
	buffers := s.writeBuffers
	if conn, ok := s.rawc.(net.Conn); ok {
		bytesSent, err = buffers.WriteTo(conn)
	}
	if err != nil {
		return bytesSent, err
	}
	for _, buf := range s.ioBuffers {
		buffer.PutIoBuffer(buf)
	}
	s.resetBuffer()
	if len(buffers) != 0 {
		for _, buf := range buffers {
			s.writeBuffers = append(s.writeBuffers, buf)
		}
	}
	return
}
