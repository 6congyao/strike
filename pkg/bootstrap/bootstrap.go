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

package bootstrap

import (
	"log"
	"strike/pkg/config"
	_ "strike/pkg/filter/network/delegation"
	_ "strike/pkg/filter/network/proxy"
	"strike/pkg/server"
	_ "strike/pkg/stream/http/v1"
)

type Strike struct {
	servers []server.Server
}

func NewStrike(sc *config.StrikeConfig) *Strike {
	sk := &Strike{}
	mode := sc.Mode()

	srvNum := len(sc.Servers)
	if srvNum == 0 {
		log.Fatalln("no server found")
	} else if srvNum > 1 {
		log.Fatalln("multiple server not supported yet, got ", srvNum)
	}

	for _, srvConfig := range sc.Servers {
		c := config.ParseServerConfig(&srvConfig)

		var srv server.Server

		if mode == config.File {
			srv = server.NewServer(c)

			for _, listenerConfig := range srvConfig.Listeners {
				lc := config.ParseListenerConfig(&listenerConfig)
				lc.DisableConnIo = config.GetListenerDisableIO(&lc.FilterChains[0])

				// NetworkFilterChainFactory
				nfcf := config.GetNetworkFilters(&lc.FilterChains[0])

				// Listener
				_, err := srv.AddListener(lc, nfcf)
				if err != nil {
					log.Fatalln("AddListener error:", err.Error())
				}
			}
		}
		sk.servers = append(sk.servers, srv)
	}
	return sk
}

// step1. New strike
// step2. Strike start
func Start(sc *config.StrikeConfig) {
	sk := NewStrike(sc)
	sk.Start()
}

// Start strike servers
// async
func (sk *Strike) Start() {
	for _, srv := range sk.servers {
		go srv.Start()
	}
}

func (sk *Strike) Stop() {
	for _, srv := range sk.servers {
		srv.Close()
	}
}
