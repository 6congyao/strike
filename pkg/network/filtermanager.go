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
	"strike/pkg/buffer"
)

type filterManager struct {
	upstreamFilters   []*activeReadFilter
	downstreamFilters []WriteFilter
	conn              Connection
}

func (fm *filterManager) AddReadFilter(rf ReadFilter) {
	newArf := &activeReadFilter{
		filter:        rf,
		filterManager: fm,
	}

	rf.InitializeReadFilterCallbacks(newArf)
	fm.upstreamFilters = append(fm.upstreamFilters, newArf)
}

func (fm *filterManager) AddWriteFilter(wf WriteFilter) {
	panic("implement me")
}

func (fm *filterManager) ListReadFilter() []ReadFilter {
	panic("implement me")
}

func (fm *filterManager) ListWriteFilters() []WriteFilter {
	panic("implement me")
}

func (fm *filterManager) InitializeReadFilters() bool {
	panic("implement me")
}

func (fm *filterManager) OnRead() {
	panic("implement me")
}

func (fm *filterManager) OnWrite(buffer []buffer.IoBuffer) FilterStatus {
	panic("implement me")
}

// as a ReadFilterCallbacks
type activeReadFilter struct {
	index         int
	filter        ReadFilter
	filterManager *filterManager
	initialized   bool
}

func (arf *activeReadFilter) Connection() Connection {
	return arf.filterManager.conn
}

func (arf *activeReadFilter) ContinueReading() {

}

func newFilterManager(conn Connection) FilterManager {
	return &filterManager{
		conn:              conn,
		upstreamFilters:   make([]*activeReadFilter, 0, 32),
		downstreamFilters: make([]WriteFilter, 0, 32),
	}
}
