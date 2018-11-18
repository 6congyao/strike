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
	"errors"
	"fmt"
	"io"
	"net"
	"strike/pkg/evio"
	"strings"
	"sync"
	"sync/atomic"
)

type Session struct {
	id         uint64
	remoteAddr net.Addr

	In  evio.InputStream
	Out []byte
	pr  *PipelineReader
	Mu  sync.Mutex
}

func NewSession(radd net.Addr) *Session {
	s := &Session{
		id:         atomic.AddUint64(&globalSessionId, 1),
		remoteAddr: radd,
		pr:         &PipelineReader{},
	}
	return s
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) RemoteAddr() net.Addr {
	return s.remoteAddr
}

func (s *Session) PipelineReader() *PipelineReader {
	return s.pr
}

func (s *Session) Write(b []byte) (n int, err error) {
	s.Out = append(s.Out, b...)
	return len(b), nil
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