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

package model

// RunMode
const (
	RunModeControl = "CONTROL"
	RunModeMonitor = "MONITOR"
)

// AuthRuleConfig config
type AuthRuleConfig struct {
	RuleConfigs []RuleConfig `json:"rule_configs"`
}

// RuleConfig config
type RuleConfig struct {
	Id      int    `json:"id"`
	Index   int    `json:"index"`
	Name    string `json:"name"`
	AppName string `json:"appName"`
	Enable  bool   `json:"enable"`
	RunMode string `json:"run_mode"`

	AuthConfig      AuthConfig       `json:"auth_config"`
	ResourceConfigs []ResourceConfig `json:"resources"`
}

// Auth config
type AuthConfig struct {
	PublicKey string `json:"public_key"`
	TokenKey  string `json:"token_key"`
}

// ComparisonCofig config
type ComparisonCofig struct {
	CompareType string `json:"compare_type"`
	Key         string `json:"key"`
	Value       string `json:"value"`
}

// ResourceConfig config
type ResourceConfig struct {
	Headers         []ComparisonCofig `json:"headers"`
	HeadersRelation string            `json:"headers_relation"`
	Params          []ComparisonCofig `json:"params"`
	ParamsRelation  string            `json:"params_relation"`
}
