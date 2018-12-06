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

package protocol

import (
	"strconv"
	"sync/atomic"
)

var defaultGenerator IDGenerator

// IDGenerator utility to generate auto-increment ids
type IDGenerator struct {
	counter uint32
}

// Get get id
func (g *IDGenerator) Get() uint32 {
	return atomic.AddUint32(&g.counter, 1)
}

// Get get id in string format
func (g *IDGenerator) GetString() string {
	n := atomic.AddUint32(&g.counter, 1)
	return strconv.FormatUint(uint64(n), 10)
}

// GenerateID get id by default global generator
func GenerateID() uint32 {
	return defaultGenerator.Get()
}

// GenerateIDString get id string by default global generator
func GenerateIDString() string {
	return defaultGenerator.GetString()
}

// StreamIDConv convert streamID from uint32 to string
func StreamIDConv(streamID uint32) string {
	return strconv.FormatUint(uint64(streamID), 10)
}

// RequestIDConv convert streamID from string to uint32
func RequestIDConv(streamID string) uint32 {
	reqID, _ := strconv.ParseUint(streamID, 10, 32)
	return uint32(reqID)
}
