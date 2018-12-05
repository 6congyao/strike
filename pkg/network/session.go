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
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strike/pkg/buffer"
	"strike/pkg/evio"
	"strings"
	"sync"
	"sync/atomic"
)

var globalSessionId uint64 = 0
var PipelineReaderPool sync.Pool

const DefaultBufferReadCapacity = 1 << 0

type Session struct {
	id         uint64
	remoteAddr net.Addr
	closeFlag  int32

	In            evio.InputStream
	Out           []byte
	Pr            *PipelineReader
	Mu            sync.Mutex
	filterManager FilterManager
	rawc          interface{}
	readBuffer    buffer.IoBuffer
}

func NewSession(rawc interface{}, radd net.Addr) *Session {
	s := &Session{
		id:         atomic.AddUint64(&globalSessionId, 1),
		remoteAddr: radd,
		rawc:       rawc,
	}

	s.filterManager = newFilterManager(s)
	return s
}

// start read loop on std net conn
// do nothing on edge conn
func (s *Session) Start(ctx context.Context) {
	if !s.IsClosed() {
		if conn, ok := s.RawConn().(net.Conn); ok {
			s.startReadLoop(ctx, conn)
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

	if conn, ok := s.RawConn().(net.Conn); ok {
		conn.Close()
	}

	return nil
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

// todo:
func (s *Session) Write(b []byte) (n int, err error) {
	s.Out = append(s.Out, b...)
	return len(b), nil
}

func (s *Session) SetRemoteAddr(addr net.Addr) {
	s.remoteAddr = addr
}

func (s *Session) AddConnectionEventListener(cb ConnectionEventListener) {

}

func (s *Session) GetReadBuffer() buffer.IoBuffer {
	return s.readBuffer
}

func (s *Session) FilterManager() FilterManager {
	return s.filterManager
}

func (s *Session) RawConn() interface{} {
	return s.rawc
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

func (s *Session) startReadLoop(ctx context.Context, conn net.Conn) {
	buf := make([]byte, 0xFFFF)

	for {
		var close bool
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("conn read error: ", err)
			return
		}
		in := buf[:n]
		s.doRead(in)

		if close {
			break
		}
	}
}

// PipelineReader ...
type PipelineReader struct {
	Rd     io.Reader
	Wr     io.Writer
	Packet [0xFFFF]byte
	Buf    []byte
}

// Type is resp type
type Type byte

// Protocol Types
const (
	Null Type = iota
	RESP
	Telnet
	Native
	HTTP
	WebSocket
	JSON
)

type Message struct {
	_command   string
	Args       []string
	ConnType   Type
	OutputType Type
	Auth       string
}

// Command returns the first argument as a lowercase string
func (msg *Message) Command() string {
	if msg._command == "" {
		msg._command = strings.ToLower(msg.Args[0])
	}
	return msg._command
}

// ReadMessages ...
func (rd *PipelineReader) ReadMessages() ([]*Message, error) {
	var msgs []*Message
moreData:
	n, err := rd.Rd.Read(rd.Packet[:])
	if err != nil {
		return nil, err
	}
	if n == 0 {
		// need more data
		goto moreData
	}
	data := rd.Packet[:n]
	if len(rd.Buf) > 0 {
		data = append(rd.Buf, data...)
	}
	for len(data) > 0 {
		msg := &Message{}
		complete, args, kind, leftover, err := readNextCommand(data, nil, msg, rd.Wr)
		if err != nil {
			break
		}
		if !complete {
			break
		}
		if kind == KindHttp {
			if len(msg.Args) == 0 {
				return nil, errors.New("invalid HTTP request")
			}
			msgs = append(msgs, msg)
		} else if len(args) > 0 {
			for i := 0; i < len(args); i++ {
				msg.Args = append(msg.Args, string(args[i]))
			}
			switch kind {
			case KindTelnet:
				msg.ConnType = RESP
				msg.OutputType = RESP
			}
			msgs = append(msgs, msg)
		}
		data = leftover
	}
	if len(data) > 0 {
		rd.Buf = append(rd.Buf[:0], data...)
	} else if len(rd.Buf) > 0 {
		rd.Buf = rd.Buf[:0]
	}
	if err != nil && len(msgs) == 0 {
		return nil, err
	}
	return msgs, nil
}

// Kind is the kind of command
type Kind int

const (
	KindHttp Kind = iota
	KindTelnet
)

func readNextCommand(packet []byte, argsIn [][]byte, msg *Message, wr io.Writer) (
	complete bool, args [][]byte, kind Kind, leftover []byte, err error,
) {
	if packet[0] == 'G' || packet[0] == 'P' {
		// could be an HTTP request
		var line []byte
		for i := 1; i < len(packet); i++ {
			if packet[i] == '\n' {
				if packet[i-1] == '\r' {
					line = packet[:i+1]
					break
				}
			}
		}

		if len(line) > 11 && string(line[len(line)-11:len(line)-5]) == " HTTP/" {
			fmt.Println("http packet")
		}
	}
	return false, argsIn[:0], KindHttp, packet, nil
}

func AcquirePipelineReader(s *Session) *PipelineReader {
	if s.Pr != nil {
		return s.Pr
	}

	v := PipelineReaderPool.Get()
	var pr *PipelineReader
	if v == nil {
		pr = &PipelineReader{}
	} else {
		pr = v.(*PipelineReader)
	}

	return pr
}

func ReleasePipelineReader(s *Session) {
	if s == nil || s.Pr == nil {
		return
	}

	s.Pr.Buf = nil
	s.Pr.Rd = nil
	s.Pr.Wr = nil
	PipelineReaderPool.Put(s.Pr)
	s.Pr = nil
}
