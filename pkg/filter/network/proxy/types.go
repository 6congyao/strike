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

package proxy

import (
	"strike/pkg/network"
)

// Proxy
type Proxy interface {
	network.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
}

// UpstreamCallbacks
// callback invoked when upstream event happened
type UpstreamCallbacks interface {
	network.ReadFilter
	network.ConnectionEventListener
}

// DownstreamCallbacks
// callback invoked when downstream event happened
type DownstreamCallbacks interface {
	network.ConnectionEventListener
}

// UpstreamFailureReason
type UpstreamFailureReason string

// Group pf some Upstream Failure Reason
const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)

// UpstreamResetType
type UpstreamResetType string

// Group of Upstream Reset Type
const (
	UpstreamReset         UpstreamResetType = "UpstreamReset"
	UpstreamGlobalTimeout UpstreamResetType = "UpstreamGlobalTimeout"
	UpstreamPerTryTimeout UpstreamResetType = "UpstreamPerTryTimeout"
)
