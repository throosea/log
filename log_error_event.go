//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 3. 6. PM 7:42
//

package log

import (
	"bytes"
	//	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func newErrorTraceLogEvent(pc uintptr, file string, line int) *ErrorTraceLogEvent {
	event := ErrorTraceLogEvent{}
	event.t = time.Now()
	event.announce = false
	event.pc = pc
	event.file = file
	event.line = line
	event.tracePoint = make([]TracePoint, 0)
	if loggPreference.ShowMethod {
		event.funcName = findFunctionName(pc)
	}
	return &event
}

type TracePoint struct {
	pc   uintptr
	file string
	line int
}

type ErrorTraceLogEvent struct {
	GeneralLogEvent
	announce   bool
	tracePoint []TracePoint
}

func (event *ErrorTraceLogEvent) append(point TracePoint) {
	event.tracePoint = append(event.tracePoint, point)
}

func (event *ErrorTraceLogEvent) publish() {
	var buffer bytes.Buffer

	codeLine := event.buildMessage(func() string {
		size := len(event.message)
		if size == 1 {
			event.announce = true
			return fmt.Sprintf("(%s) :: %s", reflect.TypeOf(event.message[0]).String(), event.message[0])
		} else {
			if format, ok := event.message[0].(string); ok {
				if size == 2 {
					return format
				} else {
					return fmt.Sprintf(format, event.message[1:size-1]...)
				}
			}
			event.announce = true
			return fmt.Sprintf("(%s) :: %s", reflect.TypeOf(event.message[size-1]).String(), event.message[size-1])
		}
	})

	buffer.WriteString(codeLine)
	buffer.WriteString(event.getTrace())
	event.published = buffer.String()
}

func (event *ErrorTraceLogEvent) getTrace() string {
	var buffer bytes.Buffer

	if event.announce {
		buffer.WriteString("\tTRACE <<<\n")
	} else {
		err := event.message[len(event.message)-1]
		buffer.WriteString(fmt.Sprintf("\t(%s) :: %s\n\tTRACE <<<\n", reflect.TypeOf(err).String(), err))
	}
	for _, v := range event.tracePoint {
		buffer.WriteString(fmt.Sprintf("\t[%s(), %s:%d]\n", findFunctionName(v.pc), buildSourcePath(v.file), v.line))
	}
	return buffer.String()

}

func buildSourcePath(file string) string {
	var location = file
	var found = strings.LastIndex(file, "/src/")
	if found > 0 {
		location = string(file[found+5:])
	}

	return strings.Replace(location, "/", ".", -1)
}
