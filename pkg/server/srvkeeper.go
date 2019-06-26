/*
 * Copyright (c) 2019. LuCongyao <6congyao@gmail.com> .
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
	"log"
	"net"
	"os"
	"os/signal"
	"strike/pkg/admin"
	"sync"
	"syscall"
	"time"
)

func init() {
	catchSignals()
}

var (
	pidFile               string
	onProcessExit         []func()
	GracefulTimeout       = time.Second * 30 //default 30s
	shutdownCallbacksOnce sync.Once
	shutdownCallbacks     []func() error
)

func catchSignals() {
	//catchSignalsCrossPlatform()
	//catchSignalsPosix()
}

func catchSignalsCrossPlatform() {
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP,
			syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

		for sig := range sigchan {
			log.Println("signal received:", sig)
			switch sig {
			case syscall.SIGQUIT:
				// quit
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				os.Exit(0)
			case syscall.SIGTERM:
				// stop to quit
				exitCode := executeShutdownCallbacks("SIGTERM")
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				Stop()
				os.Exit(exitCode)
			case syscall.SIGUSR1:

			case syscall.SIGHUP:
				// reload, fork new strike
				reconfigure(true)
			case syscall.SIGUSR2:
			}
		}
	}()
}

func catchSignalsPosix() {
	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt)

		for i := 0; true; i++ {
			<-shutdown

			if i > 0 {
				for _, f := range onProcessExit {
					f() // important cleanup actions only
				}
				os.Exit(2)
			}

			// important cleanup actions before shutdown callbacks
			for _, f := range onProcessExit {
				f()
			}

			go func() {
				os.Exit(executeShutdownCallbacks("SIGINT"))
			}()
		}
	}()
}

func executeShutdownCallbacks(signame string) (exitCode int) {
	shutdownCallbacksOnce.Do(func() {
		var errs []error

		for _, cb := range shutdownCallbacks {
			errs = append(errs, cb())
		}

		if len(errs) > 0 {
			for _, err := range errs {
				log.Println("server shutdown with err:", signame, err)
			}
			exitCode = 4
		}
	})

	return
}

func reconfigure(start bool) {
	if start {
		startNewStrike()
		return
	}
	// set strike State Reconfiguring
	admin.SetStrikeState(admin.Reconfiguring)
	// if reconfigure failed, set strike state to Running
	defer admin.SetStrikeState(admin.Running)

	//// dump lastest config, and stop DumpConfigHandler()
	//config.DumpLock()
	//config.DumpConfig()
	//// if reconfigure failed, enable DumpConfigHandler()
	//defer config.DumpUnlock()

	// transfer listen fd
	var notify net.Conn
	var err error
	var n int
	var buf [1]byte
	if notify, err = sendInheritListeners(); err != nil {
		return
	}

	// Wait new strike parse configuration
	notify.SetReadDeadline(time.Now().Add(10 * time.Minute))
	n, err = notify.Read(buf[:])
	if n != 1 {
		log.Println("new strike start failed")
		return
	}

	// Wait for new strike start
	time.Sleep(3 * time.Second)

	// Stop accepting requests
	StopAccept()

	// Wait for all connections to be finished
	WaitConnectionsDone(GracefulTimeout)

	log.Println("process gracefully shutdown with pid:", os.Getpid())

	executeShutdownCallbacks("")

	// Stop the old server, all the connections have been closed and the new one is running
	os.Exit(0)
}

func startNewStrike() error {
	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}),
	}

	// Fork exec the new version of your server
	fork, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.Println("Fail to fork:", err)
		return err
	}

	log.Println("SIGHUP received: fork-exec to:", fork)
	return nil
}