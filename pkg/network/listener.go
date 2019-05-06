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
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"strike/pkg/api/v2"
	"strike/pkg/evio"
	"time"
)

var UseEdgeMode = false

// listener impl based on golang net package
type listener struct {
	name         string
	localAddress net.Addr
	bindToPort   bool
	listenerTag  uint64
	cb           ListenerEventListener
	rawl         *net.TCPListener
	config       *v2.Listener
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
			name:         lc.Name,
			localAddress: lc.Addr,
			bindToPort:   lc.BindToPort,
			listenerTag:  lc.ListenerTag,
			config:       lc,
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

// ListenerFile returns a copy a listener file
func (l *listener) ListenerFile() (*os.File, error) {
	return l.rawl.File()
}

func (l *listener) SetListenerCallbacks(cb ListenerEventListener) {
	l.cb = cb
}

func (l *listener) GetListenerCallbacks() ListenerEventListener {
	return l.cb
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

	log.Println("listener started on: ", l.name, l.Addr())
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

		fmt.Println("Connection opened:", rawc.RemoteAddr())
		l.cb.OnAccept(rawc)
	}()

	return nil
}

// edge listener impl based on evio package
type edgeListener struct {
	name         string
	localAddress net.Addr
	bindToPort   bool
	listenerTag  uint64

	cb     ListenerEventListener
	config *v2.Listener
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
		fmt.Println("Connection opened:", econn.RemoteAddr())
		opts.ReuseInputBuffer = true
		// notify
		el.cb.OnAccept(econn)

		return
	}

	events.Closed = func(econn evio.Conn, err error) (action evio.Action) {
		el.cb.OnClose()
		session := econn.Context().(*Session)
		ReleasePipelineReader(session)

		fmt.Println("Connection closed:", session.remoteAddr.String())
		return
	}

	events.Data = func(econn evio.Conn, in []byte) (out []byte, action evio.Action) {
		session := econn.Context().(*Session)

		// todo: workerpool
		session.doRead(in)

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

//func (el *edgeListener) acquireReader(session *Session) *PipelineReader {
//	if session.pr != nil {
//		return session.pr
//	}
//
//	v := el.readerPool.Get()
//	var pr *PipelineReader
//	if v == nil {
//		pr = &PipelineReader{}
//	} else {
//		pr = v.(*PipelineReader)
//	}
//
//	return pr
//}

//func (el *edgeListener) releaseReader(session *Session) {
//	if session == nil || session.pr == nil {
//		return
//	}
//
//	session.pr.Buf = nil
//	session.pr.Rd = nil
//	session.pr.Wr = nil
//	el.readerPool.Put(session.pr)
//	session.pr = nil
//}

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

func (el *edgeListener) ListenerFile() (*os.File, error) {
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
