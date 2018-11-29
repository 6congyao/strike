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
	"context"
	"strike/pkg/api/v2"
	"strike/pkg/config"
	"strike/pkg/filter"
	"strike/pkg/network"
	"strike/pkg/proxy"
)

func init() {
	filter.RegisterNetwork(v2.DEFAULT_NETWORK_FILTER, CreateProxyFactory)
}

type genericProxyFilterConfigFactory struct {
	Proxy *v2.Proxy
}

func (gfcf *genericProxyFilterConfigFactory) CreateFilterChain(context context.Context, callbacks network.NetWorkFilterChainFactoryCallbacks) {
	p := proxy.NewProxy(context, gfcf.Proxy)
	callbacks.AddReadFilter(p)
}

func CreateProxyFactory(conf map[string]interface{}) (network.NetworkFilterChainFactory, error) {
	return &genericProxyFilterConfigFactory{
		Proxy: config.ParseProxyFilter(conf),
	}, nil
}
