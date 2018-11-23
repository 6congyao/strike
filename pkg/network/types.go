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
	"net"
	"strike/pkg/api/v2"
)

// Listener is a wrapper of tcp listener
type Listener interface {
	// Return config which initialize this listener
	Config() *v2.Listener

	// Set listener config
	SetConfig(config *v2.Listener)

	// Name returns the listener's name
	Name() string

	// Addr returns the listener's network address.
	Addr() net.Addr

	// Start starts listener with context
	Start(lctx context.Context)

	// Stop stops listener
	// Accepted connections and listening sockets will not be closed
	Stop() error

	// ListenerTag returns the listener's tag, whichi the listener should use for connection handler tracking.
	ListenerTag() uint64

	// Set listener tag
	SetListenerTag(tag uint64)

	// ListenerFD returns a copy a listener fd
	ListenerFD() (uintptr, error)

	// SetListenerCallbacks set a listener event listener
	SetListenerCallbacks(cb ListenerEventListener)

	// GetListenerCallbacks set a listener event listener
	GetListenerCallbacks() ListenerEventListener

	// Close closes listener, not closing connections
	Close(lctx context.Context) error
}

// ConnectionEventListener is a network level callbacks that happen on a connection.
type ConnectionEventListener interface {
	// OnEvent is called on ConnectionEvent
	OnEvent(event ConnectionEvent)
}

type ConnectionHandler interface {
	// NumConnections reports the connections that ConnectionHandler keeps.
	NumConnections() uint64

	// AddOrUpdateListener
	// adds a listener into the ConnectionHandler or
	// update a listener
	AddOrUpdateListener(lc *v2.Listener, networkFiltersFactories []NetworkFilterChainFactory) (ListenerEventListener, error)

	// StartListener starts a listener by the specified listener tag
	StartListener(lctx context.Context, listenerTag uint64)

	//StartListeners starts all listeners the ConnectionHandler has
	StartListeners(lctx context.Context)

	// FindListenerByAddress finds and returns a listener by the specified network address
	FindListenerByAddress(addr net.Addr) Listener

	// FindListenerByName finds and returns a listener by the listener name
	FindListenerByName(name string) Listener

	// RemoveListeners find and removes a listener by listener name.
	RemoveListeners(name string)

	// StopListener stops a listener  by listener name
	StopListener(lctx context.Context, name string, stop bool) error

	// StopListeners stops all listeners the ConnectionHandler has.
	// The close indicates whether the listening sockets will be closed.
	StopListeners(lctx context.Context, close bool) error

	// ListListenersFD reports all listeners' fd
	ListListenersFD(lctx context.Context) []uintptr

	// StopConnection Stop Connection
	StopConnection()
}

// NetworkFilterChainFactory adds filter into NetWorkFilterChainFactoryCallbacks
//type NetworkFilterChainFactory interface {
//	CreateFilterChain(context context.Context, clusterManager ClusterManager, callbacks NetWorkFilterChainFactoryCallbacks)
//}

// ListenerEventListener is a Callback invoked by a listener.
type ListenerEventListener interface {
	// OnAccept is called on listener accepted new connection
	OnAccept(rawc interface{})

	// OnNewConnection is called on new connection created
	OnNewConnection(ctx context.Context, conn Connection)

	// OnClose is called on listener closed
	OnClose()
}

// Connection status
type ConnState string

// Connection statuses
const (
	Open    ConnState = "Open"
	Closing ConnState = "Closing"
	Closed  ConnState = "Closed"
)

// ConnectionCloseType represent connection close type
type ConnectionCloseType string

// ConnectionEvent type
type ConnectionEvent string

// ConnectionEvent types
const (
	RemoteClose     ConnectionEvent = "RemoteClose"
	LocalClose      ConnectionEvent = "LocalClose"
	OnReadErrClose  ConnectionEvent = "OnReadErrClose"
	OnWriteErrClose ConnectionEvent = "OnWriteErrClose"
	OnConnect       ConnectionEvent = "OnConnect"
	Connected       ConnectionEvent = "ConnectedFlag"
	ConnectTimeout  ConnectionEvent = "ConnectTimeout"
	ConnectFailed   ConnectionEvent = "ConnectFailed"
)

// Connection interface
type Connection interface {
	// ID returns unique connection id
	ID() uint64

	// Start starts connection with context.
	// See context.go to get available keys in context
	Start(ctx context.Context)

	// Write writes data to the connection.
	// Called by other-side stream connection's read loop. Will loop through stream filters with the buffer if any are installed.
	Write(b []byte) (n int, err error)

	// Close closes connection with connection type and event type.
	// ConnectionCloseType - how to close to connection
	// 	- FlushWrite: connection will be closed after buffer flushed to underlying io
	//	- NoFlush: close connection asap
	// ConnectionEvent - why to close the connection
	// 	- RemoteClose
	//  - LocalClose
	// 	- OnReadErrClose
	//  - OnWriteErrClose
	//  - OnConnect
	//  - Connected:
	//	- ConnectTimeout
	//	- ConnectFailed
	Close(ccType ConnectionCloseType, eventType ConnectionEvent) error

	// RemoteAddr returns the remote address of the connection.
	RemoteAddr() net.Addr

	// SetRemoteAddr is used for originaldst we need to replace remoteAddr
	SetRemoteAddr(address net.Addr)

	// AddConnectionEventListener add a listener method will be called when connection event occur.
	AddConnectionEventListener(cb ConnectionEventListener)

	// GetReadBuffer is used by network read filter
	GetReadBuffer() []byte

	// FilterManager returns the FilterManager
	FilterManager() FilterManager

	// RawConn returns the original connections.
	RawConn() interface{}
}

// ConnectionStats is a group of connection metrics
type ConnectionStats struct {
	//ReadTotal     metrics.Counter
	//ReadBuffered  metrics.Gauge
	//WriteTotal    metrics.Counter
	//WriteBuffered metrics.Gauge
}

// ReadFilterCallbacks is called by read filter to talk to connection
type ReadFilterCallbacks interface {
	// Connection returns the connection triggered the callback
	Connection() Connection

	// ContinueReading filter iteration on filter stopped, next filter will be called with current read buffer
	ContinueReading()
}

// FilterManager is a groups of filters
type FilterManager interface {
	// AddReadFilter adds a read filter
	AddReadFilter(rf ReadFilter)

	// AddWriteFilter adds a write filter
	AddWriteFilter(wf WriteFilter)

	// ListReadFilter returns the list of read filters
	ListReadFilter() []ReadFilter

	// ListWriteFilters returns the list of write filters
	ListWriteFilters() []WriteFilter

	// InitializeReadFilters initialize read filters
	InitializeReadFilters() bool

	// OnRead is called on data read
	OnRead()

	// OnWrite is called before data write
	OnWrite(buffer []byte) FilterStatus
}

// ReadFilter is a connection binary read filter, registered by FilterManager.AddReadFilter
type ReadFilter interface {
	// OnData is called everytime bytes is read from the connection
	OnData(buffer []byte) FilterStatus

	// OnNewConnection is called on new connection is created
	OnNewConnection() FilterStatus

	// InitializeReadFilterCallbacks initials read filter callbacks. It used by init read filter
	InitializeReadFilterCallbacks(cb ReadFilterCallbacks)
}

// WriteFilter is a connection binary write filter, only called by conn accept loop
type WriteFilter interface {
	// OnWrite is called before data write to raw connection
	OnWrite(buffer []byte) FilterStatus
}

type TLSContextManager interface {
	Conn(net.Conn) net.Conn
	Enabled() bool
}

// FilterStatus type
type FilterStatus string

// FilterStatus types
const (
	Continue FilterStatus = "Continue"
	Stop     FilterStatus = "Stop"
)

type ListenerFilter interface {
	// OnAccept is called when a raw connection is accepted, but before a Connection is created.
	OnAccept(cb ListenerFilterCallbacks) FilterStatus
}

// ListenerFilterCallbacks is a callback handler called by listener filter to talk to listener
type ListenerFilterCallbacks interface {
	// Conn returns the Connection reference used in callback handler
	Conn() interface{}

	ContinueFilterChain(ctx context.Context, success bool)

	// SetOriginalAddr sets the original ip and port
	SetOriginalAddr(ip string, port int)
}

// NetworkFilterChainFactory adds filter into NetWorkFilterChainFactoryCallbacks
type NetworkFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks NetWorkFilterChainFactoryCallbacks)
}

// NetWorkFilterChainFactoryCallbacks is a wrapper of FilterManager that called in NetworkFilterChainFactory
type NetWorkFilterChainFactoryCallbacks interface {
	AddReadFilter(rf ReadFilter)
	AddWriteFilter(wf WriteFilter)
}