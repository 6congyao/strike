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

package upstream

import (
	"context"
	"net"
	"strike/pkg/api/v2"
	"strike/pkg/network"
)

type HealthFlag int

const (
	// The host is currently failing active health checks.
	FAILED_ACTIVE_HC HealthFlag = 0x1
	// The host is currently considered an outlier and has been ejected.
	FAILED_OUTLIER_CHECK HealthFlag = 0x02
)

// Defined for upstream
type Host interface {
	HostInfo

	// Create a connection for this host.
	CreateConnection(context context.Context) CreateConnectionData

	ClearHealthFlag(flag HealthFlag)

	ContainHealthFlag(flag HealthFlag) bool

	SetHealthFlag(flag HealthFlag)

	Health() bool

	Weight() uint32

	SetWeight(weight uint32)

	Used() bool

	SetUsed(used bool)
}

type HostInfo interface {
	Hostname() string

	Canary() bool

	// OriginMetaData used to get origin metadata, currently in map[string]string
	OriginMetaData() v2.Metadata

	ClusterInfo() ClusterInfo

	Address() net.Addr

	AddressString() string

	Config() v2.Host
}

type HostSet interface {

	// all hosts that make up the set at the current time.
	Hosts() []Host

	HealthyHosts() []Host

	HostsPerLocality() [][]Host

	HealthHostsPerLocality() [][]Host

	UpdateHosts(hosts []Host, healthyHost []Host, hostsPerLocality [][]Host,
		healthyHostPerLocality [][]Host, hostsAdded []Host, hostsRemoved []Host)

	Priority() uint32
}

type ClusterInfo interface {
	Name() string

	SourceAddress() net.Addr

	ConnectTimeout() int

	ConnBufferLimitBytes() uint32

	Features() int

	Metadata() v2.Metadata

	MaxRequestsPerConn() uint32
}

type CreateConnectionData struct {
	Connection network.Connection
	HostInfo   HostInfo
}

type ClusterManager interface {
	// Add or update a cluster via API.
	AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool

	SetInitializedCb(cb func())

	// Get, use to get the snapshot of a cluster
	GetClusterSnapshot(context context.Context, clusterName string) ClusterSnapshot

	// PutClusterSnapshot release snapshot lock
	PutClusterSnapshot(snapshot ClusterSnapshot)

	// UpdateClusterHosts used to update cluster's hosts
	// temp interface todo: remove it
	UpdateClusterHosts(clusterName string, priority uint32, hostConfigs []v2.Host) error

	// Get or Create tcp conn pool for a cluster
	TCPConnForCluster(balancerContext interface{}, snapshot ClusterSnapshot) CreateConnectionData

	// RemovePrimaryCluster used to remove cluster from set
	RemovePrimaryCluster(cluster string) error

	Shutdown() error

	SourceAddress() net.Addr

	VersionInfo() string

	LocalClusterName() string

	// ClusterExist, used to check whether 'clusterName' exist or not
	ClusterExist(clusterName string) bool

	// RemoveClusterHost, used to remove cluster's hosts
	RemoveClusterHost(clusterName string, hostAddress string) error

	// Destroy the cluster manager
	Destroy()
}

// ClusterSnapshot is a thread-safe cluster snapshot
type ClusterSnapshot interface {
	PrioritySet() PrioritySet

	ClusterInfo() ClusterInfo

	LoadBalancer() interface{}

	IsExistsHosts(metadata interface{}) bool
}

// InitializePhase type
type InitializePhase string

// InitializePhase types
const (
	Primary   InitializePhase = "Primary"
	Secondary InitializePhase = "Secondary"
)

type Cluster interface {
	Initialize(cb func())

	Info() ClusterInfo

	InitializePhase() InitializePhase

	PrioritySet() PrioritySet

	// set the cluster's health checker
	SetHealthChecker(hc HealthChecker)

	// return the cluster's health checker
	HealthChecker() HealthChecker
}

type MemberUpdateCallback func(priority uint32, hostsAdded []Host, hostsRemoved []Host)

// PrioritySet is a hostSet grouped by priority for a given cluster, for ease of load balancing.
type PrioritySet interface {
	// GetOrCreateHostSet returns the hostSet for this priority level, creating it if not exist.
	GetOrCreateHostSet(priority uint32) HostSet

	AddMemberUpdateCb(cb MemberUpdateCallback)

	HostSetsByPriority() []HostSet

	GetHostsInfo(priority uint32) []HostInfo
}

type ConcreteClusterInitHelper interface {
	Init()
}

// FailureType is the type of a failure
type FailureType string

// Failure types
const (
	FailureNetwork FailureType = "Network"
	FailurePassive FailureType = "Passive"
	FailureActive  FailureType = "Active"
)

// HealthCheckCb is the health check's callback function
type HealthCheckCb func(host Host, changedState bool)

// HealthChecker is a object that used to check an upstream cluster is health or not.
type HealthChecker interface {
	// Start starts health checking, which will continually monitor hosts in upstream cluster.
	Start()

	// Stop stops cluster health check. Client can use it to start/stop health check as a heartbeat.
	Stop()

	// AddHostCheckCompleteCb is a health check callback, which will be called on a check round-trip is completed for a specified host.
	AddHostCheckCompleteCb(cb HealthCheckCb)

	// OnClusterMemberUpdate updates cluster's hosts for health checking.
	OnClusterMemberUpdate(hostsAdded []Host, hostDel []Host)

	// SetCluster adds a cluster to health checker.
	SetCluster(cluster Cluster)
}

// HealthCheckSession is a health check session for an upstream host
type HealthCheckSession interface {
	// Start starts host health check
	Start()

	// Stop stops host health check
	Stop()

	// SetUnhealthy sets session as unhealthy for a specified reason
	SetUnhealthy(fType FailureType)
}

// TODO: move factory instance to a factory package

var HealthCheckFactoryInstance HealthCheckerFactory

type HealthCheckerFactory interface {
	New(config v2.HealthCheck) HealthChecker
}
