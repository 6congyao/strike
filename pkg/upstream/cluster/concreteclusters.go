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
)

type dynamicClusterBase struct {
	*cluster
}

func (dc *dynamicClusterBase) updateDynamicHostList(newHosts []Host, currentHosts []Host) (
	changed bool, finalHosts []Host, hostsAdded []Host, hostsRemoved []Host) {
	hostAddrs := make(map[string]bool)

	// N^2 loop, works for small and steady hosts
	for _, nh := range newHosts {
		nhAddr := nh.AddressString()
		if _, ok := hostAddrs[nhAddr]; ok {
			continue
		}

		hostAddrs[nhAddr] = true

		found := false
		for i := 0; i < len(currentHosts); {
			curNh := currentHosts[i]

			if nh.AddressString() == curNh.AddressString() {
				curNh.SetWeight(nh.Weight())
				finalHosts = append(finalHosts, curNh)
				currentHosts = append(currentHosts[:i], currentHosts[i+1:]...)
				found = true
			} else {
				i++
			}
		}

		if !found {
			finalHosts = append(finalHosts, nh)
			hostsAdded = append(hostsAdded, nh)
		}
	}

	if len(currentHosts) > 0 {
		hostsRemoved = currentHosts
	}

	if len(hostsAdded) > 0 || len(hostsRemoved) > 0 {
		changed = true
	} else {
		changed = false
	}

	return changed, finalHosts, hostsAdded, hostsRemoved
}

// SimpleCluster
// Cluster
type simpleInMemCluster struct {
	dynamicClusterBase

	hosts []Host
}

func newSimpleInMemCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaAPI bool) *simpleInMemCluster {
	cluster := newCluster(clusterConfig, sourceAddr, addedViaAPI, nil)

	return &simpleInMemCluster{
		dynamicClusterBase: dynamicClusterBase{
			cluster: cluster,
		},
	}
}

func (sc *simpleInMemCluster) UpdateHosts(newHosts []Host) {
	sc.mux.Lock()
	defer sc.mux.Unlock()

	var curHosts = make([]Host, len(sc.hosts))

	copy(curHosts, sc.hosts)
	changed, finalHosts, hostsAdded, hostsRemoved := sc.updateDynamicHostList(newHosts, curHosts)

	log.Println("update host changed:", changed)

	if changed {
		sc.hosts = finalHosts
		// todo: need to consider how to update healthyHost
		// Note: currently, we only use priority 0
		sc.prioritySet.GetOrCreateHostSet(0).UpdateHosts(sc.hosts,
			sc.hosts, nil, nil, hostsAdded, hostsRemoved)

		if sc.healthChecker != nil {
			sc.healthChecker.OnClusterMemberUpdate(hostsAdded, hostsRemoved)
		}

	}

	if len(sc.hosts) == 0 {
		log.Println(" after update final host is 0")
	}

	for i, f := range sc.hosts {
		log.Println("after update final host index and address:", i, f.AddressString())
	}
}
