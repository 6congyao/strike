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

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type AddReq struct {
	A, B int
}

func main() {
	codec, err := Dial("tcp", "0.0.0.0:2045")
	if err != nil {
		fmt.Println("dial error:", err)
	}

	for i := 0; i < 10; i++ {
		err := codec.Encode(&AddReq{
			i, i,
		})
		if err != nil {
			fmt.Println("send error:", err)
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func Dial(network, address string) (*json.Encoder, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	codec := json.NewEncoder(conn)
	if err != nil {
		return nil, err
	}
	return codec, nil
}
