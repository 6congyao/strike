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

package filter

import (
	"fmt"
	"strike/pkg/network"
)

var creatorNetworkFactory map[string]NetworkFilterFactoryCreator

func init() {
	creatorNetworkFactory = make(map[string]NetworkFilterFactoryCreator)
}

// RegisterNetwork registers the filterType as  NetworkFilterFactoryCreator
func RegisterNetwork(filterType string, creator NetworkFilterFactoryCreator) {
	creatorNetworkFactory[filterType] = creator
}

// CreateNetworkFilterChainFactory creates a StreamFilterChainFactory according to filterType
func CreateNetworkFilterChainFactory(filterType string, config map[string]interface{}) (network.NetworkFilterChainFactory, error) {
	if cf, ok := creatorNetworkFactory[filterType]; ok {
		nfcf, err := cf(config)
		if err != nil {
			return nil, fmt.Errorf("create network filter chain factory failed: %v", err)
		}
		return nfcf, nil
	}
	return nil, fmt.Errorf("unsupported network filter type: %v", filterType)
}