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

// NewConnPool
func NewConnPool(host upstream.Host) stream.ConnectionPool {
	return &connPool{
		host: host,
	}
}

func (cp *connPool) Protocol() protocol.Protocol {
	return protocol.MQ
}

func (cp *connPool) NewStream(ctx context.Context, receiver stream.StreamReceiver, cb stream.PoolEventListener) stream.Cancellable {
	//cp.mux.Lock()
	if cp.activeClient == nil {
		cp.activeClient = newActiveClient(ctx, cp)
	}
	//cp.mux.Unlock()

	ac := cp.activeClient
	if ac == nil {
		cb.OnFailure(stream.ConnectionFailure, nil)
		return nil
	}

	streamEncoder := ac.codecClient.NewStream(ctx, receiver)
	cb.OnReady(streamEncoder, cp.host)
	return nil
}

func (cp *connPool) Close() {
}

func (cp *connPool) createCodecClient(context context.Context) stream.CodecClient {
	return stream.NewCodecClient(context, protocol.MQ, nil, nil)
}

// stream.CodecClientCallbacks
// stream.StreamConnectionEventListener
// network.ConnectionEventListener
type activeClient struct {
	pool               *connPool
	codecClient        stream.CodecClient
	closeWithActiveReq bool
	totalStream        uint64
}

func newActiveClient(ctx context.Context, pool *connPool) *activeClient {
	ac := &activeClient{
		pool: pool,
	}

	codecClient := pool.createCodecClient(ctx)
	ac.codecClient = codecClient

	return ac
}
