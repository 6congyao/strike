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
	"log"
	"net"
	"strike/pkg/api/v2"
	"time"
)

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

	l := &listener{
		name:                                  lc.Name,
		localAddress:                          lc.Addr,
		bindToPort:                            lc.BindToPort,
		listenerTag:                           lc.ListenerTag,
		perConnBufferLimitBytes:               lc.PerConnBufferLimitBytes,
		handOffRestoredDestinationConnections: lc.HandOffRestoredDestinationConnections,
		config: lc,
	}

	if lc.InheritListener != nil {
		//inherit old process's listener
		l.rawl = lc.InheritListener
	}
	return l
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
	return
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
		log.Fatalf(" listener %s fd not found : %v \n", l.name, err)
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