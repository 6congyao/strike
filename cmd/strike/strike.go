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

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strike/pkg/bootstrap"
	"strike/pkg/config"
	"strike/pkg/filter/network/delegation"
	"strike/pkg/network"
	"syscall"
)

func main() {
	flag.Parse()
	cfg := config.LoadJsonFile(*config.ConfigFile)
	registerDelegationHandler()
	bootstrap.Start(cfg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-signalChan:
		fmt.Println(fmt.Sprintf("Captured %v. Exiting...", s))
		os.Exit(0)
	}
}

func registerDelegationHandler() {
	delegation.RegisterAgent("apigateway", delegationHandler)
}

// delegation example
func delegationHandler(content interface{}) error {
	if c, ok := content.(network.Connection); ok {
		fmt.Println("got connection id:", c.ID())
		c.Close("", "")
	} else {
		return errors.New("type error")
	}
	return nil
}
