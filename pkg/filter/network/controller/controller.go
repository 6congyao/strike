/*
 * Copyright (c) 2019. LuCongyao <6congyao@gmail.com> .
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

package controller

import (
	"context"
	"strike/pkg/api/v2"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
)

var (
	currControllerID uint32
)

type controller struct {
	config             *v2.Controller
	readCallbacks      network.ReadFilterCallbacks
	serverStreamConn   stream.ServerStreamConnection
	downstreamListener network.ConnectionEventListener
	context            context.Context
}

// NewProxy create proxy instance for given v2.Proxy config
func NewController(ctx context.Context, config *v2.Controller) network.ReadFilter {
	controller := &controller{
		config:  config,
		context: ctx,
	}

	controller.downstreamListener = &downstreamCallbacks{
		controller: controller,
	}

	return controller
}

//rpc realize upstream on event
func (c *controller) onDownstreamEvent(event network.ConnectionEvent) {
	if event.IsClose() {
	}
}

func (c *controller) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {
	c.readCallbacks = cb
	c.readCallbacks.Connection().AddConnectionEventListener(c.downstreamListener)

	if c.config.SourceProtocol != string(protocol.Auto) {
		c.serverStreamConn = stream.CreateServerStreamConnection(c.context, protocol.Protocol(c.config.SourceProtocol), c.readCallbacks.Connection(), c)
	}
}

func (c *controller) OnData(buf buffer.IoBuffer) network.FilterStatus {
	if c.serverStreamConn == nil {
		c.serverStreamConn = stream.CreateServerStreamConnection(c.context, protocol.Protocol(c.config.SourceProtocol), c.readCallbacks.Connection(), c)
	}
	c.serverStreamConn.Dispatch(buf)

	return network.Stop
}

func (c *controller) OnNewConnection() network.FilterStatus {
	return network.Continue
}

func (c *controller) OnGoAway() {}

func (c *controller) NewStreamDetect(ctx context.Context, responseSender stream.StreamSender) stream.StreamReceiveListener {
	s := newActiveStream(ctx, c, responseSender)

	if ff := c.context.Value(types.ContextKeyStreamFilterChainFactories); ff != nil {
		ffs := ff.([]stream.StreamFilterChainFactory)
		for _, f := range ffs {
			f.CreateFilterChain(c.context, s)
		}
	}

	return s
}

// ConnectionEventListener
type downstreamCallbacks struct {
	controller *controller
}

func (dc *downstreamCallbacks) OnEvent(event network.ConnectionEvent) {
	dc.controller.onDownstreamEvent(event)
}

