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

package resource

import (
	"strike/pkg/filter/stream/common/model"
	"strike/pkg/protocol"
	"testing"
)

var params = []model.ComparisonCofig{
	{
		Key:         "aa",
		Value:       "va",
		CompareType: CompareEquals,
	},
	{
		Key:         "bb",
		Value:       "vb",
		CompareType: CompareEquals,
	},
}

var params2 = []model.ComparisonCofig{
	{
		Key:         "aa",
		Value:       "va",
		CompareType: CompareNotEquals,
	},
	{
		Key:         "bb",
		Value:       "vb",
		CompareType: CompareNotEquals,
	},
}

func TestDefaultMatcher_Match(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match1(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match2(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do1",
		protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match3(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match4(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params:         params,
		ParamsRelation: RelationOr,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match5(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params:         params,
		ParamsRelation: RelationOr,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match11(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params2,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match12(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params2,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match13(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
			{
				CompareType: CompareEquals,
				Key:         protocol.StrikeHeaderPathKey,
				Value:       "/serverlist/xx.do",
			},
		},
		Params: params2,
	}

	headers := protocol.CommonHeader{
		protocol.StrikeHeaderPathKey:        "/serverlist/xx.do",
		protocol.StrikeHeaderQueryStringKey: "aa=va1&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}
