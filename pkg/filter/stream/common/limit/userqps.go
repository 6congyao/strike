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
	"errors"
	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

// QPSLimiter limiter
type UserQPSLimiter struct {
	maxAllows          int64
	periodMicros       int64
	maxAllowsPerSecond float64

	tokenBuckets *gocache.Cache
	mutex        sync.Mutex
}

// NewQPSLimiter new
func NewUserQPSLimiter(maxAllows int64, periodMs int64) (*UserQPSLimiter, error) {
	if maxAllows < 0 || periodMs <= 0 {
		return nil, errors.New("maxAllows must not be negtive, and periodMs be positive")
	}

	l := &UserQPSLimiter{
		maxAllows:          maxAllows,
		periodMicros:       periodMs * int64(time.Millisecond),
		tokenBuckets:       gocache.New(DefaultTokenBucketTTL, CleanupTokenBucketInterval),
		maxAllowsPerSecond: float64(maxAllows*1000) / float64(periodMs),
	}
	return l, nil
}

// TryAcquire limit
func (l *UserQPSLimiter) TryAcquire(key interface{}) bool {
	if l.maxAllows <= 0 {
		return false
	}
	tokenBucketTTL := DefaultTokenBucketTTL
	strKey := key.(string)
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if _, found := l.tokenBuckets.Get(strKey); !found {
		l.tokenBuckets.Set(
			strKey,
			rate.NewLimiter(rate.Limit(l.maxAllowsPerSecond), int(l.maxAllows)),
			tokenBucketTTL,
		)
	}
	expiringMap, found := l.tokenBuckets.Get(strKey)
	if !found {
		return false
	}
	return expiringMap.(*rate.Limiter).Allow()
}
