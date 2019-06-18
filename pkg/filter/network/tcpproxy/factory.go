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
	"strike/pkg/config"
	"strike/pkg/filter"
	"strike/pkg/network"
	"strike/pkg/upstream"
)

func init() {
	filter.RegisterNetwork(v2.TCP_PROXY, CreateTCPProxyFactory)
}

type tcpProxyFilterConfigFactory struct {
	Proxy *v2.TCPProxy
}

func (f *tcpProxyFilterConfigFactory) CreateFilterChain(context context.Context, clusterManager interface{}, callbacks network.NetWorkFilterChainFactoryCallbacks) {
	rf := NewProxy(context, f.Proxy, clusterManager.(upstream.ClusterManager))
	callbacks.AddReadFilter(rf)
}

func CreateTCPProxyFactory(conf map[string]interface{}) (network.NetworkFilterChainFactory, error) {
	p, err := config.ParseTCPProxy(conf)
	if err != nil {
		return nil, err
	}
	return &tcpProxyFilterConfigFactory{
		Proxy: p,
	}, nil
}
