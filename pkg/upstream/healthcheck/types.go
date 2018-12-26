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

package healthcheck

import (
	"strike/pkg/api/v2"
	"strike/pkg/upstream"
)

var HealthCheckFactoryInstance HealthCheckerFactory

type HealthCheckerFactory interface {
	New(config v2.HealthCheck) HealthChecker
}

// HealthChecker is a object that used to check an upstream cluster is health or not.
type HealthChecker interface {
	// Start starts health checking, which will continually monitor hosts in upstream cluster.
	Start()

	// Stop stops cluster health check. Client can use it to start/stop health check as a heartbeat.
	Stop()

	// AddHostCheckCompleteCb is a health check callback, which will be called on a check round-trip is completed for a specified host.
	AddHostCheckCompleteCb(cb HealthCheckCb)

	// OnClusterMemberUpdate updates cluster's hosts for health checking.
	OnClusterMemberUpdate(hostsAdded []upstream.Host, hostDel []upstream.Host)

	// SetCluster adds a cluster to health checker.
	SetCluster(cluster upstream.Cluster)
}

// HealthCheckCb is the health check's callback function
type HealthCheckCb func(host upstream.Host, changedState bool)
