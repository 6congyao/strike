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
	"strike/pkg/filter/stream/common/limit"
	"strike/pkg/filter/stream/common/model"
	"strike/pkg/filter/stream/common/resource"
	"strike/pkg/protocol"
)

// RuleEngine as
type RuleEngine struct {
	ruleConfig    *model.RuleConfig
	matcherEngine resource.MatcherEngine
	limitEngine   *limit.LimitEngine
	//stat          *metrix.Stat
}

// NewRuleEngine new
func NewRuleEngine(config *model.RuleConfig) *RuleEngine {
	ruleEngine := &RuleEngine{
		ruleConfig: config,
	}

	limitEngine, err := limit.NewLimitEngine(config)
	if err != nil {
		return nil
	}
	ruleEngine.limitEngine = limitEngine
	ruleEngine.matcherEngine = resource.NewMatcherEnine(nil)
	//ruleEngine.stat = metrix.NewStat(config)
	return ruleEngine
}

func (e *RuleEngine) invoke(headers protocol.HeaderMap) bool {
	if e.match(headers) {
		//e.stat.Counter(metrix.INVOKE).Inc(1)
		if e.limitEngine.OverLimit() {
			//e.stat.Counter(metrix.BLOCK).Inc(1)
			if e.ruleConfig.RunMode == model.RunModeControl {
				return false
			}
		}
	}
	return true
}

func (e *RuleEngine) match(headers protocol.HeaderMap) bool {
	return e.matcherEngine.Match(headers, e.ruleConfig)
}

//stop timer
func (e *RuleEngine) stop() {
	//e.stat.Stop()
}
