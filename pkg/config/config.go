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
	"strike/pkg/api/v2"

	"io/ioutil"
	"log"
	"path/filepath"
)

type CfgMode uint8

const (
	File CfgMode = iota
	Xds
	Mix
)

var (
	configPath string
	config     StrikeConfig
)

// ServerConfig for making up server for Strike
type ServerConfig struct {
	//default logger
	ServerName      string `json:"server_name"`
	DefaultLogPath  string `json:"default_log_path,omitempty"`
	DefaultLogLevel string `json:"default_log_level,omitempty"`

	UseNetpollMode bool `json:"use_netpoll_mode,omitempty"`
	//graceful shutdown config
	GracefulTimeout v2.DurationConfig `json:"graceful_timeout"`

	//go processor number
	Processor int `json:"processor"`

	Listeners []v2.Listener `json:"listeners,omitempty"`
}

// ClusterManagerConfig for making up cluster manager
// Cluster is the global cluster of Strike
type ClusterManagerConfig struct {
	// Note: consider to use standard configure
	AutoDiscovery bool `json:"auto_discovery"`
	// Note: this is a hack method to realize cluster's  health check which push by registry
	RegistryUseHealthCheck bool         `json:"registry_use_health_check"`
	Clusters               []v2.Cluster `json:"clusters,omitempty"`
}

type StrikeConfig struct {
	Servers        []ServerConfig       `json:"servers,omitempty"`         //server config
	ClusterManager ClusterManagerConfig `json:"cluster_manager,omitempty"` //cluster config
}

func (c *StrikeConfig) Mode() CfgMode {
	return File
}

// Load json file and parse
func LoadJsonFile(path string) *StrikeConfig {
	log.Println("load config from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("load config failed, ", err)
	}
	configPath, _ = filepath.Abs(path)
	// translate to lower case
	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatalln("json unmarshal config failed, ", err)
	}
	return &config
}
