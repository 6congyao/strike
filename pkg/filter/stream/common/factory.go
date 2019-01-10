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

package common

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strike/pkg/api/v2"
	"strike/pkg/buffer"
	"strike/pkg/filter"
	"strike/pkg/filter/stream/common/model"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
)

func init() {
	filter.RegisterStream(v2.COMMON, CreateCommonRuleFilterFactory)
}

type commonRuleFilterFactory struct {
	commonRuleConfig *model.CommonRuleConfig
}

func (f *commonRuleFilterFactory) CreateFilterChain(context context.Context, callbacks stream.StreamFilterChainFactoryCallbacks) {
	filter := NewCommonRuleFilter(context, f.commonRuleConfig)
	callbacks.AddStreamReceiverFilter(filter)
}

// CreateCommonRuleFilterFactory as
func CreateCommonRuleFilterFactory(conf map[string]interface{}) (stream.StreamFilterChainFactory, error) {
	f := &commonRuleFilterFactory{
		commonRuleConfig: parseCommonRuleConfig(conf),
	}
	NewFacatoryInstance(f.commonRuleConfig)
	return f, nil
}

type commmonRuleFilter struct {
	context           context.Context
	cb                stream.StreamReceiverFilterCallbacks
	commonRuleConfig  *model.CommonRuleConfig
	RuleEngineFactory *RuleEngineFactory
}

// NewCommonRuleFilter as
func NewCommonRuleFilter(context context.Context, config *model.CommonRuleConfig) stream.StreamReceiverFilter {
	f := &commmonRuleFilter{
		context:          context,
		commonRuleConfig: config,
	}
	f.RuleEngineFactory = factoryInstance
	return f
}

//implement StreamReceiverFilter
func (f *commmonRuleFilter) OnDecodeHeaders(headers protocol.HeaderMap, endStream bool) stream.StreamHeadersFilterStatus {
	// do filter
	if f.RuleEngineFactory.invoke(headers) {
		return stream.StreamHeadersFilterContinue
	}
	headers.Set(types.HeaderStatus, strconv.Itoa(types.LimitExceededCode))
	f.cb.AppendHeaders(headers, true)
	return stream.StreamHeadersFilterStop
}

func (f *commmonRuleFilter) OnDecodeData(buf buffer.IoBuffer, endStream bool) stream.StreamDataFilterStatus {
	//do filter
	return stream.StreamDataFilterContinue
}

func (f *commmonRuleFilter) OnDecodeTrailers(trailers protocol.HeaderMap) stream.StreamTrailersFilterStatus {
	//do filter
	return stream.StreamTrailersFilterContinue
}

func (f *commmonRuleFilter) SetDecoderFilterCallbacks(cb stream.StreamReceiverFilterCallbacks) {
	f.cb = cb
}

func (f *commmonRuleFilter) OnDestroy() {}

func parseCommonRuleConfig(config map[string]interface{}) *model.CommonRuleConfig {
	commonRuleConfig := &model.CommonRuleConfig{}

	if data, err := json.Marshal(config); err == nil {
		json.Unmarshal(data, commonRuleConfig)
	} else {
		log.Fatalln("[common] parsing common stream filter failed")
	}
	return commonRuleConfig
}

var factoryInstance *RuleEngineFactory

func NewFacatoryInstance(config *model.CommonRuleConfig) {
	factoryInstance = NewRuleEngineFactory(config)
	//log.Println("newFacatoryInstance:", factoryInstance)
}

// RuleEngineFactory as
type RuleEngineFactory struct {
	CommonRuleConfig *model.CommonRuleConfig
	ruleEngines      []RuleEngine
}

// NewRuleEngineFactory new
func NewRuleEngineFactory(config *model.CommonRuleConfig) *RuleEngineFactory {
	f := &RuleEngineFactory{
		CommonRuleConfig: config,
	}

	for _, ruleConfig := range config.RuleConfigs {
		if ruleConfig.Enable {
			ruleEngine := NewRuleEngine(&ruleConfig)
			if ruleEngine != nil {
				f.ruleEngines = append(f.ruleEngines, *ruleEngine)
			}
		}
	}

	return f
}

func (f *RuleEngineFactory) invoke(headers protocol.HeaderMap) bool {
	for _, ruleEngine := range f.ruleEngines {
		if !ruleEngine.invoke(headers) {
			return false
		}
	}
	return true
}

func (f *RuleEngineFactory) stop() {
	for _, ruleEngine := range f.ruleEngines {
		ruleEngine.stop()
	}
}
