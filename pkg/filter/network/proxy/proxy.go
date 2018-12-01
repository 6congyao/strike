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

package proxy

import (
	"container/list"
	"context"
	"encoding/json"
	"log"
	"strike/pkg/api/v2"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	"sync"
)

// types.ReadFilter
// types.ServerStreamConnectionEventListener
type proxy struct {
	config *v2.Proxy
	//clusterManager      types.ClusterManager
	readCallbacks       network.ReadFilterCallbacks
	upstreamConnection  network.ClientConnection
	downstreamCallbacks DownstreamCallbacks
	clusterName         string
	//routersWrapper      types.RouterWrapper // wrapper used to point to the routers instance
	serverCodec  stream.ServerStreamConnection
	context      context.Context
	activeSteams *list.List // downstream requests
	asMux        sync.RWMutex
}

// NewProxy create proxy instance for given v2.Proxy config
func NewProxy(ctx context.Context, config *v2.Proxy) Proxy {
	proxy := &proxy{
		config:       config,
		activeSteams: list.New(),
		context:      ctx,
	}

	extJSON, err := json.Marshal(proxy.config.ExtendConfig)
	if err == nil {
		var xProxyExtendConfig v2.XProxyExtendConfig
		json.Unmarshal([]byte(extJSON), &xProxyExtendConfig)
		proxy.context = context.WithValue(proxy.context, types.ContextSubProtocol, xProxyExtendConfig.SubProtocol)
	} else {
		log.Println("get proxy extend config fail:", err)
	}

	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

func (p *proxy) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {
	p.readCallbacks = cb

	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamCallbacks)
	p.serverCodec = stream.CreateServerStreamConnection(p.context, protocol.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
}

func (p *proxy) OnData(buf []byte) network.FilterStatus {
	p.serverCodec.Dispatch(buf)

	return network.Stop
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStream(context context.Context, streamID string, responseSender stream.StreamSender) stream.StreamReceiver {
	return nil
}

//rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event network.ConnectionEvent) {

}

func (p *proxy) OnNewConnection() network.FilterStatus {
	return network.Continue
}

func (p *proxy) ReadDisableUpstream(disable bool) {
	// TODO
}

func (p *proxy) ReadDisableDownstream(disable bool) {
	// TODO
}

// ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event network.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}