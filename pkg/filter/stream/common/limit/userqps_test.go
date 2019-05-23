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

package limit

import (
	"testing"
	"time"
)

func TestUserQpsLimiter_TryAcquire(t *testing.T) {
	limiter, err := NewUserQPSLimiter(0, 1000)
	if err != nil {
		t.Errorf("%v", err)
	}
	key := "test"
	res := limiter.TryAcquire(key)
	if res {
		t.Errorf("false")
	} else {
		t.Log("ok")
	}
}

func TestUserQpsLimiter_TryAcquire1(t *testing.T) {
	limiter, err := NewUserQPSLimiter(1, 1000)
	if err != nil {
		t.Errorf("%v", err)
	}
	key := "test"
	res := limiter.TryAcquire(key)
	if res {
		t.Log("ok")
	} else {
		t.Errorf("false")
	}

	key2 := "test2"
	res = limiter.TryAcquire(key2)
	if res {
		t.Log("ok")
	} else {
		t.Errorf("false")
	}

	res = limiter.TryAcquire(key)
	if res {
		t.Errorf("false")
	} else {
		t.Log("ok")
	}

	time.Sleep(time.Second)
	res = limiter.TryAcquire(key)
	if res {
		t.Log("ok")
	} else {
		t.Errorf("false")
	}
}
