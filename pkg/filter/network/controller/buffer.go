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

package controller

import (
	"context"
	"strike/pkg/buffer"
)

func init() {
	buffer.RegisterBuffer(&ins)
}

var ins = controllerBufferCtx{}

type controllerBufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx controllerBufferCtx) New() interface{} {
	return new(controllerBuffers)
}

func (ctx controllerBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*controllerBuffers)
	*buf = controllerBuffers{}
}

type controllerBuffers struct {
	stream sourceStream
}

func controllerBuffersByContext(ctx context.Context) *controllerBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*controllerBuffers)
}
