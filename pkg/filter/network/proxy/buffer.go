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
	"context"
	"strike/pkg/buffer"
)

type proxyBufferCtx struct{}

func (ctx proxyBufferCtx) Name() int {
	return buffer.Proxy
}

func (ctx proxyBufferCtx) New() interface{} {
	return new(proxyBuffers)
}

func (ctx proxyBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*proxyBuffers)
	*buf = proxyBuffers{}
}

type proxyBuffers struct {
	stream  downStream
	request upstreamRequest
}

func proxyBuffersByContext(ctx context.Context) *proxyBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(proxyBufferCtx{}, nil).(*proxyBuffers)
}
