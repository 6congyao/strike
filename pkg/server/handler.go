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

package server

import (
	"context"
	"net"
	"strike/pkg/network"
	"sync/atomic"
)

type connHandler struct {
	numConnections int64
}

// NewHandler
// create network.ConnectionHandler's implement connHandler
func NewHandler() network.ConnectionHandler {
	ch := &connHandler{
		numConnections: 0,
	}

	return ch
}

// ConnectionHandler
// todo
func (ch *connHandler) NumConnections() uint64 {
	return uint64(atomic.LoadInt64(&ch.numConnections))
}

func (ch *connHandler) StartListener(lctx context.Context, listenerTag uint64) {
	return
}

func (ch *connHandler) StartListeners(lctx context.Context) {
	return
}

func (ch *connHandler) FindListenerByAddress(addr net.Addr) network.Listener {
	return nil
}

func (ch *connHandler) FindListenerByName(name string) network.Listener {
	return nil
}

func (ch *connHandler) RemoveListeners(name string) {
	return
}

func (ch *connHandler) StopListener(lctx context.Context, name string, close bool) error {
	return nil
}

func (ch *connHandler) StopListeners(lctx context.Context, close bool) error {
	return nil
}

func (ch *connHandler) ListListenersFD(lctx context.Context) []uintptr {
	return nil
}

func (ch *connHandler) StopConnection() {
	return
}
