// +build !darwin,!netbsd,!freebsd,!openbsd,!dragonfly,!linux

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

package evio

import (
	"errors"
	"net"
	"os"
)

func (ln *listener) close() {
	if ln.ln != nil {
		ln.ln.Close()
	}
	if ln.pconn != nil {
		ln.pconn.Close()
	}
	if ln.network == "unix" {
		os.RemoveAll(ln.addr)
	}
}

func (ln *listener) system() error {
	return nil
}

func serve(events Events, listeners []*listener) error {
	return stdserve(events, listeners)
}

func reuseportListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return nil, errors.New("reuseport is not available")
}

func reuseportListen(proto, addr string) (l net.Listener, err error) {
	return nil, errors.New("reuseport is not available")
}
