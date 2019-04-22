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

import "testing"

func TestIoBufferPoolWithCount(t *testing.T) {
	buf := GetIoBuffer(0)
	bytes := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	buf.Write(bytes)
	if buf.Len() != len(bytes) {
		t.Error("iobuffer len not match write bytes' size")
	}
	// Add a count, need put twice to free buffer
	buf.Count(1)
	PutIoBuffer(buf)
	if buf.Len() != len(bytes) {
		t.Error("iobuffer expected put ignore")
	}
	PutIoBuffer(buf)
	if buf.Len() != 0 {
		t.Error("iobuffer expected put success")
	}
}

func TestIoBufferPooPutduplicate(t *testing.T) {
	buf := GetIoBuffer(0)
	err := PutIoBuffer(buf)
	if err != nil {
		t.Errorf("iobuffer put error:%v", err)
	}
	err = PutIoBuffer(buf)
	if err == nil {
		t.Errorf("iobuffer should be error: Put IoBuffer duplicate")
	}
}
