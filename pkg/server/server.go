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
	"strike/pkg/network"
)

var servers []*server

type server struct {
	serverName string
	logger     log.Logger
	stopChan   chan struct{}
	handler    network.ConnectionHandler
}

// currently, only one server supported
func GetServer() Server {
	if len(servers) == 0 {
		log.Fatalln("Server is nil and hasn't been initiated at this time")
		return nil
	}

	return servers[0]
}

func NewServer(config *Config) Server {
	procNum := runtime.NumCPU()

	if config != nil {
		//processor num setting
		if config.Processor > 0 {
			procNum = config.Processor
		}

		//network.UseNetpollMode = config.UseNetpollMode
		if config.UseNetpollMode {
			log.Println("Netpoll mode enabled.")
		}
	}

	runtime.GOMAXPROCS(procNum)

	server := &server{
		serverName: config.ServerName,
		logger:     *log.New(os.Stdout, "strike", log.LstdFlags),
		stopChan:   make(chan struct{}),
		handler:    NewHandler(),
	}

	servers = append(servers, server)

	return server
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

func (srv *server) Handler() network.ConnectionHandler {
	return srv.handler
}