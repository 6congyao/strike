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

package config

import "flag"

var Nodelay = flag.Int("no-delay", 1, "")
var TCPRecvBuffer = flag.Int("tcp-read-buffer", 1024*1024, "")
var TCPSendBuffer = flag.Int("tcp-send-buffer", 1024*1024, "")

var LocalPort = flag.Int("local-port", 20000, "local loop count")
var Mode = flag.String("mode", "consumer", "mode")
var ProviderPort = flag.Int("provider-port", 30000, "provide agent listen port")
var DefaultAgentCount = flag.Int("agent-count", 1, "default agent connection count")
var ProfileDir = flag.String("profile-dir", "./", "profile dir, set to /root/logs/")

// loop 设置
var ConsumerHttpLoops = flag.Int("consumer-http-loop", 1, "")
var ProviderAgentLoops = flag.Int("provider-agent-loop", 1, "")

//var MaxProcs = flag.Int("max-procs", 8, "")

// processor 设置
var ConsumerHttpProcessors = flag.Int("consumer-http-processor", 1, "")
var ConsumerAgentProcessors = flag.Int("consumer-agent-processor", 1, "")
var ProviderAgentProcessors = flag.Int("provider-agent-processor", 1, "")
var ProviderDubboProcessors = flag.Int("provider-dubbo-processor", 1, "")

var DubboMergeCountMax = flag.Int("dubbo-merge-max", 10, "")
var AgentMergeCountMax = flag.Int("agent-merge-max", 10, "")
var HttpMergeCountMax = flag.Int("http-merge-max", 10, "")

var ProviderWeight = flag.Int("provider-weight", 0, "")

var DebugFlag = flag.Bool("debug-flag", false, "")

var ConnTypeDubbo = 1
var ConnTypeHttp = 2
var ConnTypeAgent = 3
