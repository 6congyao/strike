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

package v1

import (
	"bytes"
	"errors"
	"io"
	"strike/utils"
	"sync"
	"time"
)

var zeroTime time.Time

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// CookieExpireUnlimited indicates that the cookie doesn't expire.
	CookieExpireUnlimited = zeroTime
)

// AcquireCookie returns an empty Cookie object from the pool.
//
// The returned object may be returned back to the pool with ReleaseCookie.
// This allows reducing GC load.
func AcquireCookie() *Cookie {
	return cookiePool.Get().(*Cookie)
}

// ReleaseCookie returns the Cookie object acquired with AcquireCookie back
// to the pool.
//
// Do not access released Cookie object, otherwise data races may occur.
func ReleaseCookie(c *Cookie) {
	c.Reset()
	cookiePool.Put(c)
}

var cookiePool = &sync.Pool{
	New: func() interface{} {
		return &Cookie{}
	},
}

// Cookie represents HTTP response cookie.
//
// Do not copy Cookie objects. Create new object and use CopyTo instead.
//
// Cookie instance MUST NOT be used from concurrently running goroutines.
type Cookie struct {
	noCopy utils.NoCopy

	key    []byte
	value  []byte
	expire time.Time
	domain []byte
	path   []byte

	httpOnly bool
	secure   bool

	bufKV utils.ArgsKV
	buf   []byte
}

// CopyTo copies src cookie to c.
func (c *Cookie) CopyTo(src *Cookie) {
	c.Reset()
	c.key = append(c.key[:0], src.key...)
	c.value = append(c.value[:0], src.value...)
	c.expire = src.expire
	c.domain = append(c.domain[:0], src.domain...)
	c.path = append(c.path[:0], src.path...)
	c.httpOnly = src.httpOnly
	c.secure = src.secure
}

// HTTPOnly returns true if the cookie is http only.
func (c *Cookie) HTTPOnly() bool {
	return c.httpOnly
}

// SetHTTPOnly sets cookie's httpOnly flag to the given value.
func (c *Cookie) SetHTTPOnly(httpOnly bool) {
	c.httpOnly = httpOnly
}

// Secure returns true if the cookie is secure.
func (c *Cookie) Secure() bool {
	return c.secure
}

// SetSecure sets cookie's secure flag to the given value.
func (c *Cookie) SetSecure(secure bool) {
	c.secure = secure
}

// Path returns cookie path.
func (c *Cookie) Path() []byte {
	return c.path
}

// SetPath sets cookie path.
func (c *Cookie) SetPath(path string) {
	c.buf = append(c.buf[:0], path...)
	c.path = normalizePath(c.path, c.buf)
}

// SetPathBytes sets cookie path.
func (c *Cookie) SetPathBytes(path []byte) {
	c.buf = append(c.buf[:0], path...)
	c.path = normalizePath(c.path, c.buf)
}

// Domain returns cookie domain.
//
// The returned domain is valid until the next Cookie modification method call.
func (c *Cookie) Domain() []byte {
	return c.domain
}

// SetDomain sets cookie domain.
func (c *Cookie) SetDomain(domain string) {
	c.domain = append(c.domain[:0], domain...)
}

// SetDomainBytes sets cookie domain.
func (c *Cookie) SetDomainBytes(domain []byte) {
	c.domain = append(c.domain[:0], domain...)
}

// Expire returns cookie expiration time.
//
// CookieExpireUnlimited is returned if cookie doesn't expire
func (c *Cookie) Expire() time.Time {
	expire := c.expire
	if expire.IsZero() {
		expire = CookieExpireUnlimited
	}
	return expire
}

// SetExpire sets cookie expiration time.
//
// Set expiration time to CookieExpireDelete for expiring (deleting)
// the cookie on the client.
//
// By default cookie lifetime is limited by browser session.
func (c *Cookie) SetExpire(expire time.Time) {
	c.expire = expire
}

// Value returns cookie value.
//
// The returned value is valid until the next Cookie modification method call.
func (c *Cookie) Value() []byte {
	return c.value
}

// SetValue sets cookie value.
func (c *Cookie) SetValue(value string) {
	c.value = append(c.value[:0], value...)
}

// SetValueBytes sets cookie value.
func (c *Cookie) SetValueBytes(value []byte) {
	c.value = append(c.value[:0], value...)
}

// Key returns cookie name.
//
// The returned value is valid until the next Cookie modification method call.
func (c *Cookie) Key() []byte {
	return c.key
}

// SetKey sets cookie name.
func (c *Cookie) SetKey(key string) {
	c.key = append(c.key[:0], key...)
}

// SetKeyBytes sets cookie name.
func (c *Cookie) SetKeyBytes(key []byte) {
	c.key = append(c.key[:0], key...)
}

// Reset clears the cookie.
func (c *Cookie) Reset() {
	c.key = c.key[:0]
	c.value = c.value[:0]
	c.expire = zeroTime
	c.domain = c.domain[:0]
	c.path = c.path[:0]
	c.httpOnly = false
	c.secure = false
}

// AppendBytes appends cookie representation to dst and returns
// the extended dst.
func (c *Cookie) AppendBytes(dst []byte) []byte {
	if len(c.key) > 0 {
		dst = append(dst, c.key...)
		dst = append(dst, '=')
	}
	dst = append(dst, c.value...)

	if !c.expire.IsZero() {
		c.bufKV.Value = utils.AppendHTTPDate(c.bufKV.Value[:0], c.expire)
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieExpires...)
		dst = append(dst, '=')
		dst = append(dst, c.bufKV.Value...)
	}
	if len(c.domain) > 0 {
		dst = appendCookiePart(dst, strCookieDomain, c.domain)
	}
	if len(c.path) > 0 {
		dst = appendCookiePart(dst, strCookiePath, c.path)
	}
	if c.httpOnly {
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieHTTPOnly...)
	}
	if c.secure {
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieSecure...)
	}
	return dst
}

// Cookie returns cookie representation.
//
// The returned value is valid until the next call to Cookie methods.
func (c *Cookie) Cookie() []byte {
	c.buf = c.AppendBytes(c.buf[:0])
	return c.buf
}

// String returns cookie representation.
func (c *Cookie) String() string {
	return string(c.Cookie())
}

// WriteTo writes cookie representation to w.
//
// WriteTo implements io.WriterTo interface.
func (c *Cookie) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(c.Cookie())
	return int64(n), err
}

var errNoCookies = errors.New("no cookies found")

// Parse parses Set-Cookie header.
func (c *Cookie) Parse(src string) error {
	c.buf = append(c.buf[:0], src...)
	return c.ParseBytes(c.buf)
}

// ParseBytes parses Set-Cookie header.
func (c *Cookie) ParseBytes(src []byte) error {
	c.Reset()

	var s cookieScanner
	s.b = src

	kv := &c.bufKV
	if !s.next(kv) {
		return errNoCookies
	}

	c.key = append(c.key[:0], kv.Key...)
	c.value = append(c.value[:0], kv.Value...)

	for s.next(kv) {
		if len(kv.Key) == 0 && len(kv.Value) == 0 {
			continue
		}
		switch string(kv.Key) {
		case "expires":
			v := utils.B2s(kv.Value)
			exptime, err := time.ParseInLocation(time.RFC1123, v, time.UTC)
			if err != nil {
				return err
			}
			c.expire = exptime
		case "domain":
			c.domain = append(c.domain[:0], kv.Value...)
		case "path":
			c.path = append(c.path[:0], kv.Value...)
		case "":
			switch string(kv.Value) {
			case "HttpOnly":
				c.httpOnly = true
			case "secure":
				c.secure = true
			}
		}
	}
	return nil
}

func appendCookiePart(dst, key, value []byte) []byte {
	dst = append(dst, ';', ' ')
	dst = append(dst, key...)
	dst = append(dst, '=')
	return append(dst, value...)
}

func getCookieKey(dst, src []byte) []byte {
	n := bytes.IndexByte(src, '=')
	if n >= 0 {
		src = src[:n]
	}
	return decodeCookieArg(dst, src, false)
}

func appendRequestCookieBytes(dst []byte, cookies []utils.ArgsKV) []byte {
	for i, n := 0, len(cookies); i < n; i++ {
		kv := &cookies[i]
		if len(kv.Key) > 0 {
			dst = append(dst, kv.Key...)
			dst = append(dst, '=')
		}
		dst = append(dst, kv.Value...)
		if i+1 < n {
			dst = append(dst, ';', ' ')
		}
	}
	return dst
}

func parseRequestCookies(cookies []utils.ArgsKV, src []byte) []utils.ArgsKV {
	var s cookieScanner
	s.b = src
	var kv *utils.ArgsKV
	cookies, kv = utils.AllocArg(cookies)
	for s.next(kv) {
		if len(kv.Key) > 0 || len(kv.Value) > 0 {
			cookies, kv = utils.AllocArg(cookies)
		}
	}
	return utils.ReleaseArg(cookies)
}

type cookieScanner struct {
	b []byte
}

func (s *cookieScanner) next(kv *utils.ArgsKV) bool {
	b := s.b
	if len(b) == 0 {
		return false
	}

	isKey := true
	k := 0
	for i, c := range b {
		switch c {
		case '=':
			if isKey {
				isKey = false
				kv.Key = decodeCookieArg(kv.Key, b[:i], false)
				k = i + 1
			}
		case ';':
			if isKey {
				kv.Key = kv.Key[:0]
			}
			kv.Value = decodeCookieArg(kv.Value, b[k:i], true)
			s.b = b[i+1:]
			return true
		}
	}

	if isKey {
		kv.Key = kv.Key[:0]
	}
	kv.Value = decodeCookieArg(kv.Value, b[k:], true)
	s.b = b[len(b):]
	return true
}

func decodeCookieArg(dst, src []byte, skipQuotes bool) []byte {
	for len(src) > 0 && src[0] == ' ' {
		src = src[1:]
	}
	for len(src) > 0 && src[len(src)-1] == ' ' {
		src = src[:len(src)-1]
	}
	if skipQuotes {
		if len(src) > 1 && src[0] == '"' && src[len(src)-1] == '"' {
			src = src[1 : len(src)-1]
		}
	}
	return append(dst[:0], src...)
}
