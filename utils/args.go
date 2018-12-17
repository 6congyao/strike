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

package utils

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

// AcquireArgs returns an empty Args object from the pool.
//
// The returned Args may be returned to the pool with ReleaseArgs
// when no longer needed. This allows reducing GC load.
func AcquireArgs() *Args {
	return argsPool.Get().(*Args)
}

// ReleaseArgs returns the object acquired via AquireArgs to the pool.
//
// Do not access the released Args object, otherwise data races may occur.
func ReleaseArgs(a *Args) {
	a.Reset()
	argsPool.Put(a)
}

var argsPool = &sync.Pool{
	New: func() interface{} {
		return &Args{}
	},
}

// Args represents query arguments.
//
// It is forbidden copying Args instances. Create new instances instead
// and use CopyTo().
//
// Args instance MUST NOT be used from concurrently running goroutines.
type Args struct {
	noCopy NoCopy

	args []ArgsKV
	buf  []byte
}

type ArgsKV struct {
	Key   []byte
	Value []byte
}

// Reset clears query args.
func (a *Args) Reset() {
	a.args = a.args[:0]
}

// CopyTo copies all args to dst.
func (a *Args) CopyTo(dst *Args) {
	dst.Reset()
	dst.args = CopyArgs(dst.args, a.args)
}

// VisitAll calls f for each existing arg.
//
// f must not retain references to key and value after returning.
// Make key and/or value copies if you need storing them after returning.
func (a *Args) VisitAll(f func(key, value []byte)) {
	VisitArgs(a.args, f)
}

// Len returns the number of query args.
func (a *Args) Len() int {
	return len(a.args)
}

// Parse parses the given string containing query args.
func (a *Args) Parse(s string) {
	a.buf = append(a.buf[:0], s...)
	a.ParseBytes(a.buf)
}

// ParseBytes parses the given b containing query args.
func (a *Args) ParseBytes(b []byte) {
	a.Reset()

	var s argsScanner
	s.b = b

	var kv *ArgsKV
	a.args, kv = AllocArg(a.args)
	for s.next(kv) {
		if len(kv.Key) > 0 || len(kv.Value) > 0 {
			a.args, kv = AllocArg(a.args)
		}
	}
	a.args = ReleaseArg(a.args)
}

// String returns string representation of query args.
func (a *Args) String() string {
	return string(a.QueryString())
}

// QueryString returns query string for the args.
//
// The returned value is valid until the next call to Args methods.
func (a *Args) QueryString() []byte {
	a.buf = a.AppendBytes(a.buf[:0])
	return a.buf
}

// AppendBytes appends query string to dst and returns the extended dst.
func (a *Args) AppendBytes(dst []byte) []byte {
	for i, n := 0, len(a.args); i < n; i++ {
		kv := &a.args[i]
		dst = AppendQuotedArg(dst, kv.Key)
		if len(kv.Value) > 0 {
			dst = append(dst, '=')
			dst = AppendQuotedArg(dst, kv.Value)
		}
		if i+1 < n {
			dst = append(dst, '&')
		}
	}
	return dst
}

// WriteTo writes query string to w.
//
// WriteTo implements io.WriterTo interface.
func (a *Args) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(a.QueryString())
	return int64(n), err
}

// Del deletes argument with the given key from query args.
func (a *Args) Del(key string) {
	a.args = DelAllArgs(a.args, key)
}

// DelBytes deletes argument with the given key from query args.
func (a *Args) DelBytes(key []byte) {
	a.args = DelAllArgs(a.args, B2s(key))
}

// Add adds 'key=value' argument.
//
// Multiple values for the same key may be added.
func (a *Args) Add(key, value string) {
	a.args = AppendArg(a.args, key, value)
}

// AddBytesK adds 'key=value' argument.
//
// Multiple values for the same key may be added.
func (a *Args) AddBytesK(key []byte, value string) {
	a.args = AppendArg(a.args, B2s(key), value)
}

// AddBytesV adds 'key=value' argument.
//
// Multiple values for the same key may be added.
func (a *Args) AddBytesV(key string, value []byte) {
	a.args = AppendArg(a.args, key, B2s(value))
}

// AddBytesKV adds 'key=value' argument.
//
// Multiple values for the same key may be added.
func (a *Args) AddBytesKV(key, value []byte) {
	a.args = AppendArg(a.args, B2s(key), B2s(value))
}

// Set sets 'key=value' argument.
func (a *Args) Set(key, value string) {
	a.args = SetArg(a.args, key, value)
}

// SetBytesK sets 'key=value' argument.
func (a *Args) SetBytesK(key []byte, value string) {
	a.args = SetArg(a.args, B2s(key), value)
}

// SetBytesV sets 'key=value' argument.
func (a *Args) SetBytesV(key string, value []byte) {
	a.args = SetArg(a.args, key, B2s(value))
}

// SetBytesKV sets 'key=value' argument.
func (a *Args) SetBytesKV(key, value []byte) {
	a.args = SetArgBytes(a.args, key, value)
}

// Peek returns query arg value for the given key.
//
// Returned value is valid until the next Args call.
func (a *Args) Peek(key string) []byte {
	return PeekArgStr(a.args, key)
}

// PeekBytes returns query arg value for the given key.
//
// Returned value is valid until the next Args call.
func (a *Args) PeekBytes(key []byte) []byte {
	return PeekArgBytes(a.args, key)
}

// PeekMulti returns all the arg values for the given key.
func (a *Args) PeekMulti(key string) [][]byte {
	var values [][]byte
	a.VisitAll(func(k, v []byte) {
		if string(k) == key {
			values = append(values, v)
		}
	})
	return values
}

// PeekMultiBytes returns all the arg values for the given key.
func (a *Args) PeekMultiBytes(key []byte) [][]byte {
	return a.PeekMulti(B2s(key))
}

// Has returns true if the given key exists in Args.
func (a *Args) Has(key string) bool {
	return hasArg(a.args, key)
}

// HasBytes returns true if the given key exists in Args.
func (a *Args) HasBytes(key []byte) bool {
	return hasArg(a.args, B2s(key))
}

// ErrNoArgValue is returned when Args value with the given key is missing.
var ErrNoArgValue = errors.New("no Args value for the given key")

// GetUint returns uint value for the given key.
func (a *Args) GetUint(key string) (int, error) {
	value := a.Peek(key)
	if len(value) == 0 {
		return -1, ErrNoArgValue
	}
	return ParseUint(value)
}

// GetUintOrZero returns uint value for the given key.
//
// Zero (0) is returned on error.
func (a *Args) GetUintOrZero(key string) int {
	n, err := a.GetUint(key)
	if err != nil {
		n = 0
	}
	return n
}

// GetUfloat returns ufloat value for the given key.
func (a *Args) GetUfloat(key string) (float64, error) {
	value := a.Peek(key)
	if len(value) == 0 {
		return -1, ErrNoArgValue
	}
	return ParseUfloat(value)
}

// GetUfloatOrZero returns ufloat value for the given key.
//
// Zero (0) is returned on error.
func (a *Args) GetUfloatOrZero(key string) float64 {
	f, err := a.GetUfloat(key)
	if err != nil {
		f = 0
	}
	return f
}

// GetBool returns boolean value for the given key.
//
// true is returned for '1', 'y' and 'yes' values,
// otherwise false is returned.
func (a *Args) GetBool(key string) bool {
	switch string(a.Peek(key)) {
	case "1", "y", "yes":
		return true
	default:
		return false
	}
}

func PeekArgStr(h []ArgsKV, k string) []byte {
	for i, n := 0, len(h); i < n; i++ {
		kv := &h[i]
		if string(kv.Key) == k {
			return kv.Value
		}
	}
	return nil
}

func hasArg(h []ArgsKV, key string) bool {
	for i, n := 0, len(h); i < n; i++ {
		kv := &h[i]
		if key == string(kv.Key) {
			return true
		}
	}
	return false
}

func PeekArgBytes(h []ArgsKV, k []byte) []byte {
	for i, n := 0, len(h); i < n; i++ {
		kv := &h[i]
		if bytes.Equal(kv.Key, k) {
			return kv.Value
		}
	}
	return nil
}

type argsScanner struct {
	b []byte
}

func (s *argsScanner) next(kv *ArgsKV) bool {
	if len(s.b) == 0 {
		return false
	}

	isKey := true
	k := 0
	for i, c := range s.b {
		switch c {
		case '=':
			if isKey {
				isKey = false
				kv.Key = decodeArgAppend(kv.Key[:0], s.b[:i])
				k = i + 1
			}
		case '&':
			if isKey {
				kv.Key = decodeArgAppend(kv.Key[:0], s.b[:i])
				kv.Value = kv.Value[:0]
			} else {
				kv.Value = decodeArgAppend(kv.Value[:0], s.b[k:i])
			}
			s.b = s.b[i+1:]
			return true
		}
	}

	if isKey {
		kv.Key = decodeArgAppend(kv.Key[:0], s.b)
		kv.Value = kv.Value[:0]
	} else {
		kv.Value = decodeArgAppend(kv.Value[:0], s.b[k:])
	}
	s.b = s.b[len(s.b):]
	return true
}

func decodeArgAppend(dst, src []byte) []byte {
	if bytes.IndexByte(src, '%') < 0 && bytes.IndexByte(src, '+') < 0 {
		// fast path: src doesn't contain encoded chars
		return append(dst, src...)
	}

	// slow path
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '%' {
			if i+2 >= len(src) {
				return append(dst, src[i:]...)
			}
			x2 := hex2intTable[src[i+2]]
			x1 := hex2intTable[src[i+1]]
			if x1 == 16 || x2 == 16 {
				dst = append(dst, '%')
			} else {
				dst = append(dst, x1<<4|x2)
				i += 2
			}
		} else if c == '+' {
			dst = append(dst, ' ')
		} else {
			dst = append(dst, c)
		}
	}
	return dst
}

func DelAllArgsBytes(args []ArgsKV, key []byte) []ArgsKV {
	return DelAllArgs(args, B2s(key))
}

func DelAllArgs(args []ArgsKV, key string) []ArgsKV {
	for i, n := 0, len(args); i < n; i++ {
		kv := &args[i]
		if key == string(kv.Key) {
			tmp := *kv
			copy(args[i:], args[i+1:])
			n--
			args[n] = tmp
			args = args[:n]
		}
	}
	return args
}

func SetArgBytes(h []ArgsKV, key, value []byte) []ArgsKV {
	return SetArg(h, B2s(key), B2s(value))
}

func SetArg(h []ArgsKV, key, value string) []ArgsKV {
	n := len(h)
	for i := 0; i < n; i++ {
		kv := &h[i]
		if key == string(kv.Key) {
			kv.Value = append(kv.Value[:0], value...)
			return h
		}
	}
	return AppendArg(h, key, value)
}

func AppendArgBytes(h []ArgsKV, key, value []byte) []ArgsKV {
	return AppendArg(h, B2s(key), B2s(value))
}

func AppendArg(args []ArgsKV, key, value string) []ArgsKV {
	var kv *ArgsKV
	args, kv = AllocArg(args)
	kv.Key = append(kv.Key[:0], key...)
	kv.Value = append(kv.Value[:0], value...)
	return args
}

func AllocArg(h []ArgsKV) ([]ArgsKV, *ArgsKV) {
	n := len(h)
	if cap(h) > n {
		h = h[:n+1]
	} else {
		h = append(h, ArgsKV{})
	}
	return h, &h[n]
}

func ReleaseArg(h []ArgsKV) []ArgsKV {
	return h[:len(h)-1]
}

func CopyArgs(dst, src []ArgsKV) []ArgsKV {
	if cap(dst) < len(src) {
		tmp := make([]ArgsKV, len(src))
		copy(tmp, dst)
		dst = tmp
	}
	n := len(src)
	dst = dst[:n]
	for i := 0; i < n; i++ {
		dstKV := &dst[i]
		srcKV := &src[i]
		dstKV.Key = append(dstKV.Key[:0], srcKV.Key...)
		dstKV.Value = append(dstKV.Value[:0], srcKV.Value...)
	}
	return dst
}

func VisitArgs(args []ArgsKV, f func(k, v []byte)) {
	for i, n := 0, len(args); i < n; i++ {
		kv := &args[i]
		f(kv.Key, kv.Value)
	}
}

// decodeArgAppendNoPlus is almost identical to decodeArgAppend, but it doesn't
// substitute '+' with ' '.
//
// The function is copy-pasted from decodeArgAppend due to the preformance
// reasons only.
func DecodeArgAppendNoPlus(dst, src []byte) []byte {
	if bytes.IndexByte(src, '%') < 0 {
		// fast path: src doesn't contain encoded chars
		return append(dst, src...)
	}

	// slow path
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '%' {
			if i+2 >= len(src) {
				return append(dst, src[i:]...)
			}
			x2 := hex2intTable[src[i+2]]
			x1 := hex2intTable[src[i+1]]
			if x1 == 16 || x2 == 16 {
				dst = append(dst, '%')
			} else {
				dst = append(dst, x1<<4|x2)
				i += 2
			}
		} else {
			dst = append(dst, c)
		}
	}
	return dst
}
