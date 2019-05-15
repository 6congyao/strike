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

package v1

import (
	"bytes"
	"strike/pkg/buffer"
	"strike/utils"
	"testing"
)

var singleGetWithBody = `GET /ab HTTP/1.1
Host: 127.0.0.1:8045
Connection: keep-alive
Content-Length: 10
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36
Cache-Control: no-cache
Origin: chrome-extension://fhbjgbiflinjbdggehcddcbncdddomop
Postman-Token: 6cd2751a-42e4-cc70-d8cd-728ce0d48c01
Content-Type: text/plain;charset=UTF-8
Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: zh-CN,zh;q=0.9,en;q=0.8

strike1234`

var singleRequest = `POST /ab HTTP/1.1
Host: 127.0.0.1:8045
Connection: keep-alive
Content-Length: 9
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36
Cache-Control: no-cache
Origin: chrome-extension://fhbjgbiflinjbdggehcddcbncdddomop
Postman-Token: 6cd2751a-42e4-cc70-d8cd-728ce0d48c01
Content-Type: text/plain;charset=UTF-8
Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: zh-CN,zh;q=0.9,en;q=0.8

strike123`

var doubleRequest = `POST /topic1 HTTP/1.1
Host: 127.0.0.1:8045
Connection: keep-alive
Content-Length: 10
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36
Cache-Control: no-cache
Origin: chrome-extension://fhbjgbiflinjbdggehcddcbncdddomop
Postman-Token: 6cd2751a-42e4-cc70-d8cd-728ce0d48c01
Content-Type: text/plain;charset=UTF-8
Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: zh-CN,zh;q=0.9,en;q=0.8

strike1234
POST /topic2 HTTP/1.1
Host: 127.0.0.1:8045
Connection: keep-alive
Content-Length: 10
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36
Cache-Control: no-cache
Origin: chrome-extension://fhbjgbiflinjbdggehcddcbncdddomop
Postman-Token: 6cd2751a-42e4-cc70-d8cd-728ce0d48c01
Content-Type: text/plain;charset=UTF-8
Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: zh-CN,zh;q=0.9,en;q=0.8

strike7890`

var defectiveRequest = `POST /topic1 HTTP/1.1
Host: 127.0.0.1:8045
Connection: keep-alive
Content-Length: 9
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36
Cache-Control: no-cache
Origin: chrome-extension://fhbjgbiflinjbdggehcddcbncdddomop
Postman-Token: 6cd2751a-42e4-cc70-d8cd-728ce0d48c01
Content-`

func TestGetWithBody(t *testing.T) {
	r := buffer.GetIoBuffer(1)
	r.Write(utils.S2b(singleGetWithBody))

	var req *Request
	var err error

	for r.Len() > 0 {
		req, err = decodeHttpReq(r)
		if err != nil {
			//t.Fatal(err)
			r.Drain(r.Len())
			expectedLen := 0
			if r.Len() != expectedLen {
				t.Errorf("Expect data len %v but got %v", expectedLen, r.Len())
			}
			continue
		}
		expectedMethod := []byte("GET")
		if !bytes.Equal(req.Header.method, expectedMethod) {
			t.Errorf("Expect header uri %s but got %s", expectedMethod, string(req.Header.method))
		}
	}

}

func TestSingleDecode(t *testing.T) {
	r := buffer.GetIoBuffer(1)
	r.Write(utils.S2b(singleRequest))

	req, err := decodeHttpReq(r)
	if err != nil {
		t.Fatal(err)
	}

	expectedUri := []byte("/ab")
	if !bytes.Equal(req.Header.RequestURI(), expectedUri) {
		t.Errorf("Expect header uri %s but got %s", expectedUri, string(req.Header.RequestURI()))
	}

	expectedBody := []byte("strike123")
	if !bytes.Equal(req.Body(), expectedBody) {
		t.Errorf("Expect body %s but got %s", expectedBody, string(req.Body()))
	}

	expectedDataLen := 0
	if r.Len() != expectedDataLen {
		t.Errorf("Expect data len %d but got %d", expectedDataLen, r.Len())
	}

	ReleaseRequest(req)
}

func TestDoubleDecode(t *testing.T) {
	r := buffer.GetIoBuffer(1)
	r.Write(utils.S2b(doubleRequest))

	var req *Request
	var err error
	for r.Len() > 0 {
		req, err = decodeHttpReq(r)
		if err != nil {
			t.Fatal(err)
		}
	}

	expectedUri := []byte("/topic2")
	if !bytes.Equal(req.Header.RequestURI(), expectedUri) {
		t.Errorf("Expect header uri %s but got %s", expectedUri, string(req.Header.RequestURI()))
	}

	expectedBody := []byte("strike7890")
	if !bytes.Equal(req.Body(), expectedBody) {
		t.Errorf("Expect body %s but got %s", expectedBody, string(req.Body()))
	}

	expectedDataLen := 0
	if r.Len() != expectedDataLen {
		t.Errorf("Expect data len %d but got %d", expectedDataLen, r.Len())
	}

	ReleaseRequest(req)
}

func TestDefectiveDecode(t *testing.T) {
	r := buffer.GetIoBuffer(1)
	r.Write(utils.S2b(defectiveRequest))

	var req *Request
	var err error
	for r.Len() > 0 {
		req, err = decodeHttpReq(r)
		if err != nil {
			break
		}
	}

	expectedErr := errNeedMore

	if err != expectedErr {
		t.Errorf("Expect err %s but got %s", expectedErr, err)
	}

	ReleaseRequest(req)
}