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
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"strike/pkg/api/v2"
	"strike/pkg/evio"
	"sync"
	"time"
)

var UseEdgeMode = false

// listener impl based on golang net package
type listener struct {
	name                                  string
	localAddress                          net.Addr
	bindToPort                            bool
	listenerTag                           uint64
	perConnBufferLimitBytes               uint32
	handOffRestoredDestinationConnections bool
	cb                                    ListenerEventListener
	rawl                                  *net.TCPListener
	config                                *v2.Listener
}

func NewListener(lc *v2.Listener) Listener {
	if UseEdgeMode {
		return &edgeListener{
			name:         lc.Name,
			localAddress: lc.Addr,
			bindToPort:   lc.BindToPort,
			listenerTag:  lc.ListenerTag,
			config:       lc,
		}
	} else {
		l := &listener{
			name:                                  lc.Name,
			localAddress:                          lc.Addr,
			bindToPort:                            lc.BindToPort,
			listenerTag:                           lc.ListenerTag,
			perConnBufferLimitBytes:               lc.PerConnBufferLimitBytes,
			handOffRestoredDestinationConnections: lc.HandOffRestoredDestinationConnections,
			config:                                lc,
		}

		if lc.InheritListener != nil {
			//inherit old process's listener
			l.rawl = lc.InheritListener
		}
		return l
	}
}

func (l *listener) Config() *v2.Listener {
	return l.config
}

func (l *listener) SetConfig(config *v2.Listener) {
	l.config = config
}

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Addr() net.Addr {
	return l.localAddress
}

func (l *listener) Start(lctx context.Context) {
	if l.bindToPort {
		if l.rawl == nil {
			if err := l.listen(lctx); err != nil {
				log.Fatalln(l.name, "listen failed:", err)
				return
			}
		}
	}

	for {
		if err := l.accept(lctx); err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				log.Println("listener stop accepting connections by deadline: ", l.name)
				return
			} else if ope, ok := err.(*net.OpError); ok {
				if !(ope.Timeout() && ope.Temporary()) {
					// non-recoverable
					if ope.Op == "accept" {
						log.Println("listener closed: ", l.name, l.Addr())
					} else {
						log.Println("listener occurs non-recoverable error, stop listening and accepting: ", l.name, err.Error())
					}
					return
				}
			} else {
				log.Println("listener occurs unknown error while accepting: ", l.name, err.Error())
			}
		}
	}

}

func (l *listener) Stop() error {
	return l.rawl.SetDeadline(time.Now())
}

func (l *listener) ListenerTag() uint64 {
	return l.listenerTag
}

func (l *listener) SetListenerTag(tag uint64) {
	l.listenerTag = tag
}

func (l *listener) ListenerFD() (uintptr, error) {
	file, err := l.rawl.File()
	if err != nil {
		log.Fatalln("listener fd not found:", l.name, err)
		return 0, err
	}
	//defer file.Close()
	return file.Fd(), nil
}

func (l *listener) PerConnBufferLimitBytes() uint32 {
	return l.perConnBufferLimitBytes
}

func (l *listener) SetPerConnBufferLimitBytes(limitBytes uint32) {
	l.perConnBufferLimitBytes = limitBytes
}

func (l *listener) SetListenerCallbacks(cb ListenerEventListener) {
	l.cb = cb
}

func (l *listener) GetListenerCallbacks() ListenerEventListener {
	return l.cb
}

func (l *listener) SetHandOffRestoredDestinationConnections(restoredDestation bool) {
	l.handOffRestoredDestinationConnections = restoredDestation
}

func (l *listener) HandOffRestoredDestinationConnections() bool {
	return l.handOffRestoredDestinationConnections
}

func (l *listener) Close(lctx context.Context) error {
	l.cb.OnClose()
	return l.rawl.Close()
}

func (l *listener) listen(lctx context.Context) error {
	var err error

	var rawl *net.TCPListener
	if rawl, err = net.ListenTCP("tcp", l.localAddress.(*net.TCPAddr)); err != nil {
		return err
	}

	l.rawl = rawl

	return nil
}

func (l *listener) accept(lctx context.Context) error {
	rawc, err := l.rawl.Accept()

	if err != nil {
		return err
	}

	// async
	// TODO: use thread pool
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Println("panic: ", p)

				debug.PrintStack()
			}
		}()

		l.cb.OnAccept(rawc, l.handOffRestoredDestinationConnections, nil, nil, nil)
	}()

	return nil
}

type edgeListener struct {
	name         string
	localAddress net.Addr
	bindToPort   bool
	listenerTag  uint64

	cb     ListenerEventListener
	config *v2.Listener

	connMap    sync.Map
	readerPool sync.Pool
}

func (el *edgeListener) serve(lctx context.Context) error {
	var events evio.Events
	events.NumLoops = -1
	events.LoadBalance = evio.LeastConnections

	events.Serving = func(eserver evio.Server) (action evio.Action) {
		if eserver.NumLoops == 1 {
			fmt.Println("Running single-threaded")
		} else {
			fmt.Println("Running on threads: ", eserver.NumLoops)
		}
		for _, addr := range eserver.Addrs {
			fmt.Println("Ready to accept connections at", addr)
		}
		return
	}

	events.Opened = func(econn evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
		// create the client
		session := NewSession(econn.RemoteAddr())

		// keep track of the client
		econn.SetContext(session)

		// add client to server map
		fmt.Println("Session opened:", session.RemoteAddr())
		el.connMap.Store(session.ID(), session)
		return
	}

	events.Closed = func(econn evio.Conn, err error) (action evio.Action) {
		session := econn.Context().(*Session)

		el.connMap.Delete(session.ID())

		log.Println("Closed connection: ", session.remoteAddr.String())
		return
	}

	events.Data = func(econn evio.Conn, in []byte) (out []byte, action evio.Action) {
		session := econn.Context().(*Session)
		p := session.In.Begin(in)
		session.pr = el.acquireReader(session)
		pr := session.PipelineReader()
		rbuf := bytes.NewBuffer(p)
		pr.Rd = rbuf
		pr.Wr = session

		msgs, err := pr.ReadMessages()
		if err != nil {
			action = evio.Close
			return
		} else {
			fmt.Println("give resp, conn id is:", session.ID())
			out = AppendResp(nil, "200 OK", "", string(p) + "\r\n")
			return
		}

		for _, msg := range msgs {
			if msg != nil && msg.Command() != "" {
				fmt.Println("got msg:", msg)
			}
		}
		p = p[len(p)-rbuf.Len():]
		session.In.End(p)

		out = session.Out
		session.Out = nil
		return
	}

	return evio.Serve(events, el.localAddress.String())
}

func AppendResp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: nbserver\r\n"...)
	b = append(b, "Connection: keep-alive\r\n"...)
	//b = append(b, "Content-Type: application/json;charset=UTF-8\r\n"...)

	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	if len(head) > 0 {
		b = append(b, head...)
	}
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

func (el *edgeListener) acquireReader(session *Session) *PipelineReader {
	if session.pr != nil {
		return session.pr
	}

	v := el.readerPool.Get()
	var pr *PipelineReader
	if v == nil {
		pr = &PipelineReader{}
	} else {
		pr = v.(*PipelineReader)
	}

	return pr
}

func (el *edgeListener) Config() *v2.Listener {
	return el.config
}

func (el *edgeListener) SetConfig(config *v2.Listener) {
	el.config = config
}

func (el *edgeListener) Name() string {
	return el.name
}

func (el *edgeListener) Addr() net.Addr {
	return el.localAddress
}

func (el *edgeListener) Start(lctx context.Context) {
	if !el.bindToPort {
		return
	}

	if err := el.serve(lctx); err != nil {
		log.Fatalln(el.name, "listen failed:", err)
		return
	}
}

func (el *edgeListener) Stop() error {
	panic("implement me")
}

func (el *edgeListener) ListenerTag() uint64 {
	return el.listenerTag
}

func (el *edgeListener) SetListenerTag(tag uint64) {
	el.listenerTag = tag
}

func (el *edgeListener) ListenerFD() (uintptr, error) {
	panic("implement me")
}

func (el *edgeListener) PerConnBufferLimitBytes() uint32 {
	panic("implement me")
}

func (el *edgeListener) SetPerConnBufferLimitBytes(limitBytes uint32) {
	panic("implement me")
}

func (el *edgeListener) SetHandOffRestoredDestinationConnections(restoredDestation bool) {
	panic("implement me")
}

func (el *edgeListener) HandOffRestoredDestinationConnections() bool {
	panic("implement me")
}

func (el *edgeListener) SetListenerCallbacks(cb ListenerEventListener) {
	el.cb = cb
}

func (el *edgeListener) GetListenerCallbacks() ListenerEventListener {
	return el.cb
}

func (el *edgeListener) Close(lctx context.Context) error {
	panic("implement me")
}
