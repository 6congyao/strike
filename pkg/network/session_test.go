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

package network

import (
	"fmt"
	"testing"
)

type MyEventListener struct{}

func (el *MyEventListener) OnEvent(event ConnectionEvent) {}

type MyEmitter struct{}

func (me *MyEmitter) Emit(topic string, args ...interface{}) error {
	return nil
}

func TestAddConnectionEventListener(t *testing.T) {
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("AddConnectionEventListener(%d)", i)
		t.Run(name, func(t *testing.T) {
			testAddConnectionEventListener(i, t)
		})
	}
}

func testAddConnectionEventListener(n int, t *testing.T) {
	s := Session{}

	for i := 0; i < n; i++ {
		el0 := &MyEventListener{}
		s.AddConnectionEventListener(el0)
	}

	if len(s.connCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddConnectionEventListener(el0)", n, len(s.connCallbacks))
	}
}

func TestAddEmitter(t *testing.T) {
	s := &Session{}
	for i := 0; i < 10; i++ {
		e := &MyEmitter{}
		s.AddEmitter(e)
	}

	if len(s.emitters) != 10 {
		t.Errorf("Expect %d, but got %d after AddEmitter", 10, len(s.emitters))
	}
}