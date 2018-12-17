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
	"log"
	"net"
	"strike/pkg/api/v2"
	. "strike/pkg/upstream"
	"sync"
)

type cluster struct {
	initializationStarted          bool
	initializationCompleteCallback func()
	prioritySet                    *prioritySet
	info                           *clusterInfo
	mux                            sync.RWMutex
	initHelper                     ConcreteClusterInitHelper
	healthChecker                  HealthChecker
}

func NewCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool) Cluster {
	var newCluster Cluster

	switch clusterConfig.ClusterType {

	case v2.SIMPLE_CLUSTER, v2.DYNAMIC_CLUSTER, v2.EDS_CLUSTER:
		newCluster = newSimpleInMemCluster(clusterConfig, sourceAddr, addedViaAPI)
	default:
		return nil
	}

	// init health check for cluster's host
	if clusterConfig.HealthCheck.Protocol != "" {
		var hc HealthChecker
		hc = HealthCheckFactoryInstance.New(clusterConfig.HealthCheck)
		newCluster.SetHealthChecker(hc)
	}

	return newCluster
}

func newCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool, initHelper ConcreteClusterInitHelper) *cluster {
	cluster := cluster{
		prioritySet: &prioritySet{},
		info: &clusterInfo{
			name:                 clusterConfig.Name,
			clusterType:          clusterConfig.ClusterType,
			sourceAddr:           sourceAddr,
			addedViaAPI:          addedViaAPI,
			maxRequestsPerConn:   clusterConfig.MaxRequestPerConn,
			connBufferLimitBytes: clusterConfig.ConnBufferLimitBytes,
		},
		initHelper: initHelper,
	}

	//switch clusterConfig.LbType {
	//case v2.LB_RANDOM:
	//	cluster.info.lbType = types.Random
	//
	//case v2.LB_ROUNDROBIN:
	//	cluster.info.lbType = types.RoundRobin
	//}

	// TODO: init more props: maxrequestsperconn, connecttimeout, connectionbuflimit

	//cluster.info.resourceManager = NewResourceManager(clusterConfig.CirBreThresholds)

	//cluster.prioritySet.GetOrCreateHostSet(0)
	//cluster.prioritySet.AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
	//	// TODO: update cluster stats
	//})
	//
	//var lb types.LoadBalancer
	//
	//if cluster.Info().LbSubsetInfo().IsEnabled() {
	//	// use subset loadbalancer
	//	lb = NewSubsetLoadBalancer(cluster.Info().LbType(), cluster.PrioritySet(), cluster.Info().Stats(),
	//		cluster.Info().LbSubsetInfo())
	//
	//} else {
	//	// use common loadbalancer
	//	lb = NewLoadBalancer(cluster.Info().LbType(), cluster.PrioritySet())
	//}
	//
	//cluster.info.lbInstance = lb

	//mgr, err := mtls.NewTLSClientContextManager(&clusterConfig.TLS, cluster.info)
	//if err != nil {
	//	log.DefaultLogger.Fatalf("create tls context manager failed, %v", err)
	//}
	//cluster.info.tlsMng = mgr

	return &cluster
}

func (c *cluster) Initialize(cb func()) {
	c.initializationCompleteCallback = cb

	if c.initHelper != nil {
		c.initHelper.Init()
	}

	if c.initializationCompleteCallback != nil {
		c.initializationCompleteCallback()
	}
}

func (c *cluster) Info() ClusterInfo {
	return c.info
}

func (c *cluster) InitializePhase() InitializePhase {
	return Primary
}

func (c *cluster) PrioritySet() PrioritySet {
	return c.prioritySet
}

func (c *cluster) SetHealthChecker(hc HealthChecker) {
	c.healthChecker = hc
	c.healthChecker.SetCluster(c)
	c.healthChecker.Start()
	c.healthChecker.AddHostCheckCompleteCb(func(host Host, changedState bool) {
		if changedState {
			c.refreshHealthHosts(host)
		}
	})
}

// update health-hostSet for only one hostSet, reduce update times
func (c *cluster) refreshHealthHosts(host Host) {
	if host.Health() {
		log.Println("Add health host to cluster's healthHostSet by refreshHealthHosts:", host.AddressString())
		addHealthyHost(c.prioritySet.hostSets, host)
	} else {
		log.Println("Del host from cluster's healthHostSet by refreshHealthHosts:", host.AddressString())
		delHealthHost(c.prioritySet.hostSets, host)
	}
}

func addHealthyHost(hostSets []HostSet, host Host) {
	// Note: currently, one host only belong to a hostSet

	for i, hostSet := range hostSets {
		found := false

		for _, h := range hostSet.Hosts() {
			if h.AddressString() == host.AddressString() {
				log.Println("add healthy host in priority:", host.AddressString(), i)
				found = true
				break
			}
		}

		if found {
			newHealthHost := hostSet.HealthyHosts()
			newHealthHost = append(newHealthHost, host)
			newHealthyHostPerLocality := hostSet.HealthHostsPerLocality()
			newHealthyHostPerLocality[len(newHealthyHostPerLocality)-1] = append(newHealthyHostPerLocality[len(newHealthyHostPerLocality)-1], host)

			hostSet.UpdateHosts(hostSet.Hosts(), newHealthHost, hostSet.HostsPerLocality(),
				newHealthyHostPerLocality, nil, nil)
			break
		}
	}
}

func delHealthHost(hostSets []HostSet, host Host) {
	for i, hostSet := range hostSets {
		// Note: currently, one host only belong to a hostSet
		found := false

		for _, h := range hostSet.Hosts() {
			if h.AddressString() == host.AddressString() {
				log.Println("del healthy host in priority:", host.AddressString(), i)
				found = true
				break
			}
		}

		if found {
			newHealthHost := hostSet.HealthyHosts()
			newHealthyHostPerLocality := hostSet.HealthHostsPerLocality()

			for i, hh := range newHealthHost {
				if host.Hostname() == hh.Hostname() {
					//remove
					newHealthHost = append(newHealthHost[:i], newHealthHost[i+1:]...)
					break
				}
			}

			for i := range newHealthyHostPerLocality {
				for j := range newHealthyHostPerLocality[i] {

					if host.Hostname() == newHealthyHostPerLocality[i][j].Hostname() {
						newHealthyHostPerLocality[i] = append(newHealthyHostPerLocality[i][:j], newHealthyHostPerLocality[i][j+1:]...)
						break
					}
				}
			}

			hostSet.UpdateHosts(hostSet.Hosts(), newHealthHost, hostSet.HostsPerLocality(),
				newHealthyHostPerLocality, nil, nil)
			break
		}
	}
}

func (c *cluster) HealthChecker() HealthChecker {
	return c.healthChecker
}

type clusterInfo struct {
	name        string
	clusterType v2.ClusterType
	//lbType               types.LoadBalancerType // if use subset lb , lbType is used as inner LB algorithm for choosing subset's host
	//lbInstance           types.LoadBalancer     // load balancer used for this cluster
	sourceAddr           net.Addr
	connectTimeout       int
	connBufferLimitBytes uint32
	features             int
	maxRequestsPerConn   uint32
	addedViaAPI          bool

	healthCheckProtocol string
}

func (ci *clusterInfo) Name() string {
	return ci.name
}

func (ci *clusterInfo) SourceAddress() net.Addr {
	return ci.sourceAddr
}

func (ci *clusterInfo) ConnectTimeout() int {
	return ci.connectTimeout
}

func (ci *clusterInfo) ConnBufferLimitBytes() uint32 {
	return ci.connBufferLimitBytes
}

func (ci *clusterInfo) Features() int {
	return ci.features
}

func (ci *clusterInfo) Metadata() v2.Metadata {
	return v2.Metadata{}
}

func (ci *clusterInfo) MaxRequestsPerConn() uint32 {
	return ci.maxRequestsPerConn
}

type prioritySet struct {
	hostSets        []HostSet // Note: index is the priority
	updateCallbacks []MemberUpdateCallback
	mux             sync.RWMutex
}

func (ps *prioritySet) AddMemberUpdateCb(cb MemberUpdateCallback) {
	ps.updateCallbacks = append(ps.updateCallbacks, cb)
}

func (ps *prioritySet) HostSetsByPriority() []HostSet {
	ps.mux.RLock()
	defer ps.mux.RUnlock()

	return ps.hostSets
}

func (ps *prioritySet) GetHostsInfo(priority uint32) []HostInfo {
	var hostinfos []HostInfo
	if uint32(len(ps.hostSets)) > priority {
		hostset := ps.hostSets[priority]
		for _, host := range hostset.Hosts() {
			// host is an implement of hostinfo
			hostinfos = append(hostinfos, host)
		}
	}
	return hostinfos
}

func (ps *prioritySet) GetOrCreateHostSet(priority uint32) HostSet {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	// Create a priority set
	if uint32(len(ps.hostSets)) < priority+1 {

		for i := uint32(len(ps.hostSets)); i <= priority; i++ {
			hostSet := ps.createHostSet(i)
			hostSet.addMemberUpdateCb(func(priority uint32, hostsAdded []Host, hostsRemoved []Host) {
				for _, cb := range ps.updateCallbacks {
					cb(priority, hostsAdded, hostsRemoved)
				}
			})
			ps.hostSets = append(ps.hostSets, hostSet)
		}
	}

	return ps.hostSets[priority]
}

func (ps *prioritySet) createHostSet(priority uint32) *hostSet {
	return &hostSet{
		priority: priority,
	}
}
