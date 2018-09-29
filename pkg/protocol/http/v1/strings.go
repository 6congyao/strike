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
	defaultServerName  = []byte("fasthttp")
	defaultUserAgent   = []byte("fasthttp")
	defaultContentType = []byte("text/plain; charset=utf-8")
)

var (
	strSlash            = []byte("/")
	strSlashSlash       = []byte("//")
	strSlashDotDot      = []byte("/..")
	strSlashDotSlash    = []byte("/./")
	strSlashDotDotSlash = []byte("/../")
	strCRLF             = []byte("\r\n")
	strHTTP             = []byte("http")
	strHTTPS            = []byte("https")
	strHTTP11           = []byte("HTTP/1.1")
	strColonSlashSlash  = []byte("://")
	strColonSpace       = []byte(": ")
	strGMT              = []byte("GMT")

	strResponseContinue = []byte("HTTP/1.1 100 Continue\r\n\r\n")

	strGet    = []byte("GET")
	strHead   = []byte("HEAD")
	strPost   = []byte("POST")
	strPut    = []byte("PUT")
	strDelete = []byte("DELETE")

	strExpect           = []byte("Expect")
	strConnection       = []byte("Connection")
	strContentLength    = []byte("Content-Length")
	strContentType      = []byte("Content-Type")
	strDate             = []byte("Date")
	strHost             = []byte("Host")
	strReferer          = []byte("Referer")
	strServer           = []byte("Server")
	strTransferEncoding = []byte("Transfer-Encoding")
	strContentEncoding  = []byte("Content-Encoding")
	strAcceptEncoding   = []byte("Accept-Encoding")
	strUserAgent        = []byte("User-Agent")
	strCookie           = []byte("Cookie")
	strSetCookie        = []byte("Set-Cookie")
	strLocation         = []byte("Location")
	strIfModifiedSince  = []byte("If-Modified-Since")
	strLastModified     = []byte("Last-Modified")
	strAcceptRanges     = []byte("Accept-Ranges")
	strRange            = []byte("Range")
	strContentRange     = []byte("Content-Range")

	strCookieExpires  = []byte("expires")
	strCookieDomain   = []byte("domain")
	strCookiePath     = []byte("path")
	strCookieHTTPOnly = []byte("HttpOnly")
	strCookieSecure   = []byte("secure")

	strClose               = []byte("close")
	strGzip                = []byte("gzip")
	strDeflate             = []byte("deflate")
	strKeepAlive           = []byte("keep-alive")
	strKeepAliveCamelCase  = []byte("Keep-Alive")
	strUpgrade             = []byte("Upgrade")
	strChunked             = []byte("chunked")
	strIdentity            = []byte("identity")
	str100Continue         = []byte("100-continue")
	strPostArgsContentType = []byte("application/x-www-form-urlencoded")
	strMultipartFormData   = []byte("multipart/form-data")
	strBoundary            = []byte("boundary")
	strBytes               = []byte("bytes")
	strTextSlash           = []byte("text/")
	strApplicationSlash    = []byte("application/")
)
