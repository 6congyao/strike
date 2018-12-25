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

package buffer

import (
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func intRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func intN(n int) int {
	return rand.Intn(n) + 1
}

func TestByteBufferPoolSmallBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := intN(1 << minShift)
		bp := pool.take(size)

		if cap(*bp) != size {
			t.Errorf("Expect get the %d bytes from pool, but got %d", size, cap(*bp))
		}

		// Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesBufferPoolMediumBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := minShift; i < maxShift; i++ {
		size := intRange(1<<uint(i), 1<<uint(i+1))
		bp := pool.take(size)

		if cap(*bp) != 1<<uint(i+1) {
			t.Errorf("Expect get the slab size (%d) from pool, but got %d", 1<<uint(i+1), cap(*bp))
		}

		//Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesBufferPoolLargeBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := 1<<maxShift + intN(i+1)
		bp := pool.take(size)

		if cap(*bp) != size {
			t.Errorf("Expect get the %d bytes from pool, but got %d", size, cap(*bp))
		}

		// Puts the bytes to pool
		pool.give(bp)
	}
}
