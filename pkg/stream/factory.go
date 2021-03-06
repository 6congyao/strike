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

package stream

import (
	"context"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/upstream"
)

var (
	streamFactories   map[protocol.Protocol]ProtocolStreamFactory
	ConnPoolFactories map[protocol.Protocol]connNewPool
)

type connNewPool func(host upstream.Host) ConnectionPool

func init() {
	streamFactories = make(map[protocol.Protocol]ProtocolStreamFactory)
	ConnPoolFactories = make(map[protocol.Protocol]connNewPool)
}

func Register(prot protocol.Protocol, factory ProtocolStreamFactory) {
	streamFactories[prot] = factory
}

func CreateServerStreamConnection(context context.Context, prot protocol.Protocol, connection network.Connection,
	callbacks ServerStreamConnectionEventListener) ServerStreamConnection {

	if ssc, ok := streamFactories[prot]; ok {
		return ssc.CreateServerStreamConnection(context, connection, callbacks)
	}

	return nil
}

func RegisterConnPool(protocol protocol.Protocol, factory connNewPool) {
	ConnPoolFactories[protocol] = factory
}
