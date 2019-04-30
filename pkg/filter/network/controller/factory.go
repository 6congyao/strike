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
	"strike/pkg/config"
	"strike/pkg/filter"
	"strike/pkg/network"
)

func init() {
	filter.RegisterNetwork(v2.CONTROLLER_NETWORK_FILTER, CreateControllerFactory)
}

type genericControllerFilterConfigFactory struct {
	Controller *v2.Controller
}

func (gfcf *genericControllerFilterConfigFactory) CreateFilterChain(context context.Context, clusterManager interface{}, callbacks network.NetWorkFilterChainFactoryCallbacks) {
	p := NewController(context, gfcf.Controller)
	callbacks.AddReadFilter(p)
}

func CreateControllerFactory(conf map[string]interface{}) (network.NetworkFilterChainFactory, error) {
	return &genericControllerFilterConfigFactory{
		Controller: config.ParseControllerFilter(conf),
	}, nil
}