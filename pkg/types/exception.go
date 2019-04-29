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

// Header key types
const (
	HeaderStatus        = "x-strike-status"
	HeaderMethod        = "x-strike-method"
	HeaderHost          = "x-strike-host"
	HeaderPath          = "x-strike-path"
	HeaderQueryString   = "x-strike-querystring"
	HeaderStreamID      = "x-strike-streamid"
	HeaderGlobalTimeout = "x-strike-global-timeout"
	HeaderTryTimeout    = "x-strike-try-timeout"
	HeaderException     = "x-strike-exception"
	HeaderStremEnd      = "x-strike-endstream"
	HeaderRPCService    = "x-strike-rpc-service"
	HeaderRPCMethod     = "x-strike-rpc-method"
	HeaderPacketID      = "x-strike-packet-id"
	HeaderUsername      = "x-strike-username"
	HeaderKeepAlive     = "x-strike-keep-alive"
)

// Error messages
const (
	UnSupportedProCode   string = "Protocol Code not supported"
	CodecException       string = "Codec exception occurs"
	DeserializeException string = "Deserial exception occurs"
)

// Error codes
const (
	CodecExceptionCode    = 0
	UnknownCode           = 2
	DeserialExceptionCode = 3
	SuccessCode           = 200
	Unauthorized          = 401
	RouterUnavailableCode = 404
	NoHealthUpstreamCode  = 502
	UpstreamOverFlowCode  = 503
	TimeoutExceptionCode  = 504
	LimitExceededCode     = 509
)
