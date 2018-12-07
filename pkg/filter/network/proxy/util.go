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

import "time"

// Timeout
type Timeout struct {
	GlobalTimeout time.Duration
	TryTimeout    time.Duration
}

type timer struct {
	callback func()
	interval time.Duration
	stopped  bool
	stopChan chan bool
}

func newTimer(callback func(), interval time.Duration) *timer {
	return &timer{
		callback: callback,
		interval: interval,
		stopChan: make(chan bool, 1),
	}
}

func (t *timer) start() {
	go func() {
		select {
		case <-time.After(t.interval):
			t.stopped = true
			t.callback()
		case <-t.stopChan:
			t.stopped = true
		}
	}()
}

func (t *timer) stop() {
	if t.stopped {
		return
	}

	t.stopChan <- true
}
