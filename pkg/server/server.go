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
	"log"
	"os"
	"runtime"
	"strike/pkg/api/v2"
	"strike/pkg/config"
	"strike/pkg/network"
	"strike/pkg/stream"
	"strike/pkg/types"
	"strike/pkg/upstream"
	"time"
)

var servers []*server

type server struct {
	serverName string
	logger     log.Logger
	stopChan   chan struct{}
	handler    ConnectionHandler
}

func NewConfig(c *v2.ServerConfig) *Config {
	return &Config{
		ServerName: c.ServerName,
		LogPath:    c.DefaultLogPath,
		LogLevel:   config.ParseLogLevel(c.DefaultLogLevel),

		GracefulTimeout: c.GracefulTimeout.Duration,
		Processor:       c.Processor,
		UseEdgeMode:     c.UseEdgeMode,
	}
}

// currently, only one server supported
func GetServer() Server {
	if len(servers) == 0 {
		log.Fatalln("Server is nil and hasn't been initiated at this time")
		return nil
	}

	return servers[0]
}

func NewServer(config *Config, cm upstream.ClusterManager) Server {
	if config != nil {
		//processor num setting
		if config.GracefulTimeout != 0 {
			GracefulTimeout = config.GracefulTimeout
		}

		network.UseEdgeMode = config.UseEdgeMode

		if network.UseEdgeMode {
			log.Println("Edge mode enabled.")
		}
	}

	runtime.GOMAXPROCS(config.Processor)

	server := &server{
		serverName: config.ServerName,
		logger:     *log.New(os.Stdout, "strike", log.LstdFlags),
		stopChan:   make(chan struct{}),
		handler:    NewHandler(cm),
	}

	servers = append(servers, server)

	return server
}

func (srv *server) AddListener(lc *v2.Listener, networkFiltersFactories []network.NetworkFilterChainFactory,
	streamFiltersFactories []stream.StreamFilterChainFactory) (network.ListenerEventListener, error) {

	return srv.handler.AddOrUpdateListener(lc, networkFiltersFactories, streamFiltersFactories)
}

func (srv *server) Start() {
	srv.handler.StartListeners(nil)

	for {
		select {
		case <-srv.stopChan:
			return
		}
	}
}

func (srv *server) Restart() {
	// TODO
}

func (srv *server) Close() {
	// stop listener and connections
	srv.handler.StopListeners(nil, true)

	close(srv.stopChan)
}

func (srv *server) Handler() ConnectionHandler {
	return srv.handler
}

func Stop() {
	for _, server := range servers {
		server.Close()
	}
}

func StopAccept() {
	for _, server := range servers {
		server.handler.StopListeners(nil, false)
	}
}

func StopConnection() {
	for _, server := range servers {
		server.handler.StopConnection()
	}
}

func ListListenersFile() []*os.File {
	var files []*os.File
	for _, server := range servers {
		files = append(files, server.handler.ListListenersFile(nil)...)
	}
	return files
}

func WaitConnectionsDone(duration time.Duration) error {
	// one duration wait for connection to active close
	// two duration wait for connection to transfer
	// DefaultConnReadTimeout wait for read timeout
	timeout := time.NewTimer(2*duration + types.DefaultConnReadTimeout)
	StopConnection()
	log.Println("Stop connection...")
	select {
	case <-timeout.C:
		return nil
	}
}