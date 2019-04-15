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

package stream

type BaseStream struct {
	streamListeners []StreamEventListener
}

func (s *BaseStream) AddEventListener(streamCb StreamEventListener) {
	s.streamListeners = append(s.streamListeners, streamCb)
}

func (s *BaseStream) RemoveEventListener(streamCb StreamEventListener) {
	cbIdx := -1

	for i, streamCb := range s.streamListeners {
		if streamCb == streamCb {
			cbIdx = i
			break
		}
	}

	if cbIdx > -1 {
		s.streamListeners = append(s.streamListeners[:cbIdx], s.streamListeners[cbIdx+1:]...)
	}
}

func (s *BaseStream) ResetStream(reason StreamResetReason) {
	defer s.DestroyStream()

	for _, listener := range s.streamListeners {
		listener.OnResetStream(reason)
	}
}

func (s *BaseStream) DestroyStream() {
	for _, listener := range s.streamListeners {
		listener.OnDestroyStream()
	}
}