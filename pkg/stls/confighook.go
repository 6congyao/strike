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

package stls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
)

type defaultConfigHooks struct{}


// DefaultConfigHooks returns the default config hooks implement
func DefaultConfigHooks() ConfigHooks {
	return &defaultConfigHooks{}
}

// GetCertificate returns certificate if the index is cert/key file or pem string
func (hook *defaultConfigHooks) GetCertificate(certIndex, keyIndex string) (tls.Certificate, error) {
	if certIndex == "" || keyIndex == "" {
		return tls.Certificate{}, ErrorNoCertConfigure
	}
	if strings.Contains(certIndex, "-----BEGIN") && strings.Contains(keyIndex, "-----BEGIN") {
		return tls.X509KeyPair([]byte(certIndex), []byte(keyIndex))
	}
	return tls.LoadX509KeyPair(certIndex, keyIndex)
}

// GetX509Pool returns a CertPool with index's file or pem srting
func (hook *defaultConfigHooks) GetX509Pool(caIndex string) (*x509.CertPool, error) {
	if caIndex == "" {
		return nil, nil
	}
	var caBytes []byte
	var err error
	if strings.Contains(caIndex, "-----BEGIN") {
		caBytes = []byte(caIndex)
	} else {
		caBytes, err = ioutil.ReadFile(caIndex)
	}
	if err != nil {
		return nil, fmt.Errorf("load ca certificate error: %v", err)
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caBytes); !ok {
		return nil, fmt.Errorf("load ca certificate error: no certificate")
	}
	return pool, nil
}

// VerifyPeerCertificate returns a nil function, which means use standard tls verification
func (hook *defaultConfigHooks) VerifyPeerCertificate() func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return nil
}