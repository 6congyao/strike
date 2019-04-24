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

package auth

import (
	"context"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/filter/stream/auth/model"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
)

type authRuleFilter struct {
	context           context.Context
	handler           stream.StreamReceiverFilterHandler
	authRuleConfig    *model.AuthRuleConfig
	RuleEngineFactory *RuleEngineFactory
}

// NewAuthRuleFilter as
func NewAuthRuleFilter(context context.Context, config *model.AuthRuleConfig) stream.StreamReceiverFilter {
	f := &authRuleFilter{
		context:        context,
		authRuleConfig: config,
	}
	if v, exist := factoryInstanceMap.Load(config); exist {
		f.RuleEngineFactory = v.(*RuleEngineFactory)
	}

	return f
}

//implement StreamReceiverFilter
func (f *authRuleFilter) OnReceiveHeaders(context context.Context, headers protocol.HeaderMap, endStream bool) stream.StreamHeadersFilterStatus {
	// do filter
	if f.RuleEngineFactory.invoke(headers) {
		return stream.StreamHeadersFilterContinue
	}
	headers.Set(types.HeaderStatus, strconv.Itoa(types.Unauthorized))
	f.handler.AppendHeaders(headers, true)
	return stream.StreamHeadersFilterStop
}

func (f *authRuleFilter) OnReceiveData(context context.Context, buf buffer.IoBuffer, endStream bool) stream.StreamDataFilterStatus {
	//do filter
	return stream.StreamDataFilterContinue
}

func (f *authRuleFilter) OnReceiveTrailers(context context.Context, trailers protocol.HeaderMap) stream.StreamTrailersFilterStatus {
	//do filter
	return stream.StreamTrailersFilterContinue
}

func (f *authRuleFilter) SetReceiveFilterHandler(handler stream.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *authRuleFilter) OnDestroy() {}
