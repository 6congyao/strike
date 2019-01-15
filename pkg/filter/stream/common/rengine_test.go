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
	"log"
	"strike/pkg/filter/stream/common/limit"
	"strike/pkg/filter/stream/common/model"
	"strike/pkg/filter/stream/common/resource"
	"strike/pkg/protocol"
	"testing"
	"time"
)

var params = []model.ComparisonCofig{
	{
		Key:         "aa",
		Value:       "va",
		CompareType: resource.CompareEquals,
	},
	{
		Key:         "bb",
		Value:       "vb",
		CompareType: resource.CompareEquals,
	},
}

var resourceConfig = model.ResourceConfig{
	Headers: []model.ComparisonCofig{
		{
			CompareType: resource.CompareEquals,
			Key:         protocol.StrikeHeaderPathKey,
			Value:       "/serverlist/xx.do",
		},
	},
	Params: params,
}

var limitConfig = model.LimitConfig{
	LimitStrategy: limit.QPSStrategy,
	MaxBurstRatio: 1.0,
	PeriodMs:      1000,
	MaxAllows:     10,
}

var ruleConfig = model.RuleConfig{
	Id:              9,
	Name:            "test_666",
	Enable:          true,
	RunMode:         model.RunModeControl,
	ResourceConfigs: []model.ResourceConfig{resourceConfig},
	LimitConfig:     limitConfig,
}

var headers = protocol.CommonHeader{
	protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
	protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb",
}

func TestNewRuleEngine(t *testing.T) {
	ruleEngine := NewRuleEngine(&ruleConfig)

	{
		log.Println("start ticker")
		ruleEngine.invoke(headers)
		//ticker := metrix.NewTicker(func() {
		//	ruleEngine.invoke(headers)
		//})
		//ticker.Start(time.Millisecond * 10)
		//time.Sleep(5 * time.Second)
		//ticker.Stop()
		log.Println("stop ticker")
	}

	ruleEngine.stop()
	log.Println("stop ruleEngine")
	time.Sleep(2 * time.Second)
	log.Println("end")
}
