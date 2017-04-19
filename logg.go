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
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)


const (
	LOG4FATIMA_PROP_BACKUP_DAYS           = "log4fatima.backup.days"
	LOG4FATIMA_PROP_SHOW_METHOD           = "log4fatima.method.show"
	LOG4FATIMA_PROP_SOURCE_PRINTSIZE      = "log4fatima.source.printsize"
	LOG4FATIMA_PROP_FILE_SIZE_LIMIT       = "log4fatima.filesize.limit"
	LOG4FATIMA_DEFAULT_BACKUP_FILE_NUMBER = 30
	LOG4FATIMA_DEFAULT_SOURCE_PRINTSIZE = 30
)

var writingLogEvent = false

var logEventChannel = make(chan LogEvent, 128)

var effectiveLogLevel LogLevel = LOG_NONE

type loggingStatus uint8

var loggStatus loggingStatus = LOGGING_STATUS_NOT_STARTED
var loggPreference preference

func Initialize(pref preference)  {
	if loggStatus > LOGGING_STATUS_NOT_STARTED {
		return
	}

	loggStatus = LOGGING_STATUS_RUNNING
	loggPreference = pref
	normalizePreference(&loggPreference)
	loggPreference.logFilePath = fmt.Sprintf("%s.log", filepath.Join(pref.logFolder, pref.ProcessName))
	if loggPreference.DeliveryMode == DELIVERY_MODE_ASYNC {
		go func() {
			for {
				logEvent := <-logEventChannel
				writingLogEvent = true
				writeLogEvent(logEvent)
				if len(logEventChannel) == 0 {
					writingLogEvent = false
				}
			}
		}()
	}
	SetLevel(loggPreference.DefaultLogLevel)
}

func SetSourcePrintSize(newValue int) {
	if newValue < 4 {
		return
	}

	loggPreference.SourcePrintSize = newValue
}

func SetShowMethod(newValue bool) {
	loggPreference.ShowMethod = newValue
}

func SetKeepingFileDays(days int)	{
	if days < 1 {
		return
	}

	var old = loggPreference.KeepingFileDays
	loggPreference.KeepingFileDays = days
	if old != days {
		Info("logging backup days changed to %d", loggPreference.KeepingFileDays)
		go func() {
			keepingFileDaysChanged()
		}()
	}
}

func SetFileSizeLimit(mb int)	{
	if mb < 1 {
		return
	}

	loggPreference.LogfileSizeLimit = mb
	Info("[Not yet support] logging file size limit to %d MB", loggPreference.LogfileSizeLimit)
}

func SetLevel(level LogLevel) {
	effectiveLogLevel = level
}

func GetLevel() LogLevel {
	return effectiveLogLevel
}

func Close() error {
	if loggPreference.DeliveryMode == DELIVERY_MODE_SYNC {
		return nil
	}

	for {
		if len(logEventChannel) == 0 && !writingLogEvent {
			loggStatus = LOGGING_STATUS_SHUTDOWN
			return nil
		}
		time.Sleep(time.Millisecond * 1)
	}
}

func Error(v ...interface{}) {
	if effectiveLogLevel >= LOG_ERROR && len(v) > 0 {
		print(LOG_ERROR, v...)
	}
}

func Warn(v ...interface{}) {
	if effectiveLogLevel >= LOG_WARN && len(v) > 0 {
		print(LOG_WARN, v...)
	}
}

func Info(v ...interface{}) {
	if effectiveLogLevel >= LOG_INFO && len(v) > 0 {
		print(LOG_INFO, v...)
	}
}

func Debug(v ...interface{}) {
	if effectiveLogLevel >= LOG_DEBUG && len(v) > 0 {
		print(LOG_DEBUG, v...)
	}
}

func Trace(v ...interface{}) {
	if effectiveLogLevel >= LOG_TRACE && len(v) > 0 {
		print(LOG_TRACE, v...)
	}
}

func print(level int, v ...interface{}) {
	pc, file, line, _ := runtime.Caller(2)

	var logEvent LogEvent

	if _, ok := v[len(v)-1].(error); ok {
		errEvent := newErrorTraceLogEvent(pc, file, line)
		for i := 3; i < loggPreference.MaxErrorTraceLevel; i++ {
			pc, file, line, exist := runtime.Caller(i)
			if !exist {
				break
			}
			point := TracePoint{pc: pc, file: file, line: line}
			errEvent.append(point)
		}
		logEvent = errEvent
	} else {
		logEvent = newGeneralLogEvent(pc, file, line)
	}

	logEvent.setLevel(level)
	logEvent.setArgs(v...)

	if loggPreference.DeliveryMode == DELIVERY_MODE_SYNC {
		writeLogEvent(logEvent)
	} else {
		//message.publish()
		logEventChannel <- logEvent
	}
}
