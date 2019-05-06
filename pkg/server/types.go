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
	"context"
	"net"
	"os"
	"strike/pkg/api/v2"
	"strike/pkg/log"
	"strike/pkg/network"
	"strike/pkg/stream"
	"time"
)

const (
	BasePath = string(os.PathSeparator) + "home" + string(os.PathSeparator) +
		"admin" + string(os.PathSeparator) + "strike"

	LogBasePath = BasePath + string(os.PathSeparator) + "logs"

	LogDefaultPath = LogBasePath + string(os.PathSeparator) + "strike.log"

	PidFileName = "strike.pid"
)

type Config struct {
	ServerName      string
	LogPath         string
	LogLevel        log.Level
	GracefulTimeout time.Duration
	Processor       int
	UseEdgeMode     bool
}

type Server interface {
	AddListener(lc *v2.Listener, networkFiltersFactories []network.NetworkFilterChainFactory,
		streamFiltersFactories []stream.StreamFilterChainFactory) (network.ListenerEventListener, error)

	Start()

	Restart()

	Close()

	Handler() ConnectionHandler
}

type ConnectionHandler interface {
	// NumConnections reports the connections that ConnectionHandler keeps.
	NumConnections() uint64

	// AddOrUpdateListener
	// adds a listener into the ConnectionHandler or
	// update a listener
	AddOrUpdateListener(lc *v2.Listener, networkFiltersFactories []network.NetworkFilterChainFactory,
		streamFiltersFactories []stream.StreamFilterChainFactory) (network.ListenerEventListener, error)

	// StartListener starts a listener by the specified listener tag
	StartListener(lctx context.Context, listenerTag uint64)

	//StartListeners starts all listeners the ConnectionHandler has
	StartListeners(lctx context.Context)

	// FindListenerByAddress finds and returns a listener by the specified network address
	FindListenerByAddress(addr net.Addr) network.Listener

	// FindListenerByName finds and returns a listener by the listener name
	FindListenerByName(name string) network.Listener

	// RemoveListeners find and removes a listener by listener name.
	RemoveListeners(name string)

	// StopListener stops a listener  by listener name
	StopListener(lctx context.Context, name string, stop bool) error

	// StopListeners stops all listeners the ConnectionHandler has.
	// The close indicates whether the listening sockets will be closed.
	StopListeners(lctx context.Context, close bool) error

	// ListListenersFile reports all listeners' fd
	ListListenersFile(lctx context.Context) []*os.File

	// StopConnection Stop Connection
	StopConnection()
}