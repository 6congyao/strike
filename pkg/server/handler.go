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
	"os"
	"reflect"
	"strconv"
	"strike/pkg/api/v2"
	"strike/pkg/evio"
	"strike/pkg/network"
	"strike/pkg/stls"
	"strike/pkg/stream"
	"strike/pkg/types"
	"strike/pkg/upstream"
	"strings"
	"sync"
	"sync/atomic"
)

type connHandler struct {
	numConnections int64
	listeners      []*activeListener
	cm             upstream.ClusterManager
}

// NewHandler
// create network.ConnectionHandler's implement connHandler
func NewHandler(cm upstream.ClusterManager) ConnectionHandler {
	ch := &connHandler{
		numConnections: 0,
		listeners:      make([]*activeListener, 0),
		cm:             cm,
	}

	return ch
}

func (ch *connHandler) GenerateListenerID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		log.Fatalln("generate an uuid failed, error:", err)
	}
	// see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// ConnectionHandler
func (ch *connHandler) AddOrUpdateListener(lc *v2.Listener, networkFiltersFactories []network.NetworkFilterChainFactory,
	streamFiltersFactories []stream.StreamFilterChainFactory) (network.ListenerEventListener, error) {
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
		equalNetworkFilter := reflect.DeepEqual(al.networkFiltersFactories, networkFiltersFactories)
		equalStreamFilters := reflect.DeepEqual(al.streamFiltersFactories, streamFiltersFactories)
		// duplicate config does nothing
		if equalConfig && equalNetworkFilter {
			log.Println("duplicate listener found. no add/update: ", listenerName)
			return nil, nil
		}

		// update some config, and as Address and Name doesn't change , so need't change *rawl
		al.updatedLabel = true
		if !equalConfig {
			al.disableConnIo = lc.DisableConnIo
			al.listener.SetConfig(lc)
			al.listener.SetListenerTag(lc.ListenerTag)
			log.Println("AddOrUpdateListener: use new listen config: ", lc)
		}

		// update network filter
		if !equalNetworkFilter {
			al.networkFiltersFactories = networkFiltersFactories
			log.Println("AddOrUpdateListener: use new networkFiltersFactories: ", networkFiltersFactories)
		}

		// update stream filter
		if !equalStreamFilters {
			al.streamFiltersFactories = streamFiltersFactories
			log.Println("AddOrUpdateListener: use new streamFiltersFactories:", streamFiltersFactories)
		}
	} else {
		// listener doesn't exist, add the listener
		listenerStopChan := make(chan struct{})

		l := network.NewListener(lc)

		al, err := newActiveListener(l, lc, networkFiltersFactories, streamFiltersFactories, ch, listenerStopChan)
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

// async
func (ch *connHandler) StartListener(lctx context.Context, listenerTag uint64) {
	for _, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			go l.listener.Start(lctx)
		}
	}
}

// async
func (ch *connHandler) StartListeners(lctx context.Context) {
	for _, l := range ch.listeners {
		go l.listener.Start(lctx)
	}
}

func (ch *connHandler) FindListenerByAddress(addr net.Addr) network.Listener {
	l := ch.findActiveListenerByAddress(addr)

	if l == nil {
		return nil
	}

	return l.listener
}

func (ch *connHandler) FindListenerByName(name string) network.Listener {
	l := ch.findActiveListenerByName(name)

	if l == nil {
		return nil
	}

	return l.listener
}

func (ch *connHandler) RemoveListeners(name string) {
	for i, l := range ch.listeners {
		if l.listener.Name() == name {
			ch.listeners = append(ch.listeners[:i], ch.listeners[i+1:]...)
		}
	}
}

func (ch *connHandler) StopListener(lctx context.Context, name string, close bool) error {
	for _, l := range ch.listeners {
		if l.listener.Name() == name {
			// stop goroutine
			if close {
				return l.listener.Close(lctx)
			}

			return l.listener.Stop()
		}
	}

	return nil
}

func (ch *connHandler) StopListeners(lctx context.Context, close bool) error {
	var errGlobal error
	for _, l := range ch.listeners {
		// stop goroutine
		if close {
			if err := l.listener.Close(lctx); err != nil {
				errGlobal = err
			}
		} else {
			if err := l.listener.Stop(); err != nil {
				errGlobal = err
			}
		}
	}

	return errGlobal
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

func (ch *connHandler) findActiveListenerByAddress(addr net.Addr) *activeListener {
	for _, l := range ch.listeners {
		if l.listener != nil {
			if l.listener.Addr().Network() == addr.Network() &&
				l.listener.Addr().String() == addr.String() {
				return l
			}
		}
	}

	return nil
}

// ClusterConfigFactoryCb
func (ch *connHandler) UpdateClusterConfig(clusters []v2.Cluster) error {

	for _, cluster := range clusters {
		if !ch.cm.AddOrUpdatePrimaryCluster(cluster) {
			return fmt.Errorf("UpdateClusterConfig: AddOrUpdatePrimaryCluster failure, cluster name = %s", cluster.Name)
		}
	}

	// TODO: remove cluster

	return nil
}

// ClusterHostFactoryCb
func (ch *connHandler) UpdateClusterHost(cluster string, priority uint32, hosts []v2.Host) error {
	return ch.cm.UpdateClusterHosts(cluster, priority, hosts)
}

// ListenerEventListener
type activeListener struct {
	disableConnIo           bool
	listener                network.Listener
	networkFiltersFactories []network.NetworkFilterChainFactory
	streamFiltersFactories  []stream.StreamFilterChainFactory
	listenIP                string
	listenPort              int
	sMap                    sync.Map
	//connsMux                sync.RWMutex
	handler                 *connHandler
	stopChan                chan struct{}
	updatedLabel            bool
	tlsMgr                  network.TLSContextManager
}

func newActiveListener(listener network.Listener, lc *v2.Listener, networkFiltersFactories []network.NetworkFilterChainFactory,
	streamFiltersFactories []stream.StreamFilterChainFactory, handler *connHandler, stopChan chan struct{}) (*activeListener, error) {
	al := &activeListener{
		disableConnIo:           lc.DisableConnIo,
		listener:                listener,
		networkFiltersFactories: networkFiltersFactories,
		streamFiltersFactories:  streamFiltersFactories,
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

	mgr, err := stls.NewTLSServerContextManager(lc, listener)
	if err != nil {
		log.Println("create tls context manager failed")
		return nil, err
	}
	al.tlsMgr = mgr

	return al, nil
}

func (al *activeListener) OnAccept(rawc interface{}) {
	if al.tlsMgr != nil && al.tlsMgr.Enabled() {
		if c, ok := rawc.(net.Conn); ok {
			rawc = al.tlsMgr.Conn(c)
		}
	}

	arc := newActiveRawConn(rawc, al)

	ctx := context.WithValue(context.Background(), types.ContextKeyListenerPort, al.listenPort)
	ctx = context.WithValue(ctx, types.ContextKeyListenerName, al.listener.Name())
	ctx = context.WithValue(ctx, types.ContextKeyNetworkFilterChainFactories, al.networkFiltersFactories)
	ctx = context.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, al.streamFiltersFactories)

	arc.ContinueFilterChain(ctx, true)
}

func (al *activeListener) OnNewConnection(ctx context.Context, conn network.Connection) {
	filterManager := conn.FilterManager()
	for _, nfcf := range al.networkFiltersFactories {
		nfcf.CreateFilterChain(ctx, al.handler.cm, filterManager)
	}

	filterManager.InitializeReadFilters()

	if len(filterManager.ListReadFilter()) == 0 &&
		len(filterManager.ListWriteFilters()) == 0 {
		// no filter found, close connection
		conn.Close(network.NoFlush, network.LocalClose)
		return
	}

	al.sMap.Store(conn.ID(), conn)
	atomic.AddInt64(&al.handler.numConnections, 1)

	conn.Start(ctx)
}

func (al *activeListener) OnClose() {

}

func (al *activeListener) newConnection(ctx context.Context, rawc interface{}) {
	var session *network.Session
	if conn, ok := rawc.(net.Conn); ok {
		session = network.NewSession(conn, conn.RemoteAddr())

		//sc := network.NewServerSimpleConn(ctx, conn, al.stopChan)
		//newCtx := context.WithValue(ctx, types.ContextKeyConnectionID, sc.ID())
		//
	} else if conn, ok := rawc.(evio.Conn); ok {
		// create the session
		session = network.NewSession(conn, conn.RemoteAddr())

		// keep track of the session
		conn.SetContext(session)
	}
	newCtx := context.WithValue(ctx, types.ContextKeyConnectionID, session.ID())
	newCtx = context.WithValue(newCtx, types.ContextKeyConnectionRef, session)

	// notify
	al.OnNewConnection(newCtx, session)
}

type activeRawConn struct {
	rawc                interface{}
	rawf                *os.File
	originalDstIP       string
	originalDstPort     int
	oriRemoteAddr       net.Addr
	rawcElement         *list.Element
	activeListener      *activeListener
	acceptedFilters     []network.ListenerFilter
	acceptedFilterIndex int
}

func newActiveRawConn(rawc interface{}, activeListener *activeListener) *activeRawConn {
	return &activeRawConn{
		rawc:           rawc,
		activeListener: activeListener,
	}
}

func (arc *activeRawConn) ContinueFilterChain(ctx context.Context, success bool) {
	if !success {
		return
	}

	for ; arc.acceptedFilterIndex < len(arc.acceptedFilters); arc.acceptedFilterIndex++ {
		filterStatus := arc.acceptedFilters[arc.acceptedFilterIndex].OnAccept(arc)
		if filterStatus == network.Stop {
			return
		}
	}

	arc.activeListener.newConnection(ctx, arc.rawc)
}

func (arc *activeRawConn) Conn() interface{} {
	return arc.rawc
}

func (arc *activeRawConn) SetOriginalAddr(ip string, port int) {
	arc.originalDstIP = ip
	arc.originalDstPort = port
	arc.oriRemoteAddr, _ = net.ResolveTCPAddr("", ip+":"+strconv.Itoa(port))
	log.Println("conn set origin addr: ", ip, port)
}
