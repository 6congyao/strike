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

package common

import (
	"context"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/filter/stream/common/model"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
)

type commmonRuleFilter struct {
	context           context.Context
	handler           stream.StreamReceiverFilterHandler
	commonRuleConfig  *model.CommonRuleConfig
	RuleEngineFactory *RuleEngineFactory
}

// NewCommonRuleFilter as
func NewCommonRuleFilter(context context.Context, config *model.CommonRuleConfig) stream.StreamReceiverFilter {
	f := &commmonRuleFilter{
		context:          context,
		commonRuleConfig: config,
	}
	if v, exist := factoryInstanceMap.Load(config); exist {
		f.RuleEngineFactory = v.(*RuleEngineFactory)
	}

	return f
}

//implement StreamReceiverFilter
func (f *commmonRuleFilter) OnReceiveHeaders(ctx context.Context, headers protocol.HeaderMap, endStream bool) stream.StreamHeadersFilterStatus {
	// do filter
	if f.RuleEngineFactory.invoke(headers) {
		return stream.StreamHeadersFilterContinue
	}
	headers.Set(types.HeaderStatus, strconv.Itoa(types.LimitExceededCode))
	f.handler.AppendHeaders(headers, true)
	return stream.StreamHeadersFilterStop
}

func (f *commmonRuleFilter) OnReceiveData(ctx context.Context, buf buffer.IoBuffer, endStream bool) stream.StreamDataFilterStatus {
	//do filter
	return stream.StreamDataFilterContinue
}

func (f *commmonRuleFilter) OnReceiveTrailers(ctx context.Context, trailers protocol.HeaderMap) stream.StreamTrailersFilterStatus {
	//do filter
	return stream.StreamTrailersFilterContinue
}

func (f *commmonRuleFilter) SetReceiveFilterHandler(handler stream.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *commmonRuleFilter) OnDestroy() {}
