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
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"strike/pkg/api/v2"
	. "strike/pkg/upstream"
	"sync"
)

var instanceMutex = sync.Mutex{}
var clusterMangerInstance *clusterManager

// ClusterManager
type clusterManager struct {
	sourceAddr             net.Addr
	primaryClusters        sync.Map
	protocolConnPool       sync.Map
	autoDiscovery          bool
	registryUseHealthCheck bool
}

func (cm *clusterManager) AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool {
	clusterName := cluster.Name

	ok := false
	if v, exist := cm.primaryClusters.Load(clusterName); exist {
		if !v.(*primaryCluster).addedViaAPI {
			return false
		}
		// update cluster
		ok = cm.updateCluster(cluster, v.(*primaryCluster), true)
	} else {
		// add new cluster
		ok = cm.loadCluster(cluster, true)
	}

	return ok
}

func (cm *clusterManager) SetInitializedCb(cb func()) {
	panic("implement me")
}

func (cm *clusterManager) GetClusterSnapshot(context context.Context, clusterName string) ClusterSnapshot {
	//if v, ok := cm.primaryClusters.Load(clusterName); ok {
	//	pc := v.(*primaryCluster)
	//	pcc := pc.cluster
	//
	//	clusterSnapshot := &clusterSnapshot{
	//		prioritySet:  pcc.PrioritySet(),
	//		clusterInfo:  pcc.Info(),
	//	}
	//
	//	return clusterSnapshot
	//}

	return nil
}

func (cm *clusterManager) PutClusterSnapshot(snapshot ClusterSnapshot) {
	panic("implement me")
}

func (cm *clusterManager) UpdateClusterHosts(clusterName string, priority uint32, hostConfigs []v2.Host) error {
	if v, ok := cm.primaryClusters.Load(clusterName); ok {
		pc := v.(*primaryCluster)
		var hosts []Host
		for _, hc := range hostConfigs {
			hosts = append(hosts, NewHost(hc, pc.cluster.Info()))
		}
		if err := pc.UpdateHosts(hosts); err != nil {
			return fmt.Errorf("UpdateClusterHosts failed, cluster's hostset %s can't be update", clusterName)
		}

		return nil
	}

	return fmt.Errorf("UpdateClusterHosts failed, cluster %s not found", clusterName)
}

func (cm *clusterManager) TCPConnForCluster(balancerContext interface{}, snapshot ClusterSnapshot) CreateConnectionData {
	panic("implement me")
}

func (cm *clusterManager) RemovePrimaryCluster(cluster string) error {
	panic("implement me")
}

func (cm *clusterManager) Shutdown() error {
	panic("implement me")
}

func (cm *clusterManager) SourceAddress() net.Addr {
	return cm.sourceAddr
}

func (cm *clusterManager) VersionInfo() string {
	panic("implement me")
}

func (cm *clusterManager) LocalClusterName() string {
	panic("implement me")
}

func (cm *clusterManager) ClusterExist(clusterName string) bool {
	panic("implement me")
}

func (cm *clusterManager) RemoveClusterHost(clusterName string, hostAddress string) error {
	panic("implement me")
}

func (cm *clusterManager) Destroy() {
	panic("implement me")
}

func (cm *clusterManager) loadCluster(clusterConfig v2.Cluster, addedViaAPI bool) bool {
	//clusterConfig.UseHealthCheck
	cluster := NewCluster(clusterConfig, cm.sourceAddr, addedViaAPI)

	if nil == cluster {
		return false
	}

	cluster.Initialize(func() {
		cluster.PrioritySet().AddMemberUpdateCb(func(priority uint32, hostsAdded []Host, hostsRemoved []Host) {
		})
	})

	cm.primaryClusters.Store(clusterConfig.Name, NewPrimaryCluster(cluster, &clusterConfig, addedViaAPI))

	return true
}

func (cm *clusterManager) updateCluster(clusterConf v2.Cluster, pcluster *primaryCluster, addedViaAPI bool) bool {
	if reflect.DeepEqual(clusterConf, pcluster.configUsed) {
		log.Println("update cluster but get duplicate configure")
		return true
	}

	if concretedCluster, ok := pcluster.cluster.(*simpleInMemCluster); ok {
		hosts := concretedCluster.hosts
		cluster := NewCluster(clusterConf, cm.sourceAddr, addedViaAPI)
		cluster.(*simpleInMemCluster).UpdateHosts(hosts)
		pcluster.UpdateCluster(cluster, &clusterConf, addedViaAPI)

		return true
	}

	return false
}

func NewClusterManager(sourceAddr net.Addr, clusters []v2.Cluster,
	clusterMap map[string][]v2.Host, autoDiscovery bool, useHealthCheck bool) ClusterManager {
	instanceMutex.Lock()
	defer instanceMutex.Unlock()
	if clusterMangerInstance != nil {
		return clusterMangerInstance
	}

	clusterMangerInstance = &clusterManager{
		sourceAddr:       sourceAddr,
		primaryClusters:  sync.Map{},
		protocolConnPool: sync.Map{},
		autoDiscovery:    true, //todo delete
	}

	//for k := range types.ConnPoolFactories {
	//	clusterMangerInstance.protocolConnPool.Store(k, &sync.Map{})
	//}

	//init clusterMngInstance when run app
	//initClusterMngAdapterInstance(clusterMangerInstance)

	//Add cluster to cm
	//Register upstream update type
	for _, cluster := range clusters {

		if !clusterMangerInstance.AddOrUpdatePrimaryCluster(cluster) {
			log.Println("NewClusterManager: AddOrUpdatePrimaryCluster failure, cluster name:", cluster.Name)
		}
	}

	// Add hosts to cluster
	// Note: currently, use priority = 0
	for clusterName, hosts := range clusterMap {
		clusterMangerInstance.UpdateClusterHosts(clusterName, 0, hosts)
	}

	return clusterMangerInstance
}

type primaryCluster struct {
	cluster     Cluster
	addedViaAPI bool
	configUsed  *v2.Cluster // used for update
	//configLock  *rcu.Value
	updateLock sync.Mutex
}

func NewPrimaryCluster(cluster Cluster, config *v2.Cluster, addedViaAPI bool) *primaryCluster {
	return &primaryCluster{
		cluster:     cluster,
		addedViaAPI: addedViaAPI,
		configUsed:  config,
		updateLock:  sync.Mutex{},
	}
}

func (pc *primaryCluster) UpdateCluster(cluster Cluster, config *v2.Cluster, addedViaAPI bool) error {
	if cluster == nil || config == nil {
		return errors.New("cannot update nil cluster or cluster config")
	}
	pc.updateLock.Lock()
	defer pc.updateLock.Unlock()
	pc.cluster = cluster
	pc.configUsed = deepCopyCluster(config)
	pc.addedViaAPI = addedViaAPI

	return nil
}

func (pc *primaryCluster) UpdateHosts(hosts []Host) error {
	pc.updateLock.Lock()
	defer pc.updateLock.Unlock()
	if c, ok := pc.cluster.(*simpleInMemCluster); ok {
		c.UpdateHosts(hosts)
	}
	config := deepCopyCluster(pc.configUsed)
	var hostConfig []v2.Host
	for _, h := range hosts {
		hostConfig = append(hostConfig, h.Config())
	}
	config.Hosts = hostConfig
	pc.configUsed = config

	return nil
}

type clusterSnapshot struct {
	prioritySet  PrioritySet
	clusterInfo  ClusterInfo
}

func deepCopyCluster(cluster *v2.Cluster) *v2.Cluster {
	if cluster == nil {
		return nil
	}
	clusterCopy := *cluster
	return &clusterCopy
}
