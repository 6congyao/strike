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
	"strike/pkg/buffer"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"testing"
)

// Mock stream filters
type mockStreamReceiverFilter struct {
	cb stream.StreamReceiverFilterCallbacks
	// api called count
	onHeaders  int
	onData     int
	onTrailers int
	// returns status
	headersStatus  stream.StreamHeadersFilterStatus
	dataStatus     stream.StreamDataFilterStatus
	trailersStatus stream.StreamTrailersFilterStatus
}

func (f *mockStreamReceiverFilter) OnDestroy() {}

func (f *mockStreamReceiverFilter) OnDecodeHeaders(headers protocol.HeaderMap, endStream bool) stream.StreamHeadersFilterStatus {
	f.onHeaders++
	return f.headersStatus
}

func (f *mockStreamReceiverFilter) OnDecodeData(buf buffer.IoBuffer, endStream bool) stream.StreamDataFilterStatus {
	f.onData++
	return f.dataStatus
}

func (f *mockStreamReceiverFilter) OnDecodeTrailers(trailers protocol.HeaderMap) stream.StreamTrailersFilterStatus {
	f.onTrailers++
	return f.trailersStatus
}

func (f *mockStreamReceiverFilter) SetDecoderFilterCallbacks(cb stream.StreamReceiverFilterCallbacks) {
	f.cb = cb
}

// Strike receive a request, run StreamReceiverFilters, and send request to upstream
func TestRunReiverFilters(t *testing.T) {
	testCases := []struct {
		filters []*mockStreamReceiverFilter
	}{
		{
			filters: []*mockStreamReceiverFilter{
				// this filter returns all continue, like mixer filter or fault inject filter not matched condition
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterContinue,
					dataStatus:     stream.StreamDataFilterContinue,
					trailersStatus: stream.StreamTrailersFilterContinue,
				},
				// this filter like fault inject filter matched condition
				// in fault inject, it will call ContinueDecoding/SendHijackReply
				// this test will ignore it
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterStop,
					dataStatus:     stream.StreamDataFilterStopAndBuffer,
					trailersStatus: stream.StreamTrailersFilterStop,
				},
			},
		},
		// The Header filter returns stop to run next filter,
		// but the data/trailer filter wants to be continue
		{
			filters: []*mockStreamReceiverFilter{
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterStop,
					dataStatus:     stream.StreamDataFilterContinue,
					trailersStatus: stream.StreamTrailersFilterContinue,
				},
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterContinue,
					dataStatus:     stream.StreamDataFilterContinue,
					trailersStatus: stream.StreamTrailersFilterContinue,
				},
				// to prevent proxy. if a real stream filter returns all stop,
				// it should call ContinueDecoding or SendHijackReply, or the stream will be hung up
				// this test will ignore it
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterStop,
					dataStatus:     stream.StreamDataFilterStop,
					trailersStatus: stream.StreamTrailersFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamReceiverFilter{
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterStop,
					dataStatus:     stream.StreamDataFilterStop,
					trailersStatus: stream.StreamTrailersFilterContinue,
				},
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterContinue,
					dataStatus:     stream.StreamDataFilterContinue,
					trailersStatus: stream.StreamTrailersFilterContinue,
				},
				// to prevent proxy. if a real stream filter returns all stop,
				// it should call ContinueDecoding or SendHijackReply, or the stream will be hung up
				// this test will ignore it
				&mockStreamReceiverFilter{
					headersStatus:  stream.StreamHeadersFilterStop,
					dataStatus:     stream.StreamDataFilterStop,
					trailersStatus: stream.StreamTrailersFilterStop,
				},
			},
		},
	}
	for i, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				//routersWrapper: &mockRouterWrapper{},
				//clusterManager: &mockClusterManager{},
			},
		}
		for _, f := range tc.filters {
			s.AddStreamReceiverFilter(f)
		}
		// mock run
		s.doReceiveHeaders(nil, nil, false)
		// to continue data
		s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		s.doReceiveData(nil, s.downstreamReqDataBuf, false)
		// to continue trailer
		s.downstreamReqTrailers = protocol.CommonHeader{}
		s.doReceiveTrailers(nil, s.downstreamReqTrailers)
		for j, f := range tc.filters {
			if !(f.onHeaders == 1 && f.onData == 1 && f.onTrailers == 1) {
				t.Errorf("#%d.%d stream filter is not called; OnHeader:%d, OnData:%d, OnTrailer:%d", i, j, f.onHeaders, f.onData, f.onTrailers)
			}
		}
	}
}
