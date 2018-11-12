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

package delegation

import (
	"context"
	"strike/pkg/api/v2"
	"strike/pkg/filter"
	"strike/pkg/network"
)

func init() {
	filter.RegisterNetwork(v2.DELEGATION, CreateDelegationFactory)
}

type delegationFilterConfigFactory struct {
	Delegation []byte
}

func (f *delegationFilterConfigFactory) CreateFilterChain(context context.Context, callbacks network.NetWorkFilterChainFactoryCallbacks) {

}

func CreateDelegationFactory(conf map[string]interface{}) (network.NetworkFilterChainFactory, error) {
	return &delegationFilterConfigFactory{}, nil
}
