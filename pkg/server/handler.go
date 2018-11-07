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

package server

import (
	"container/list"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"reflect"
	"strconv"
	"strike/pkg/api/v2"
	"strike/pkg/network"
	"strings"
	"sync"
	"sync/atomic"
)

type connHandler struct {
	numConnections int64
	listeners      []*activeListener
}

// NewHandler
// create network.ConnectionHandler's implement connHandler
func NewHandler() network.ConnectionHandler {
	ch := &connHandler{
		numConnections: 0,
	}

	return ch
}

func (ch *connHandler) GenerateListenerID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		log.Fatalf("generate an uuid failed, error: %v \n", err)
	}
	// see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// ConnectionHandler
// todo
func (ch *connHandler) AddOrUpdateListener(lc *v2.Listener) (network.ListenerEventListener, error) {

	var listenerName string
	if lc.Name == "" {
		listenerName = ch.GenerateListenerID()
		lc.Name = listenerName
	} else {
		listenerName = lc.Name
	}

	var al *activeListener
	if al = ch.findActiveListenerByName(listenerName); al != nil {
		// listener already exist, update the listener

		// a listener with the same name must have the same configured address
		if al.listener.Addr().String() != lc.Addr.String() ||
			al.listener.Addr().Network() != lc.Addr.Network() {
			return nil, fmt.Errorf("error updating listener, listen address and listen name doesn't match")
		}

		equalConfig := reflect.DeepEqual(al.listener.Config(), lc)
		// duplicate config does nothing
		if equalConfig {
			log.Printf("duplicate listener:%s found. no add/update \n", listenerName)
			return nil, nil
		}

		// update some config, and as Address and Name doesn't change , so need't change *rawl
		al.updatedLabel = true
		if !equalConfig {
			al.disableConnIo = lc.DisableConnIo
			al.listener.SetConfig(lc)
			al.listener.SetPerConnBufferLimitBytes(lc.PerConnBufferLimitBytes)
			al.listener.SetListenerTag(lc.ListenerTag)
			al.listener.SetHandOffRestoredDestinationConnections(lc.HandOffRestoredDestinationConnections)
			log.Printf("AddOrUpdateListener: use new listen config = %+v \n", lc)
		}
	} else {
		// listener doesn't exist, add the listener
		listenerStopChan := make(chan struct{})

		l := network.NewListener(lc)

		al, err := newActiveListener(l, lc, ch, listenerStopChan)
		if err != nil {
			return al, err
		}
		l.SetListenerCallbacks(al)
		ch.listeners = append(ch.listeners, al)
	}
	return al, nil
}

func (ch *connHandler) NumConnections() uint64 {
	return uint64(atomic.LoadInt64(&ch.numConnections))
}

func (ch *connHandler) StartListener(lctx context.Context, listenerTag uint64) {
	return
}

func (ch *connHandler) StartListeners(lctx context.Context) {
	return
}

func (ch *connHandler) FindListenerByAddress(addr net.Addr) network.Listener {
	return nil
}

func (ch *connHandler) FindListenerByName(name string) network.Listener {
	return nil
}

func (ch *connHandler) RemoveListeners(name string) {
	return
}

func (ch *connHandler) StopListener(lctx context.Context, name string, close bool) error {
	return nil
}

func (ch *connHandler) StopListeners(lctx context.Context, close bool) error {
	return nil
}

func (ch *connHandler) ListListenersFD(lctx context.Context) []uintptr {
	return nil
}

func (ch *connHandler) StopConnection() {
	return
}

func (ch *connHandler) findActiveListenerByName(name string) *activeListener {
	for _, l := range ch.listeners {
		if l.listener != nil {
			if l.listener.Name() == name {
				return l
			}
		}
	}

	return nil
}

// ListenerEventListener
type activeListener struct {
	disableConnIo bool
	listener      network.Listener
	listenIP      string
	listenPort    int
	conns         *list.List
	connsMux      sync.RWMutex
	handler       *connHandler
	stopChan      chan struct{}
	updatedLabel  bool
	tlsMng        network.TLSContextManager
}

func newActiveListener(listener network.Listener, lc *v2.Listener, handler *connHandler, stopChan chan struct{}) (*activeListener, error) {
	al := &activeListener{
		disableConnIo:           lc.DisableConnIo,
		listener:                listener,
		conns:                   list.New(),
		handler:                 handler,
		stopChan:                stopChan,
		updatedLabel:            false,
	}

	listenPort := 0
	var listenIP string
	localAddr := al.listener.Addr().String()

	if temps := strings.Split(localAddr, ":"); len(temps) > 0 {
		listenPort, _ = strconv.Atoi(temps[len(temps)-1])
		listenIP = temps[0]
	}

	al.listenIP = listenIP
	al.listenPort = listenPort

	return al, nil
}

func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool, oriRemoteAddr net.Addr, ch chan network.Connection, buf []byte) {

}

func (al *activeListener) OnNewConnection(ctx context.Context, conn network.Connection) {

}

func (al *activeListener) OnClose() {

}