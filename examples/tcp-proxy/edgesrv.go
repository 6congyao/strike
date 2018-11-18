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

package main

import (
	"bytes"
	"fmt"
	"strike/pkg/evio"
	"strike/pkg/network"
	"sync"
)

var connMapEdge sync.Map

func main() {
	EdgeServe()
}

func EdgeServe() error {
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
		session := network.NewSession(econn.RemoteAddr())

		// keep track of the client
		econn.SetContext(session)

		// add client to server map
		fmt.Println("Session opened:", session.RemoteAddr())
		connMapEdge.Store(session.ID(), session)
		return
	}

	events.Data = func(econn evio.Conn, in []byte) (out []byte, action evio.Action) {
		session := econn.Context().(*network.Session)
		p := session.In.Begin(in)
		pr := session.PipelineReader()
		rbuf := bytes.NewBuffer(p)
		pr.Rd = rbuf
		pr.Wr = session

		msgs, err := pr.ReadMessages()
		if err != nil {
			action = evio.Close
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

	return evio.Serve(events, "127.0.0.1:2045")
}
