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
	"reflect"
	"sync/atomic"
)

var globalConnId uint64 = 0

type simpleConn struct {
	id            uint64
	rawc          net.Conn
	filterManager FilterManager
	stopChan      chan struct{}
	closeFlag     int32
}

func NewServerSimpleConn(ctx context.Context, rawc net.Conn, stopChan chan struct{}) Connection {
	sc := &simpleConn{
		id:       atomic.AddUint64(&globalConnId, 1),
		rawc:     rawc,
		stopChan: stopChan,
	}

	sc.filterManager = newFilterManager(sc)
	return sc
}

func (sc *simpleConn) ID() uint64 {
	return sc.id
}

func (sc *simpleConn) Start(ctx context.Context) {

}

func (sc *simpleConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (sc *simpleConn) Close(ccType ConnectionCloseType, eventType ConnectionEvent) error {
	if !atomic.CompareAndSwapInt32(&sc.closeFlag, 0, 1) {
		return nil
	}

	if reflect.ValueOf(sc.RawConn()).IsNil() {
		return nil
	}

	sc.RawConn().(net.Conn).Close()
	return nil
}

func (sc *simpleConn) RemoteAddr() net.Addr {
	return nil
}

func (sc *simpleConn) SetRemoteAddr(address net.Addr) {

}

func (sc *simpleConn) AddConnectionEventListener(cb ConnectionEventListener) {

}

func (sc *simpleConn) GetReadBuffer() []byte {
	return nil
}

func (sc *simpleConn) FilterManager() FilterManager {
	return sc.filterManager
}

func (sc *simpleConn) RawConn() interface{} {
	return sc.rawc
}
