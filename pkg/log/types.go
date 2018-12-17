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

package log

type Level uint8

const (
	FATAL Level = iota
	ERROR
	WARN
	INFO
	DEBUG
	TRACE
)

const (
	InfoPre  string = "[INFO]"
	DebugPre string = "[DEBUG]"
	WarnPre  string = "[WARN]"
	ErrorPre string = "[ERROR]"
	FatalPre string = "[Fatal]"
	TracePre string = "[TRACE]"
)

type Logger interface {
	Println(args ...interface{})

	Printf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Debugf(format string, args ...interface{})

	Warnf(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Tracef(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Close() error

	Reopen() error
}
