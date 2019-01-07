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

package qmq

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

// stream.ConnectionPool
type connPool struct {
	activeClient *activeClient
	host         upstream.Host

	mux sync.RWMutex
}

func (cp *connPool) Protocol() protocol.Protocol {
	return protocol.MQ
}

func (cp *connPool) NewStream(ctx context.Context, receiver stream.StreamReceiver, cb stream.PoolEventListener) stream.Cancellable {

	cp.activeClient = cp.getAvailableClient(ctx)
	return nil
}

func (cp *connPool) Close() {
	panic("implement me")
}

func (cp *connPool) getAvailableClient(ctx context.Context) *activeClient {
	ac := &activeClient{
		pool: cp,
	}

	return ac
}

// NewConnPool
func NewConnPool(host upstream.Host) stream.ConnectionPool {
	return &connPool{
		host: host,
	}
}

// stream.CodecClientCallbacks
// stream.StreamConnectionEventListener
// network.ConnectionEventListener
type activeClient struct {
	pool               *connPool
	closeWithActiveReq bool
	totalStream        uint64
}
