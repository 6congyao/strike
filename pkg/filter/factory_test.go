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
	"context"
	"errors"
	"strike/pkg/network"
	"strike/pkg/stream"
	"testing"
)

type testStreamFilterFactory struct{}

func (f *testStreamFilterFactory) CreateFilterChain(context context.Context, callbacks stream.StreamFilterChainFactoryCallbacks) {
}
func testStreamFilterFactoryCreator(config map[string]interface{}) (stream.StreamFilterChainFactory, error) {
	if _, ok := config["error"]; ok {
		return nil, errors.New("error")
	}
	return &testStreamFilterFactory{}, nil
}

type testNetworkFilterFactory struct{}

func (f *testNetworkFilterFactory) CreateFilterChain(context context.Context, clusterManager interface{}, callbacks network.NetWorkFilterChainFactoryCallbacks) {
}
func testNetworkFilterFactoryCreator(config map[string]interface{}) (network.NetworkFilterChainFactory, error) {
	if _, ok := config["error"]; ok {
		return nil, errors.New("error")
	}
	return &testNetworkFilterFactory{}, nil
}

func TestCreateStreamFilterChainFactory(t *testing.T) {
	name := "test"
	RegisterStream(name, testStreamFilterFactoryCreator)
	config := make(map[string]interface{})
	if _, err := CreateStreamFilterChainFactory("no", config); err == nil {
		t.Error("no register type should return an error")
	}
	if _, err := CreateStreamFilterChainFactory(name, config); err != nil {
		t.Error(err)
	}
	config["error"] = true
	if _, err := CreateStreamFilterChainFactory(name, config); err == nil {
		t.Error("create factory failed, expected an error")
	}
}
func TestCreateNetworkFilterChainFactory(t *testing.T) {
	name := "test"
	RegisterNetwork(name, testNetworkFilterFactoryCreator)
	config := make(map[string]interface{})
	if _, err := CreateNetworkFilterChainFactory("no", config); err == nil {
		t.Error("no register type should return an error")
	}
	if _, err := CreateNetworkFilterChainFactory(name, config); err != nil {
		t.Error(err)
	}
	config["error"] = true
	if _, err := CreateNetworkFilterChainFactory(name, config); err == nil {
		t.Error("create factory failed, expected an error")
	}
}
