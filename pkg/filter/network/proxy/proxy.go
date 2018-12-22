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
	// default shardsNum is equal to the cpu num
	shardsNum := runtime.NumCPU()
	// use 4096 as chan buffer length
	poolSize := shardsNum * 4096

	workerPool, _ = strikesync.NewShardWorkerPool(poolSize, shardsNum, eventDispatch)
	workerPool.Init()
}

// network.ReadFilter
// stream.ServerStreamConnectionEventListener
type proxy struct {
	config              *v2.Proxy
	cm                  upstream.ClusterManager
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

	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

func (p *proxy) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {
	p.readCallbacks = cb
	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamCallbacks)
}

func (p *proxy) OnData(buf buffer.IoBuffer) network.FilterStatus {
	if p.serverCodec == nil {
		p.serverCodec = stream.CreateServerStreamConnection(p.context, protocol.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
	}
	p.serverCodec.Dispatch(buf)

	return network.Stop
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStreamDetect(ctx context.Context, streamID string, responseSender stream.StreamSender) stream.StreamReceiver {
	s := newActiveStream(ctx, streamID, p, responseSender)

	//todo: stream filter
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
		downStreams := make([]*downStream, 0, p.activeSteams.Len())
		for urEle := p.activeSteams.Front(); urEle != nil; urEle = urEleNext {
			urEleNext = urEle.Next()

			ds := urEle.Value.(*downStream)
			downStreams = append(downStreams, ds)
		}
		p.asMux.RUnlock()

		for _, ds := range downStreams {
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
	if p.serverCodec == nil {
		dp = protocol.Protocol(p.config.DownstreamProtocol)
	} else {
		dp = p.serverCodec.Protocol()
	}
	up = protocol.Protocol(p.config.UpstreamProtocol)
	if up == protocol.AUTO {
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
