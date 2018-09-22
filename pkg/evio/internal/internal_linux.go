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

package internal

import (
	"log"
	"syscall"
)

// Poll ...
type Poll struct {
	fd     int    // epoll fd
	wfd    int    // wake fd
	wfdBuf []byte // wfd buffer to read packet
	notes  noteQueue
}

// OpenPoll ...
func OpenPoll() *Poll {
	l := new(Poll)
	p, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Println(err)
	}
	l.fd = p
	r0, _, e0 := syscall.Syscall(syscall.SYS_EVENTFD, 0, 0, 0)
	if e0 != 0 {
		syscall.Close(p)
		log.Println(err)
	}
	l.wfd = int(r0)
	l.wfdBuf = make([]byte, 0xFF)
	l.AddRead(l.wfd)
	return l
}

// Close ...
func (p *Poll) Close() error {
	if err := syscall.Close(p.wfd); err != nil {
		return err
	}
	return syscall.Close(p.fd)
}

// Trigger ...
func (p *Poll) Trigger(note interface{}) error {
	p.notes.Add(note)
	_, err := syscall.Write(p.wfd, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	return err
}

// Wait ...
func (p *Poll) Wait(iter func(fd int, note interface{}, event int) error) error {
	events := make([]syscall.EpollEvent, 64)
	for {
		n, err := syscall.EpollWait(p.fd, events, -1)
		if err != nil && err != syscall.EINTR {
			return err
		}
		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note, 0)
		}); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if fd := int(events[i].Fd); fd != p.wfd {
				e := PollEvent_None
				if events[i].Events&syscall.EPOLLIN > 0 {
					e |= PollEvent_Read
				}
				if events[i].Events&syscall.EPOLLOUT > 0 {
					e |= PollEvent_Write
				}
				if err := iter(fd, nil, e); err != nil {
					return err
				}
			} else {
				syscall.Read(p.wfd, p.wfdBuf)
			}
		}
	}
}

// AddReadWrite ...
func (p *Poll) AddReadWrite(fd int) {
	if err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd,
		&syscall.EpollEvent{Fd: int32(fd),
			Events: syscall.EPOLLIN | syscall.EPOLLOUT | syscall.EPOLLRDHUP | syscall.EPOLLHUP,
		},
	); err != nil {
		p.ModReadWrite(fd)
	}
}

// AddRead ...
func (p *Poll) AddRead(fd int) {
	if err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd,
		&syscall.EpollEvent{Fd: int32(fd),
			Events: syscall.EPOLLIN | syscall.EPOLLRDHUP | syscall.EPOLLHUP,
		},
	); err != nil {
		log.Println(err)
	}
}

// ModRead ...
func (p *Poll) ModRead(fd int) {
	if err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_MOD, fd,
		&syscall.EpollEvent{Fd: int32(fd),
			Events: syscall.EPOLLIN | syscall.EPOLLRDHUP | syscall.EPOLLHUP,
		},
	); err != nil {
		log.Println(err)
	}
}

// ModReadWrite ...
func (p *Poll) ModReadWrite(fd int) {
	if err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_MOD, fd,
		&syscall.EpollEvent{Fd: int32(fd),
			Events: syscall.EPOLLIN | syscall.EPOLLOUT | syscall.EPOLLRDHUP | syscall.EPOLLHUP,
		},
	); err != nil {
		log.Println(err)
	}
}

// ModDetach ...
func (p *Poll) ModDetach(fd int) {
	if err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_DEL, fd,
		&syscall.EpollEvent{Fd: int32(fd),
			Events: syscall.EPOLLIN | syscall.EPOLLOUT | syscall.EPOLLRDHUP | syscall.EPOLLHUP,
		},
	); err != nil {
		log.Println(err)
	}
}
