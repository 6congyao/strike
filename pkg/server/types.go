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

package server

import (
	"os"
	"strike/pkg/api/v2"
	"strike/pkg/log"
	"strike/pkg/network"
	"time"
)

const (
	BasePath = string(os.PathSeparator) + "home" + string(os.PathSeparator) +
		"admin" + string(os.PathSeparator) + "strike"

	LogBasePath = BasePath + string(os.PathSeparator) + "logs"

	LogDefaultPath = LogBasePath + string(os.PathSeparator) + "strike.log"

	PidFileName = "strike.pid"
)

type Config struct {
	ServerName      string
	LogPath         string
	LogLevel        log.Level
	GracefulTimeout time.Duration
	Processor       int
	UseEdgeMode     bool
}

type Server interface {
	AddListener(lc *v2.Listener) (network.ListenerEventListener, error)

	Start()

	Restart()

	Close()

	Handler() network.ConnectionHandler
}
