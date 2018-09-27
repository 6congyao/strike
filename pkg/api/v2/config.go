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

package v2

type HostConfig struct {
	Address        string         `json:"address,omitempty"`
	Hostname       string         `json:"hostname,omitempty"`
	Weight         uint32         `json:"weight,omitempty"`
	TLSDisable     bool           `json:"tls_disable,omitempty"`
}

type ListenerConfig struct {
	Name                                  string        `json:"name"`
	AddrConfig                            string        `json:"address"`
	BindToPort                            bool          `json:"bind_port"`
	HandOffRestoredDestinationConnections bool          `json:"handoff_restoreddestination"`
	LogPath                               string        `json:"log_path,omitempty"`
	LogLevelConfig                        string        `json:"log_level,omitempty"`
	Inspector                             bool          `json:"inspector,omitempty"`
}
