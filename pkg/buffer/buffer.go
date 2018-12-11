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

package buffer

import (
	"context"
	"runtime"
	"strike/pkg/types"
	"sync"
	"sync/atomic"
)

const maxPoolSize = 1

// Register the bufferpool's name
const (
	Protocol = iota
	MHTTP2
	Stream
	Proxy
	Bytes
	End
)

var nullBufferCtx [End]interface{}

var (
	index                   uint32
	poolSize                = runtime.NumCPU()
	bufferPoolContainers    [maxPoolSize]bufferPoolContainer
	bufferPoolCtxContainers [maxPoolSize]bufferPoolCtxContainer
)

type bufferPoolContainer struct {
	pool [End]bufferPool
}

// bufferPool is buffer pool
type bufferPool struct {
	ctx BufferPoolCtx
	sync.Pool
}

type bufferPoolCtxContainer struct {
	sync.Pool
}

// Take returns a buffer from buffer pool
func (p *bufferPool) take() (value interface{}) {
	value = p.Get()
	if value == nil {
		value = p.ctx.New()
	}
	return
}

// Give returns a buffer to buffer pool
func (p *bufferPool) give(value interface{}) {
	p.ctx.Reset(value)
	p.Put(value)
}

// PoolCtx is buffer pool's context
type PoolCtx struct {
	*bufferPoolContainer
	value    [End]interface{}
	transmit [End]interface{}
}

// NewBufferPoolContext returns a context with PoolCtx
func NewBufferPoolContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, types.ContextKeyBufferPoolCtx, newBufferPoolCtx())
}

// TransmitBufferPoolContext copy a context
func TransmitBufferPoolContext(dst context.Context, src context.Context) {
	sctx := PoolContext(src)
	if sctx.value == nullBufferCtx {
		return
	}
	dctx := PoolContext(dst)
	dctx.transmit = sctx.value
	sctx.value = nullBufferCtx
}

func bufferPoolIndex() int {
	i := atomic.AddUint32(&index, 1)
	i = i % uint32(maxPoolSize) % uint32(poolSize)
	return int(i)
}

// newBufferPoolCtx returns PoolCtx
func newBufferPoolCtx() (ctx *PoolCtx) {
	i := bufferPoolIndex()
	value := bufferPoolCtxContainers[i].Get()
	if value == nil {
		ctx = &PoolCtx{
			bufferPoolContainer: &bufferPoolContainers[i],
		}
	} else {
		ctx = value.(*PoolCtx)
	}
	return
}

func initBufferPoolCtx(poolCtx BufferPoolCtx) {
	for i := 0; i < maxPoolSize; i++ {
		pool := &bufferPoolContainers[i].pool[poolCtx.Name()]
		pool.ctx = poolCtx
	}
}

// GetPool returns buffer pool
func (ctx *PoolCtx) getPool(poolCtx BufferPoolCtx) *bufferPool {
	pool := &ctx.pool[poolCtx.Name()]
	if pool.ctx == nil {
		initBufferPoolCtx(poolCtx)
	}
	return pool
}

// Find returns buffer from PoolCtx
func (ctx *PoolCtx) Find(poolCtx BufferPoolCtx, i interface{}) interface{} {
	if ctx.value[poolCtx.Name()] != nil {
		return ctx.value[poolCtx.Name()]
	}
	return ctx.Take(poolCtx)
}

// Take returns buffer from buffer pools
func (ctx *PoolCtx) Take(poolCtx BufferPoolCtx) (value interface{}) {
	pool := ctx.getPool(poolCtx)
	value = pool.take()
	ctx.value[poolCtx.Name()] = value
	return
}

// Give returns buffer to buffer pools
func (ctx *PoolCtx) Give() {
	for i := 0; i < len(ctx.value); i++ {
		value := ctx.value[i]
		if value != nil {
			ctx.pool[i].give(value)
		}
		value = ctx.transmit[i]
		if value != nil {
			ctx.pool[i].give(value)
		}
	}
	ctx.transmit = nullBufferCtx
	ctx.value = nullBufferCtx

	i := bufferPoolIndex()

	// Give PoolCtx to Pool
	bufferPoolCtxContainers[i].Put(ctx)
}

func bufferCtxCopy(ctx *PoolCtx) *PoolCtx {
	newctx := newBufferPoolCtx()
	if ctx != nil {
		newctx.value = ctx.value
		ctx.value = nullBufferCtx
	}
	return newctx
}

// PoolContext returns PoolCtx by context
func PoolContext(context context.Context) *PoolCtx {
	if context != nil && context.Value(types.ContextKeyBufferPoolCtx) != nil {
		return context.Value(types.ContextKeyBufferPoolCtx).(*PoolCtx)
	}
	return newBufferPoolCtx()
}
