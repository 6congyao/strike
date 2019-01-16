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

package proxy

import (
	"context"
	"strike/pkg/api/v2"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
	"testing"
)

type mockResponseSender struct {
	// receive data
	headers  protocol.HeaderMap
	data     buffer.IoBuffer
	trailers protocol.HeaderMap
}

func (s *mockResponseSender) AppendHeaders(ctx context.Context, headers protocol.HeaderMap, endStream bool) error {
	s.headers = headers
	return nil
}

func (s *mockResponseSender) AppendData(ctx context.Context, data buffer.IoBuffer, endStream bool) error {
	s.data = data
	return nil
}

func (s *mockResponseSender) AppendTrailers(ctx context.Context, trailers protocol.HeaderMap) error {
	s.trailers = trailers
	return nil
}

func (s *mockResponseSender) GetStream() stream.Stream {
	return &mockStream{}
}

type mockStream struct {
	stream.Stream
}

func (s *mockStream) ResetStream(reason stream.StreamResetReason) {
	// do nothing
}

type mockReadFilterCallbacks struct {
	network.ReadFilterCallbacks
}

func (cb *mockReadFilterCallbacks) Connection() network.Connection {
	return &mockConnection{}
}

type mockConnection struct {
	network.Connection
}

func (c *mockConnection) ID() uint64 {
	return 0
}

func TestDirectResponse(t *testing.T) {
	testCases := []struct {
		client *mockResponseSender
		//route  *mockRoute
		check func(t *testing.T, sender *mockResponseSender)
	}{
		// without body
		{
			client: &mockResponseSender{},
			//route: &mockRoute{
			//	direct: &mockDirectRule{
			//		status: 500,
			//	},
			//},
			check: func(t *testing.T, client *mockResponseSender) {
				if client.headers == nil {
					t.Fatal("want to receive a header response")
				}
				if code, ok := client.headers.Get(types.HeaderStatus); !ok || code != "500" {
					t.Error("response status code not expected")
				}
			},
		},
		// with body
		{
			client: &mockResponseSender{},
			//route: &mockRoute{
			//	direct: &mockDirectRule{
			//		status: 400,
			//		body:   "mock 400 response",
			//	},
			//},
			check: func(t *testing.T, client *mockResponseSender) {
				if client.headers == nil {
					t.Fatal("want to receive a header response")
				}
				if code, ok := client.headers.Get(types.HeaderStatus); !ok || code != "400" {
					t.Error("response status code not expected")
				}
				if client.data == nil {
					t.Fatal("want to receive a body response")
				}
				if client.data.String() != "mock 400 response" {
					t.Error("response  data not expected")
				}
			},
		},
	}
	for _, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				config: &v2.Proxy{},
				//routersWrapper: &mockRouterWrapper{
				//	routers: &mockRouters{
				//		route: tc.route,
				//	},
				//},
				//clusterManager: &mockClusterManager{},
				readCallbacks: &mockReadFilterCallbacks{},
			},
			responseSender: tc.client,
		}
		// event call Receive Headers
		// trigger direct response
		s.ReceiveHeaders(nil, false)
		// check
		tc.check(t, tc.client)
	}
}
