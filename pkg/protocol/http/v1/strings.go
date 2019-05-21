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

var (
	defaultServerName  = []byte("strike")
	defaultUserAgent   = []byte("strike")
	defaultContentType = []byte("text/plain; charset=utf-8")
)

var (
	StrSlash            = []byte("/")
	StrSlashSlash       = []byte("//")
	StrSlashDotDot      = []byte("/..")
	StrSlashDotSlash    = []byte("/./")
	StrSlashDotDotSlash = []byte("/../")
	StrCRLF             = []byte("\r\n")
	StrHTTP             = []byte("http")
	StrHTTPS            = []byte("https")
	StrHTTP11           = []byte("HTTP/1.1")
	StrColonSlashSlash  = []byte("://")
	StrColonSpace       = []byte(": ")
	StrGMT              = []byte("GMT")

	StrResponseContinue = []byte("HTTP/1.1 100 Continue\r\n\r\n")

	StrGet    = []byte("GET")
	StrHead   = []byte("HEAD")
	StrPost   = []byte("POST")
	StrPut    = []byte("PUT")
	StrDelete = []byte("DELETE")

	StrExpect           = []byte("Expect")
	StrConnection       = []byte("Connection")
	StrContentLength    = []byte("Content-Length")
	StrContentType      = []byte("Content-Type")
	StrDate             = []byte("Date")
	StrHost             = []byte("Host")
	StrReferer          = []byte("Referer")
	StrServer           = []byte("Server")
	StrTransferEncoding = []byte("Transfer-Encoding")
	StrContentEncoding  = []byte("Content-Encoding")
	StrAcceptEncoding   = []byte("Accept-Encoding")
	StrUserAgent        = []byte("User-Agent")
	StrCookie           = []byte("Cookie")
	StrSetCookie        = []byte("Set-Cookie")
	StrLocation         = []byte("Location")
	StrIfModifiedSince  = []byte("If-Modified-Since")
	StrLastModified     = []byte("Last-Modified")
	StrAcceptRanges     = []byte("Accept-Ranges")
	StrRange            = []byte("Range")
	StrContentRange     = []byte("Content-Range")

	StrCookieExpires  = []byte("expires")
	StrCookieDomain   = []byte("domain")
	StrCookiePath     = []byte("path")
	StrCookieHTTPOnly = []byte("HttpOnly")
	StrCookieSecure   = []byte("secure")

	StrClose               = []byte("close")
	StrGzip                = []byte("gzip")
	StrDeflate             = []byte("deflate")
	StrKeepAlive           = []byte("keep-alive")
	StrKeepAliveCamelCase  = []byte("Keep-Alive")
	StrUpgrade             = []byte("Upgrade")
	StrChunked             = []byte("chunked")
	StrIdentity            = []byte("identity")
	Str100Continue         = []byte("100-continue")
	StrPostArgsContentType = []byte("application/x-www-form-urlencoded")
	StrMultipartFormData   = []byte("multipart/form-data")
	StrBoundary            = []byte("boundary")
	StrBytes               = []byte("bytes")
	StrTextSlash           = []byte("text/")
	StrApplicationSlash    = []byte("application/")
)
