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

package delegation

import (
	"context"
	"log"
	"strike/pkg/api/v2"
	"strike/pkg/buffer"
	"strike/pkg/network"
)

type Agent func(content interface{}) error

var agents map[string]Agent

func init() {
	agents = make(map[string]Agent)
}

func RegisterAgent(name string, handler Agent) {
	agents[name] = handler
}

type delegate struct {
	agentName     string
	contentType   string
	readCallbacks network.ReadFilterCallbacks
}

func NewDelegate(ctx context.Context, config *v2.Delegation) network.ReadFilter {
	return &delegate{
		agentName:   config.AgentName,
		contentType: config.ContentType,
	}
}

func (d *delegate) OnNewConnection() network.FilterStatus {
	return network.Continue
}

func (d *delegate) OnData(buffer buffer.IoBuffer) network.FilterStatus {
	return network.Continue
}

func (d *delegate) InitializeReadFilterCallbacks(cb network.ReadFilterCallbacks) {
	d.readCallbacks = cb

	if ag := agents[d.agentName]; ag != nil {
		switch d.contentType {
		case "conn":
			fallthrough
		default:
			if err := ag(d.readCallbacks.Connection().RawConn()); err != nil {
				log.Println("delegation got error:", err)
			}
		}
	}
}
