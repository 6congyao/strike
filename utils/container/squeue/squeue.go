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

package squeue

import (
	"strike/utils/container/slist"
	"sync/atomic"
)

type Queue struct {
	limit  int              // Limit for queue size.
	list   *slist.List      // Underlying list structure for data maintaining.
	closed int32            // Whether queue is closed.
	events chan struct{}    // Events for data writing.
	C      chan interface{} // Underlying channel for data reading.
}

const (
	// Size for queue buffer.
	DEFAULT_QUEUE_SIZE = 512
	// Max batch size per-fetching from list.
	DEFAULT_MAX_BATCH_SIZE = 10
)

// New returns an empty queue object.
// Optional parameter <limit> is used to limit the size of the queue, which is unlimited in default.
// When <limit> is given, the queue will be static and high performance which is comparable with stdlib channel.
func New(limit ...int) *Queue {
	q := &Queue{}
	if len(limit) > 0 {
		q.limit = limit[0]
		q.C = make(chan interface{}, limit[0])
	} else {
		q.list = slist.New()
		q.events = make(chan struct{}, DEFAULT_QUEUE_SIZE+DEFAULT_MAX_BATCH_SIZE)
		q.C = make(chan interface{}, DEFAULT_QUEUE_SIZE)
		go q.startAsyncLoop()
	}
	return q
}

// startAsyncLoop starts an asynchronous goroutine,
// which handles the data synchronization from list <q.list> to channel <q.C>.
func (q *Queue) startAsyncLoop() {
	defer func() {
		//if q.closed.Val() {
		if atomic.LoadInt32(&q.closed) > 0 {
			_ = recover()
		}
	}()
	//for !q.closed.Val() {
	for atomic.LoadInt32(&q.closed) < 1 {
		<-q.events
		for atomic.LoadInt32(&q.closed) < 1 {
			if length := q.list.Len(); length > 0 {
				if length > DEFAULT_MAX_BATCH_SIZE {
					length = DEFAULT_MAX_BATCH_SIZE
				}
				for _, v := range q.list.PopFronts(length) {
					// When q.C is closed, it will panic here, especially q.C is being blocked for writing.
					// If any error occurs here, it will be caught by recover and be ignored.
					q.C <- v
				}
			} else {
				break
			}
		}
		// Clear q.events to remain just one event to do the next synchronization check.
		for i := 0; i < len(q.events)-1; i++ {
			<-q.events
		}
	}
	// It should be here to close q.C.
	// It's the sender's responsibility to close channel when it should be closed.
	close(q.C)
}

// Push pushes the data <v> into the queue.
// Note that it would panics if Push is called after the queue is closed.
func (q *Queue) Push(v interface{}) {
	if q.limit > 0 {
		q.C <- v
	} else {
		q.list.PushBack(v)
		if len(q.events) < DEFAULT_QUEUE_SIZE {
			q.events <- struct{}{}
		}
	}
}

// Pop pops an item from the queue in FIFO way.
// Note that it would return nil immediately if Pop is called after the queue is closed.
func (q *Queue) Pop() interface{} {
	return <-q.C
}

// Close closes the queue.
// Notice: It would notify all goroutines return immediately,
// which are being blocked reading using Pop method.
func (q *Queue) Close() {
	atomic.SwapInt32(&q.closed, 1)
	if q.events != nil {
		close(q.events)
	}
	for i := 0; i < DEFAULT_MAX_BATCH_SIZE; i++ {
		q.Pop()
	}
}

// Len returns the length of the queue.
func (q *Queue) Len() (length int) {
	if q.list != nil {
		length += q.list.Len()
	}
	length += len(q.C)
	return
}

// Size is alias of Len.
func (q *Queue) Size() int {
	return q.Len()
}
