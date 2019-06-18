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

package tcpproxy

import (
	"context"
	"strike/pkg/api/v2"
	"strike/pkg/network"
	"strike/pkg/upstream"
)

// network.ReadFilter
type tcpproxy struct {
	config              ProxyConfig
	clusterManager      upstream.ClusterManager
	readCallbacks       network.ReadFilterCallbacks
	upstreamConnection  network.ClientConnection
	upstreamCallbacks   UpstreamCallbacks
	downstreamCallbacks DownstreamCallbacks

	upstreamConnecting bool
}

func NewProxy(ctx context.Context, config *v2.TCPProxy, clusterManager upstream.ClusterManager) TCPProxy {
	//p := &tcpproxy{
	//	config:         NewProxyConfig(config),
	//	clusterManager: clusterManager,
	//}
	//
	//p.upstreamCallbacks = &upstreamCallbacks{
	//	proxy: p,
	//}
	//p.downstreamCallbacks = &downstreamCallbacks{
	//	proxy: p,
	//}
	//
	//return p
	return nil
}
