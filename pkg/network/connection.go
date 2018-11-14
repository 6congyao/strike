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

package network

import (
	"context"
	"net"
	"strike/pkg/buffer"
	"sync/atomic"
)

var globalSessionId uint64 = 1

type simpleConn struct {
	id            uint64
	rawConnection net.Conn
	filterManager FilterManager
	stopChan      chan struct{}
	closeFlag     int32
}

func NewServerSimpleConn(ctx context.Context, rawc net.Conn, stopChan chan struct{}) Connection {
	sc := &simpleConn{
		id:            atomic.AddUint64(&globalSessionId, 1),
		rawConnection: rawc,
		stopChan:      stopChan,
	}

	sc.filterManager = newFilterManager(sc)
	return sc
}

func (sc *simpleConn) ID() uint64 {
	return sc.id
}

func (sc *simpleConn) Start(lctx context.Context) {

}

func (sc *simpleConn) Write(buf ...buffer.IoBuffer) error {
	return nil
}

func (sc *simpleConn) Close(ccType ConnectionCloseType, eventType ConnectionEvent) error {
	if atomic.CompareAndSwapInt32(&sc.closeFlag, 0, 1) {
		close(sc.stopChan)
	}
	return nil
}

func (sc *simpleConn) LocalAddr() net.Addr {
	return nil
}

func (sc *simpleConn) RemoteAddr() net.Addr {
	return nil
}

func (sc *simpleConn) SetRemoteAddr(address net.Addr) {

}

func (sc *simpleConn) AddConnectionEventListener(cb ConnectionEventListener) {

}

func (sc *simpleConn) AddBytesReadListener(cb func(bytesRead uint64)) {

}

func (sc *simpleConn) AddBytesSentListener(cb func(bytesSent uint64)) {

}

func (sc *simpleConn) NextProtocol() string {
	return ""
}

func (sc *simpleConn) SetNoDelay(enable bool) {

}

func (sc *simpleConn) SetReadDisable(disable bool) {

}

func (sc *simpleConn) ReadEnabled() bool {
	return false
}

func (sc *simpleConn) TLS() net.Conn {
	return nil
}

func (sc *simpleConn) SetBufferLimit(limit uint32) {

}

func (sc *simpleConn) BufferLimit() uint32 {
	return 0
}

func (sc *simpleConn) SetLocalAddress(localAddress net.Addr, restored bool) {

}

func (sc *simpleConn) SetStats(stats *ConnectionStats) {

}

func (sc *simpleConn) LocalAddressRestored() bool {
	return false
}

func (sc *simpleConn) GetWriteBuffer() []buffer.IoBuffer {
	return nil
}

func (sc *simpleConn) GetReadBuffer() buffer.IoBuffer {
	return nil
}

func (sc *simpleConn) FilterManager() FilterManager {
	return sc.filterManager
}

func (sc *simpleConn) RawConn() net.Conn {
	return sc.rawConnection
}
