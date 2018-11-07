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
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	. "strike/pkg/evio"
	"strike/pkg/protocol/http/v1"
	"syscall"
	"time"
)

type HttpRequest struct {
	conn                 Conn
	proto, method        string
	path, query          string
	body                 []byte
	remoteAddr           string
	bodyLen              int
	interf               string
	callMethod           string
	parameterTypesString string
	parameter            string
	header               v1.RequestHeader

	//profileGetHttpTime   time.Time
	//profileSendAgentTime time.Time
	//profileGetAgentTime  time.Time
	//profileSendHttpTime  time.Time
}

type HttpContext struct {
	is  InputStream
	req *HttpRequest
}

var ParseFormBodyError = errors.New("ParseFormBodyError")

//func (hq *HttpRequest) ParseFormBody() error {
//	kvs := strings.Split(hq.body, "&")
//	if len(kvs) != 4 {
//		return ParseFormBodyError
//	}
//	hq.interf = kvs[0][strings.IndexByte(kvs[0], '=')+1:]
//	hq.callMethod = kvs[1][strings.IndexByte(kvs[1], '=')+1:]
//	hq.parameterTypesString = kvs[2][strings.IndexByte(kvs[2], '=')+1:]
//	hq.parameter = kvs[3][strings.IndexByte(kvs[3], '=')+1:]
//	//logger.Info("ParseFormBody", hq.interf, hq.callMethod, hq.parameterTypesString, hq.parameter)
//	return nil
//}

func getHashCode(param string) int {
	if len(param) >= 1 {
		return int(param[0])
	}
	return 0
}

func main() {
	workerQueues := make(chan *HttpRequest, 1000)
	ServeListenHttp(1, 9090, workerQueues)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for req := range workerQueues {
			fmt.Println("length is: ", req.header.ContentLength())
		}
	}()

	select {
	case s := <-signalChan:
		fmt.Println(fmt.Sprintf("Captured %v. Exiting...", s))
		os.Exit(0)
	}

}

func ServeListenHttp(loops int, port int, workerQueues chan *HttpRequest) error {
	var events Events
	events.NumLoops = loops

	events.Serving = func(srv Server) (action Action) {
		//logger.Info("http server started on port %d (loops: %d)", port, srv.NumLoops)
		fmt.Printf("http server started on port %d (loops: %d) \n", port, srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		c.SetContext(&HttpContext{})

		opts.ReuseInputBuffer = true
		fmt.Printf("http opened: laddr: %v: raddr: %v \n", c.LocalAddr(), c.RemoteAddr())
		return
	}

	// 被动监听的 close 不用管
	events.Closed = func(c Conn, err error) (action Action) {
		fmt.Printf("http closed: laddr: %v: raddr: %v \n", c.LocalAddr(), c.RemoteAddr())
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		//fmt.Printf("Data: laddr: %v: raddr: %v, data \n", c.LocalAddr(), c.RemoteAddr(), string(in))
		//gid := simpleGID()
		//fmt.Println("GID is : ", gid)

		if in == nil {
			return
		}
		httpContext := c.Context().(*HttpContext)

		data := httpContext.is.Begin(in)
		//httpContext.is.End(data)
		//if bytes.Contains(data, []byte("\r\n\r\n")) {
		if true {
			if httpContext.req == nil {
				httpContext.req = &HttpRequest{}
				httpContext.req.conn = c
			}
			// for testing minimal single packet request -> response.
			err := readLimitBody(data, httpContext.req, 40000)
			if err != nil {
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				//httpContext.is.End(data)
				return
			}

			//body := "0"
			//if len(httpContext.req.body) > 0 {
			//	body = string(httpContext.req.body)
			//}
			workerQueues <- httpContext.req
			httpContext.req = nil
			//out = AppendResp(nil, "200 OK", "", body)
			//httpContext.is.End(data)
			return
		}
		// process the pipeline
		for {
			if len(data) > 0 {
				if httpContext.req == nil {
					httpContext.req = &HttpRequest{}
					httpContext.req.conn = c
				}
			} else {
				break
			}
			err := readLimitBody(data, httpContext.req, 60000)

			if err != nil {
				//logger.Warning("bad thing happened\n")
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				break
			}

			//err = httpContext.req.ParseFormBody()
			if err != nil {
				//logger.Warning("parse form body error \n")
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				break
			}

			//index := getHashCode(httpContext.req.parameter) % *config.ConsumerHttpProcessors
			workerQueues <- httpContext.req
			httpContext.req = nil
			//data = leftover
			// handle the request
			//httpContext.req.remoteAddr = c.RemoteAddr().String()
		}
		httpContext.is.End(data)
		return
	}

	// We at least want the single http address.
	//addrs := []string{fmt.Sprintf("tcp://:%d?reuseport=true", port)}
	addrs := []string{fmt.Sprintf("tcp://:%d", port)}
	// Start serving!
	_, err := ServeAndReturn(events, addrs...)
	//GlobalRemoteAgentManager.server = ser
	return err
}

var contentLengthHeaderLower = "content-length:"
var contentLengthHeaderUpper = "CONTENT-LENGTH:"
var contentLengthHeaderLen = 15

func getContentLength(header string) (bool, int) {
	headerLen := len(header)
	if headerLen < contentLengthHeaderLen {
		return false, 0
	}
	i := 0
	for ; i < contentLengthHeaderLen; i++ {
		if header[i] != contentLengthHeaderLower[i] &&
			header[i] != contentLengthHeaderUpper[i] {
			return false, 0
		}
	}
	for ; i < headerLen; i++ {
		if header[i] != ' ' {
			l, _ := strconv.Atoi(header[i:])
			return true, l
		}
	}
	return false, 0
}

// parse req is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
//func parseReq(data []byte, req *HttpRequest) (leftover []byte, err error, ready bool) {
//	readLimitBody(data, req, 60000)
//	sdata := string(data)
//
//	if req.bodyLen > 0 {
//		if len(sdata) < req.bodyLen {
//			return data, nil, false
//		}
//
//		req.body = string(data[0:req.bodyLen])
//		return data[req.bodyLen:], nil, true
//	}
//	var i, s int
//	//var top string
//	var clen int
//	var q = -1
//	// method, path, proto line
//	for ; i < len(sdata); i++ {
//		if sdata[i] == ' ' {
//			req.method = sdata[s:i]
//			for i, s = i+1, i+1; i < len(sdata); i++ {
//				if sdata[i] == '?' && q == -1 {
//					q = i - s
//				} else if sdata[i] == ' ' {
//					if q != -1 {
//						req.path = sdata[s:q]
//						req.query = req.path[q+1 : i]
//					} else {
//						req.path = sdata[s:i]
//					}
//					for i, s = i+1, i+1; i < len(sdata); i++ {
//						if sdata[i] == '\n' && sdata[i-1] == '\r' {
//							req.proto = sdata[s:i]
//							i, s = i+1, i+1
//							break
//						}
//					}
//					break
//				}
//			}
//			break
//		}
//	}
//	if req.proto == "" {
//		return data, nil, false
//	}
//	//top = sdata[:s]
//	for ; i < len(sdata); i++ {
//		if i > 1 && sdata[i] == '\n' && sdata[i-1] == '\r' {
//			line := sdata[s : i-1]
//			//fmt.Print("header line %", line, "%\n")
//			s = i + 1
//			if line == "" {
//				req.bodyLen = clen
//				//req.head = sdata[len(top) : i+1]
//				i++
//				if clen > 0 {
//					if len(sdata[i:]) < clen {
//						return data[i:], nil, false
//					}
//					req.body = sdata[i : i+clen]
//					i += clen
//				}
//				return data[i:], nil, true
//			}
//			ok, cl := getContentLength(line)
//			if ok {
//				clen = cl
//			}
//		}
//	}
//	// not enough data
//	return data, nil, false
//}

func readLimitBody(data []byte, req *HttpRequest, maxRequestBodySize int) error {
	n, err := parseHeader(data, &req.header)
	if err != nil {
		return err
	}
	if req.header.NoBody() {
		return nil
	}

	return continueReadBody(data, n, req, maxRequestBodySize)
}

var ErrBodyTooLarge = errors.New("body size exceeds the given limit")

func continueReadBody(data []byte, offset int, req *HttpRequest, maxRequestBodySize int) error {
	data = data[offset:]
	contentLength := req.header.ContentLength()
	if contentLength > 0 {
		if maxRequestBodySize > 0 && contentLength > maxRequestBodySize {
			return ErrBodyTooLarge
		}
	}

	if contentLength == -2 {
		// identity body has no sense for http requests, since
		// the end of body is determined by connection close.
		// So just ignore request body for requests without
		// 'Content-Length' and 'Transfer-Encoding' headers.
		req.header.SetContentLength(0)
		return nil
	}

	var err error
	req.body, err = readBody(data, contentLength, req.body)
	return err
}

func readBody(data []byte, contentLength int, dst []byte) ([]byte, error) {
	dst = dst[:0]
	if contentLength >= 0 {
		return appendBodyFixedSize(data, dst, contentLength)
	}
	if contentLength == -1 {
		//return readBodyChunked(r, maxBodySize, dst)
	}
	return dst, nil
}

func appendBodyFixedSize(data []byte, dst []byte, n int) ([]byte, error) {
	if n == 0 {
		return dst, nil
	}

	offset := len(dst)
	dstLen := offset + n
	if cap(dst) < dstLen {
		b := make([]byte, round2(dstLen))
		copy(b, dst)
		dst = b
	}
	dst = dst[:dstLen]

	return append(dst[:0], data[:n]...), nil
}

func round2(n int) int {
	if n <= 0 {
		return 0
	}
	n--
	x := uint(0)
	for n > 0 {
		n >>= 1
		x++
	}
	return 1 << x
}

func parseHeader(data []byte, header *v1.RequestHeader) (int, error) {
	header.ResetSkipNormalize()
	n, err := header.Parse(data)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Printf("header len is : %v, body len is : %v \n" , n, len(data) - n)
	return n, err
}

// AppendResp will append a valid http response to the provide bytes.
// The status param should be the code plus text such as "200 OK".
// The head parameter should be a series of lines ending with "\r\n" or empty.
func AppendResp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: nbserver\r\n"...)
	b = append(b, "Connection: keep-alive\r\n"...)
	//b = append(b, "Content-Type: application/json;charset=UTF-8\r\n"...)

	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	if len(head) > 0 {
		b = append(b, head...)
	}
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

func simpleGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
