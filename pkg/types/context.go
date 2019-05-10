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

package types

// ContextKey type
type ContextKey string

// Context key types
const (
	ContextKeyStreamID                    ContextKey = "StreamId"
	ContextKeyConnectionID                ContextKey = "ConnectionId"
	ContextKeyListenerPort                ContextKey = "ListenerPort"
	ContextKeyListenerName                ContextKey = "ListenerName"
	ContextKeyListenerType                ContextKey = "ListenerType"
	ContextKeyListenerStatsNameSpace      ContextKey = "ListenerStatsNameSpace"
	ContextKeyNetworkFilterChainFactories ContextKey = "NetworkFilterChainFactory"
	ContextKeyStreamFilterChainFactories  ContextKey = "StreamFilterChainFactory"
	ContextKeyBufferPoolCtx               ContextKey = "ConnectionBufferPoolCtx"
	ContextKeyLogger                      ContextKey = "Logger"
	ContextKeyAccessLogs                  ContextKey = "AccessLogs"
	ContextOriRemoteAddr                  ContextKey = "OriRemoteAddr"
	ContextKeyAcceptChan                  ContextKey = "ContextKeyAcceptChan"
	ContextKeyAcceptBuffer                ContextKey = "ContextKeyAcceptBuffer"
	ContextKeyConnectionFd                ContextKey = "ConnectionFd"
	ContextSubProtocol                    ContextKey = "ContextSubProtocol"
	ContextKeyTraceSpanKey                ContextKey = "TraceSpanKey"
	ContextKeyConnHandlerRef              ContextKey = "ConnHandlerRef"
	ContextKeyListenerRef                 ContextKey = "ListenerRef"
	ContextKeyConnectionToClose           ContextKey = "ConnectionToClose"
	ContextKeyUpstreamAddress             ContextKey = "UpstreamAddress"
	ContextKeyPushControlAddress          ContextKey = "PushControlAddress"
)

// GlobalProxyName represents proxy name for metrics
const (
	GlobalProxyName = "global"
)
