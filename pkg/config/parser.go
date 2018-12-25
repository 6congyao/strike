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
	"strike/pkg/stream"
)

var protocolsSupported = map[string]bool{
	string(protocol.MQ):        true,
	string(protocol.HTTP2):     true,
	string(protocol.HTTP1):     true,
	string(protocol.Xprotocol): true,
}

const (
	MinHostWeight               = uint32(1)
	MaxHostWeight               = uint32(128)
	DefaultMaxRequestPerConn    = uint32(1024)
	DefaultConnBufferLimitBytes = uint32(16 * 1024)
)

// ParsedCallback is an
// alias for closure func(data interface{}, endParsing bool) error
type ParsedCallback func(data interface{}, endParsing bool) error
type ContentKey string

var configParsedCBMaps = make(map[ContentKey][]ParsedCallback)

// Group of ContentKey
// notes: configcontentkey equals to the key of config file
const (
	ParseCallbackKeyCluster        ContentKey = "clusters"
	ParseCallbackKeyServiceRgtInfo ContentKey = "service_registry"
)

// RegisterConfigParsedListener
// used to register ParsedCallback
func RegisterConfigParsedListener(key ContentKey, cb ParsedCallback) {
	if cbs, ok := configParsedCBMaps[key]; ok {
		cbs = append(cbs, cb)
	} else {
		stdlog.Println("added to configParsedCBMaps:", key)
		cpc := []ParsedCallback{cb}
		configParsedCBMaps[key] = cpc
	}
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
				if downstream == string(protocol.HTTP2) {
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

// GetStreamFilters returns a stream filter factory by filter.Type
func GetStreamFilters(configs []v2.Filter) []stream.StreamFilterChainFactory {
	var factories []stream.StreamFilterChainFactory

	for _, c := range configs {
		sfcc, err := filter.CreateStreamFilterChainFactory(c.Type, c.Config)
		if err != nil {
			stdlog.Println(err)
			continue
		}
		factories = append(factories, sfcc)
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

// ParseFaultInjectFilter
func ParseFaultInjectFilter(cfg map[string]interface{}) *v2.FaultInject {
	filterConfig := &v2.FaultInject{}
	if data, err := json.Marshal(cfg); err == nil {
		json.Unmarshal(data, filterConfig)
	} else {
		stdlog.Println("parsing fault inject filter error")
	}
	return filterConfig
}

// ParseStreamFaultInjectFilter
func ParseStreamFaultInjectFilter(cfg map[string]interface{}) (*v2.StreamFaultInject, error) {
	filterConfig := &v2.StreamFaultInject{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}
	return filterConfig, nil
}

// ParseClusterConfig parses config data to api data, verify whether the config is valid
func ParseClusterConfig(clusters []v2.Cluster) ([]v2.Cluster, map[string][]v2.Host) {
	if len(clusters) == 0 {
		stdlog.Println("No Cluster provided in cluster config")
	}
	var pClusters []v2.Cluster
	clusterV2Map := make(map[string][]v2.Host)
	for _, c := range clusters {
		if c.Name == "" {
			stdlog.Println("[name] is required in cluster config")
		}
		if c.MaxRequestPerConn == 0 {
			c.MaxRequestPerConn = DefaultMaxRequestPerConn
			stdlog.Println("[max_request_per_conn] is not specified, use default value:",
				DefaultMaxRequestPerConn)
		}
		if c.ConnBufferLimitBytes == 0 {
			c.ConnBufferLimitBytes = DefaultConnBufferLimitBytes
			stdlog.Println("[conn_buffer_limit_bytes] is not specified, use default value:",
				DefaultConnBufferLimitBytes)
		}
		if c.LBSubSetConfig.FallBackPolicy > 2 {
			stdlog.Println("lb subset config 's fall back policy set error. ",
				"For 0, represent NO_FALLBACK",
				"For 1, represent ANY_ENDPOINT",
				"For 2, represent DEFAULT_SUBSET")
		}
		if _, ok := protocolsSupported[c.HealthCheck.Protocol]; !ok && c.HealthCheck.Protocol != "" {
			stdlog.Println("unsupported health check protocol:", c.HealthCheck.Protocol)
		}
		c.Hosts = parseHostConfig(c.Hosts)
		clusterV2Map[c.Name] = c.Hosts
		pClusters = append(pClusters, c)
	}
	// trigger all callbacks
	if cbs, ok := configParsedCBMaps[ParseCallbackKeyCluster]; ok {
		for _, cb := range cbs {
			cb(pClusters, false)
		}
	}
	return pClusters, clusterV2Map
}

func parseHostConfig(hosts []v2.Host) (hs []v2.Host) {
	for _, host := range hosts {
		host.Weight = transHostWeight(host.Weight)
		hs = append(hs, host)
	}
	return
}

func transHostWeight(weight uint32) uint32 {
	if weight > MaxHostWeight {
		return MaxHostWeight
	}
	if weight < MinHostWeight {
		return MinHostWeight
	}
	return weight
}
