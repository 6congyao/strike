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

package config

import (
	"encoding/json"
	stdlog "log"
	"net"
	"strike/pkg/api/v2"
	"strike/pkg/filter"
	"strike/pkg/log"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/server"
)

var protocolsSupported = map[string]bool{
	string(protocol.HTTP2):     true,
	string(protocol.HTTP1):     true,
	string(protocol.Xprotocol): true,
}

// ParseServerConfig
func ParseServerConfig(c *ServerConfig) *server.Config {
	sc := &server.Config{
		ServerName:      c.ServerName,
		LogPath:         c.DefaultLogPath,
		LogLevel:        parseLogLevel(c.DefaultLogLevel),
		GracefulTimeout: c.GracefulTimeout.Duration,
		Processor:       c.Processor,
		UseEdgeMode:     c.UseEdgeMode,
	}

	return sc
}

func parseLogLevel(level string) log.Level {
	if logLevel, ok := logLevelMap[level]; ok {
		return logLevel
	}
	return log.INFO
}

var logLevelMap = map[string]log.Level{
	"TRACE": log.TRACE,
	"DEBUG": log.DEBUG,
	"FATAL": log.FATAL,
	"ERROR": log.ERROR,
	"WARN":  log.WARN,
	"INFO":  log.INFO,
}

// ParseListenerConfig
func ParseListenerConfig(lc *v2.Listener) *v2.Listener {
	if lc.AddrConfig == "" {
		stdlog.Fatalln("[Address] is required in listener config")
	}
	addr, err := net.ResolveTCPAddr("tcp", lc.AddrConfig)
	if err != nil {
		stdlog.Fatalln("[Address] not valid:", lc.AddrConfig)
	}

	lc.Addr = addr
	lc.PerConnBufferLimitBytes = 1 << 15
	lc.LogLevel = uint8(parseLogLevel(lc.LogLevelConfig))
	return lc
}

// GetListenerDisableIO used to check downstream protocol and return ListenerDisableIO
func GetListenerDisableIO(c *v2.FilterChain) bool {
	for _, f := range c.Filters {
		if f.Type == v2.DEFAULT_NETWORK_FILTER {
			if downstream, ok := f.Config["downstream_protocol"]; ok {
				if downstream == string(protocol.HTTP2) || downstream == string(protocol.HTTP1) {
					return true
				}
			}
		}
	}
	return false
}

// ParseDelegationFilter
func ParseDelegationFilter(cfg map[string]interface{}) *v2.Delegation {
	delegationConfig := &v2.Delegation{}
	if data, err := json.Marshal(cfg); err == nil {
		json.Unmarshal(data, delegationConfig)
	} else {
		stdlog.Fatalln("Parsing Delegation network filter error")
	}

	if delegationConfig.AgentName == "" {
		stdlog.Println("AgentName in String Needed in Delegation Network Filter")
	}
	if delegationConfig.AgentType == "" {
		stdlog.Println("AgentType in String Needed in Delegation Network Filter")
	}

	return delegationConfig
}

func GetNetworkFilters(c *v2.FilterChain) []network.NetworkFilterChainFactory {
	var factories []network.NetworkFilterChainFactory
	for _, f := range c.Filters {
		factory, err := filter.CreateNetworkFilterChainFactory(f.Type, f.Config)
		if err != nil {
			stdlog.Println("network filter create failed: ", err)
			continue
		}
		factories = append(factories, factory)
	}
	return factories
}

// ParseProxyFilter
func ParseProxyFilter(cfg map[string]interface{}) *v2.Proxy {
	proxyConfig := &v2.Proxy{}
	if data, err := json.Marshal(cfg); err == nil {
		json.Unmarshal(data, proxyConfig)
	} else {
		stdlog.Fatalln("Parsing Proxy Network Filter Error")
	}

	if proxyConfig.DownstreamProtocol == "" || proxyConfig.UpstreamProtocol == "" {
		stdlog.Fatalln("Protocol in String Needed in Proxy Network Filter")
	} else if _, ok := protocolsSupported[proxyConfig.DownstreamProtocol]; !ok {
		stdlog.Fatalln("Invalid Downstream Protocol = ", proxyConfig.DownstreamProtocol)
	} else if _, ok := protocolsSupported[proxyConfig.UpstreamProtocol]; !ok {
		stdlog.Fatalln("Invalid Upstream Protocol = ", proxyConfig.UpstreamProtocol)
	}

	return proxyConfig
}
