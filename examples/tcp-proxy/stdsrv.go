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
	"net"
	"strike/pkg/network"
	"strings"
	"sync"
)

var connMap sync.Map

func main() {
	StdServe()
}

func StdServe() error {
	ln, err := net.Listen("tcp", "127.0.0.1:2045")
	if err != nil {
		return err
	}
	defer ln.Close()
	fmt.Println("Ready to accept conn at:", ln.Addr())

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		go func(conn net.Conn) {
			defer func() {
				conn.Close()
			}()

			session := network.NewSession(conn.RemoteAddr())
			fmt.Println("Session opened:", session.RemoteAddr())
			connMap.Store(session.ID(), session)

			packet := make([]byte, 0xFFFF)
			for {
				var close bool
				n, err := conn.Read(packet)

				if err != nil {
					return
				}
				in := packet[:n]

				p := session.In.Begin(in)
				pr := session.PipelineReader()
				rbuf := bytes.NewBuffer(p)
				pr.Rd = rbuf
				pr.Wr = session

				msgs, err := pr.ReadMessages()
				if err != nil {
					return
				}

				for _, msg := range msgs {
					if msg != nil && msg.Command() != "" {
						fmt.Println("got msg:", msg)
					}
				}

				p = packet[len(p)-rbuf.Len():]
				session.In.End(p)
				if close {
					break
				}
			}
		}(conn)
	}

	return nil
}

var deniedMessage = []byte(strings.Replace(strings.TrimSpace(`
-DENIED Tile38 is running in protected mode because protected mode is enabled,
no bind address was specified, no authentication password is requested to
clients. In this mode connections are only accepted from the loopback
interface. If you want to connect from external computers to Tile38 you may
adopt one of the following solutions: 1) Just disable protected mode sending
the command 'CONFIG SET protected-mode no' from the loopback interface by
connecting to Tile38 from the same host the server is running, however MAKE
SURE Tile38 is not publicly accessible from internet if you do so. Use CONFIG
REWRITE to make this change permanent. 2) Alternatively you can just disable
the protected mode by editing the Tile38 configuration file, and setting the
protected mode option to 'no', and then restarting the server. 3) If you
started the server manually just for testing, restart it with the
'--protected-mode no' option. 4) Setup a bind address or an authentication
password. NOTE: You only need to do one of the above things in order for the
server to start accepting connections from the outside.
`), "\n", " ", -1) + "\r\n")
