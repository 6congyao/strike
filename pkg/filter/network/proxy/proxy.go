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
	"runtime"
	"strike/pkg/api/v2"
	"strike/pkg/buffer"
	"strike/pkg/config"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	strikesync "strike/pkg/sync"
	"strike/pkg/types"
	"strike/pkg/upstream"
	"sync"
)

var (
	currProxyID uint32
	workerPool  strikesync.ShardWorkerPool
)

func init() {
	config.RegisterConfigParsedListener(config.ParseCallbackKeyProcessor, initWorkePpool)
}

func initWorkePpool(data interface{}, endParsing bool) error {
	// default shardsNum is equal to the cpu num
	shardsNum := runtime.NumCPU()
	// use 32768 as chan buffer length
	poolSize := shardsNum * 4096

	// set shardsNum equal to processor if it was specified
	if pNum, ok := data.(int); ok && pNum > 0 {
		shardsNum = pNum
	}

	workerPool, _ = strikesync.NewShardWorkerPool(poolSize, shardsNum, eventDispatch)
	workerPool.Init()
	return nil
}

// network.ReadFilter
// stream.ServerStreamConnectionEventListener
type proxy struct {
	config             *v2.Proxy
	cm                 upstream.ClusterManager
	readCallbacks      network.ReadFilterCallbacks
	upstreamConnection network.ClientConnection
	downstreamListener network.ConnectionEventListener
	clusterName        string
	//routersWrapper      types.RouterWrapper // wrapper used to point to the routers instance
	serverStreamConn stream.ServerStreamConnection
	context          context.Context
	activeSteams     *list.List // downstream requests
	asMux            sync.RWMutex
}

// NewProxy create proxy instance for given v2.Proxy config
func NewProxy(ctx context.Context, config *v2.Proxy, clusterManager interface{}) Proxy {
	proxy := &proxy{
		config:       config,
		activeSteams: list.New(),
		context:      ctx,
		cm:           clusterManager.(upstream.ClusterManager),
	}

	extJSON, err := json.Marshal(proxy.config.ExtendConfig)
	if err == nil {
		var xProxyExtendConfig v2.XProxyExtendConfig
		json.Unmarshal([]byte(extJSON), &xProxyExtendConfig)
		proxy.context = context.WithValue(proxy.context, types.ContextSubProtocol, xProxyExtendConfig.SubProtocol)
	} else {
		log.Println("get proxy extend config fail:", err)
	}

	proxy.downstreamListener = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

func (p *proxy) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {
	p.readCallbacks = cb
	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamListener)

	if p.config.DownstreamProtocol != string(protocol.Auto) {
		p.serverStreamConn = stream.CreateServerStreamConnection(p.context, protocol.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
	}
}

func (p *proxy) OnData(buf buffer.IoBuffer) network.FilterStatus {
	if p.serverStreamConn == nil {
		p.serverStreamConn = stream.CreateServerStreamConnection(p.context, protocol.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
	}
	p.serverStreamConn.Dispatch(buf)

	return network.Stop
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStreamDetect(ctx context.Context, responseSender stream.StreamSender) stream.StreamReceiveListener {
	s := newActiveStream(ctx, p, responseSender)

	if ff := p.context.Value(types.ContextKeyStreamFilterChainFactories); ff != nil {
		ffs := ff.([]stream.StreamFilterChainFactory)
		for _, f := range ffs {
			f.CreateFilterChain(p.context, s)
		}
	}

	p.asMux.Lock()
	s.element = p.activeSteams.PushBack(s)
	p.asMux.Unlock()

	return s
}

//rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event network.ConnectionEvent) {
	if event.IsClose() {
		var urEleNext *list.Element

		p.asMux.RLock()
		defer p.asMux.RUnlock()

		for urEle := p.activeSteams.Front(); urEle != nil; urEle = urEleNext {
			urEleNext = urEle.Next()

			ds := urEle.Value.(*downStream)
			ds.OnResetStream(stream.StreamConnectionTermination)
		}
	}
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

func (p *proxy) deleteActiveStream(s *downStream) {
	if s.element != nil {
		p.asMux.Lock()
		p.activeSteams.Remove(s.element)
		p.asMux.Unlock()
	}
}

func (p *proxy) convertProtocol() (dp, up protocol.Protocol) {
	if p.serverStreamConn == nil {
		dp = protocol.Protocol(p.config.DownstreamProtocol)
	} else {
		dp = p.serverStreamConn.Protocol()
	}
	up = protocol.Protocol(p.config.UpstreamProtocol)
	if up == protocol.Auto {
		up = dp
	}
	return
}

// network.ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event network.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}
