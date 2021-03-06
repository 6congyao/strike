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

package tcpproxy

import "strike/pkg/network"

type TCPProxy interface {
	network.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
}

// ProxyConfig
type ProxyConfig interface {
	GetRouteFromEntries(connection network.Connection) string
}

// UpstreamCallbacks for upstream's callbacks
type UpstreamCallbacks interface {
	network.ReadFilter
	network.ConnectionEventListener
}

// DownstreamCallbacks for downstream's callbacks
type DownstreamCallbacks interface {
	network.ConnectionEventListener
}

// UpstreamFailureReason used to define Upstream Failure Reason
type UpstreamFailureReason string

// Upstream Failure Reason const
const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)
