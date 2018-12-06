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
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/types"
)

func init() {
	filter.RegisterStream(v2.COMMON, CreateCommonRuleFilterFactory)
}

type commonRuleFilterFactory struct {
	commonRuleConfig *CommonRuleConfig
}

func (f *commonRuleFilterFactory) CreateFilterChain(context context.Context, callbacks stream.StreamFilterChainFactoryCallbacks) {
	filter := NewCommonRuleFilter(context, f.commonRuleConfig)
	callbacks.AddStreamReceiverFilter(filter)
}

type commmonRuleFilter struct {
	context           context.Context
	cb                stream.StreamReceiverFilterCallbacks
	commonRuleConfig  *CommonRuleConfig
	RuleEngineFactory *RuleEngineFactory
}

// NewCommonRuleFilter as
func NewCommonRuleFilter(context context.Context, config *CommonRuleConfig) stream.StreamReceiverFilter {
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

// CreateCommonRuleFilterFactory as
func CreateCommonRuleFilterFactory(conf map[string]interface{}) (stream.StreamFilterChainFactory, error) {
	f := &commonRuleFilterFactory{
		commonRuleConfig: parseCommonRuleConfig(conf),
	}
	NewFacatoryInstance(f.commonRuleConfig)
	return f, nil
}

func parseCommonRuleConfig(config map[string]interface{}) *CommonRuleConfig {
	commonRuleConfig := &CommonRuleConfig{}

	if data, err := json.Marshal(config); err == nil {
		json.Unmarshal(data, commonRuleConfig)
	} else {
		log.Fatalln("[common] parsing common stream filter failed")
	}
	return commonRuleConfig
}

var factoryInstance *RuleEngineFactory

// NewFacatoryInstance as
func NewFacatoryInstance(config *CommonRuleConfig) {
	factoryInstance = NewRuleEngineFactory(config)
	log.Println("newFacatoryInstance:", factoryInstance)
}

// RuleEngine as
type RuleEngine struct {
	ruleConfig *RuleConfig
}

// NewRuleEngine new
func NewRuleEngine(config *RuleConfig) *RuleEngine {
	ruleEngine := &RuleEngine{
		ruleConfig: config,
	}

	return ruleEngine
}

func (e *RuleEngine) invoke(headers protocol.HeaderMap) bool {
	if e.match(headers) {
		//e.stat.Counter(metrix.INVOKE).Inc(1)
		//if e.limitEngine.OverLimit() {
		//	e.stat.Counter(metrix.BLOCK).Inc(1)
		//	if e.ruleConfig.RunMode == model.RunModeControl {
		//		return false
		//	}
		//}
	}
	return true
}

func (e RuleEngine) match(headers protocol.HeaderMap) bool {
	//return e.matcherEngine.Match(headers, e.ruleConfig)
	return true
}

//stop timer
func (e RuleEngine) stop() {
	//e.stat.Stop()
}

// RuleEngineFactory as
type RuleEngineFactory struct {
	CommonRuleConfig *CommonRuleConfig
	ruleEngines      []RuleEngine
}

// NewRuleEngineFactory new
func NewRuleEngineFactory(config *CommonRuleConfig) *RuleEngineFactory {
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
