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

import ()

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
	var readFilters []ReadFilter

	for _, uf := range fm.upstreamFilters {
		readFilters = append(readFilters, uf.filter)
	}

	return readFilters
}

func (fm *filterManager) ListWriteFilters() []WriteFilter {
	return fm.downstreamFilters
}

func (fm *filterManager) InitializeReadFilters() bool {
	if len(fm.upstreamFilters) == 0 {
		return false
	}

	fm.onContinueReading(nil)
	return true
}

func (fm *filterManager) onContinueReading(filter *activeReadFilter) {
	var index int
	var uf *activeReadFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(fm.upstreamFilters); index++ {
		uf = fm.upstreamFilters[index]
		uf.index = index

		if !uf.initialized {
			uf.initialized = true

			status := uf.filter.OnNewConnection()

			if status == Stop {
				return
			}
		}

		buf := fm.conn.GetReadBuffer()

		if buf != nil && buf.Len() > 0 {
			status := uf.filter.OnData(buf)

			if status == Stop {
				return
			}
		}
	}
}

func (fm *filterManager) OnRead() {
	fm.onContinueReading(nil)
}

func (fm *filterManager) OnWrite(buffer []byte) FilterStatus {
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
