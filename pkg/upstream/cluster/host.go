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

package cluster

import (
	"context"
	"net"
	"strike/pkg/api/v2"
	. "strike/pkg/upstream"
	"sync"
)

type host struct {
	hostInfo
	weight uint32
	used   bool

	healthFlags uint64
}

func (h *host) CreateConnection(context context.Context) CreateConnectionData {
	panic("implement me")
}

func (h *host) ClearHealthFlag(flag HealthFlag) {
	panic("implement me")
}

func (h *host) ContainHealthFlag(flag HealthFlag) bool {
	panic("implement me")
}

func (h *host) SetHealthFlag(flag HealthFlag) {
	panic("implement me")
}

func (h *host) Health() bool {
	panic("implement me")
}

func (h *host) Weight() uint32 {
	panic("implement me")
}

func (h *host) SetWeight(weight uint32) {
	panic("implement me")
}

func (h *host) Used() bool {
	panic("implement me")
}

func (h *host) SetUsed(used bool) {
	panic("implement me")
}

func NewHost(config v2.Host, clusterInfo ClusterInfo) Host {
	addr, _ := net.ResolveTCPAddr("tcp", config.Address)

	return &host{
		hostInfo: newHostInfo(addr, config, clusterInfo),
		weight:   config.Weight,
	}
}

type hostInfo struct {
	hostname       string
	address        net.Addr
	addressString  string
	canary         bool
	clusterInfo    ClusterInfo
	originMetaData v2.Metadata
	tlsDisable     bool
	config         v2.Host

	// TODO: locality, outlier, healthchecker
}

func (hi *hostInfo) Hostname() string {
	return hi.hostname
}

func (hi *hostInfo) Canary() bool {
	return hi.canary
}

func (hi *hostInfo) OriginMetaData() v2.Metadata {
	return hi.originMetaData
}

func (hi *hostInfo) ClusterInfo() ClusterInfo {
	return hi.clusterInfo
}

func (hi *hostInfo) Address() net.Addr {
	return hi.address
}

func (hi *hostInfo) AddressString() string {
	return hi.addressString
}

func (hi *hostInfo) Config() v2.Host {
	return hi.config
}

func newHostInfo(addr net.Addr, config v2.Host, clusterInfo ClusterInfo) hostInfo {
	//var name string
	//if clusterInfo != nil {
	//	name = clusterInfo.Name()
	//}
	return hostInfo{
		address:        addr,
		addressString:  config.Address,
		hostname:       config.Hostname,
		clusterInfo:    clusterInfo,
		originMetaData: config.MetaData,
		tlsDisable:     config.TLSDisable,
		config:         config,
	}
}

type hostSet struct {
	priority                uint32
	hosts                   []Host
	healthyHosts            []Host
	hostsPerLocality        [][]Host
	healthyHostsPerLocality [][]Host
	mux                     sync.RWMutex
	updateCallbacks         []MemberUpdateCallback
	metadata                v2.Metadata
}

func (hs *hostSet) Hosts() []Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.hosts
}

func (hs *hostSet) HealthyHosts() []Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.healthyHosts
}

func (hs *hostSet) HostsPerLocality() [][]Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.hostsPerLocality
}

func (hs *hostSet) HealthHostsPerLocality() [][]Host {
	hs.mux.RLock()
	defer hs.mux.RUnlock()

	return hs.healthyHostsPerLocality
}

func (hs *hostSet) UpdateHosts(hosts []Host, healthyHost []Host, hostsPerLocality [][]Host,
	healthyHostPerLocality [][]Host, hostsAdded []Host, hostsRemoved []Host) {

}

func (hs *hostSet) Priority() uint32 {
	return hs.priority
}

func (hs *hostSet) addMemberUpdateCb(cb MemberUpdateCallback) {
	hs.updateCallbacks = append(hs.updateCallbacks, cb)
}
