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

package v1

import (
	"context"
	"strike/pkg/protocol"
	"strike/pkg/stream"
	"strike/pkg/upstream"
	"sync"
)

func init() {
	stream.RegisterConnPool(protocol.MQ, NewConnPool)
}

func NewConnPool(host upstream.Host) stream.ConnectionPool {
	return &connPool{
		host: host,
	}
}

// stream.ConnectionPool
type connPool struct {
	host          upstream.Host
	activeClients sync.Map
}

func (p *connPool) Close() {
	panic("implement me")
}

func (p *connPool) NewStream(ctx context.Context, receiver stream.StreamReceiver, cb stream.PoolEventListener) stream.Cancellable {

	//cb.OnReady(nil, nil)
	return nil
}

func (p *connPool) Protocol() protocol.Protocol {
	return protocol.HTTP1
}
