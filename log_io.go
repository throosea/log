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
	"os"
	"path/filepath"
	"time"
	"io/ioutil"
	"regexp"
	"strings"
)

func writeLogEvent(log LogEvent) {
	log.publish()
	ensureLogFileExist()
	ensureTodayLog(log.getTime())
	writeLogEventToFile(log.getMessage())
}

func ensureTodayLog(t *time.Time) {
	if loggPreference.currentLogFileTime.Year() != t.Year() ||
		loggPreference.currentLogFileTime.Month() != t.Month() ||
		loggPreference.currentLogFileTime.Day() != t.Day() {
		moveToBackupLog()
	}
}

func ensureLogFileExist() {
	if loggPreference.logFileLoaded {
		return
	}

	var err error
	var stat os.FileInfo

	stat, err = os.Stat(loggPreference.logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			loggPreference.logFilePtr, err = os.Create(loggPreference.logFilePath)
			if err != nil {
				fmt.Printf("%s fail to create : %s", loggPreference.logFilePath, err)
				loggPreference.logFilePtr = nil
				return
			}
			loggPreference.currentLogFileTime = time.Now()
		} else if stat.IsDir() {
			fmt.Printf("%s path exist as directory. fail to logging", loggPreference.logFilePath)
			loggPreference.logFilePtr = nil
		}
	} else {
		loggPreference.logFilePtr, err = os.OpenFile(loggPreference.logFilePath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("fail to open : %s", err)
			loggPreference.logFilePtr = nil
		}
		loggPreference.currentLogFileTime = stat.ModTime()
	}

	loggPreference.logFileLoaded = true
}

// 오래된(지난 날짜) 로그 파일을 이동시키고 신규 로그 파일을 생성한다
func moveToBackupLog() {
	var err error
	var stat os.FileInfo

	stat, err = os.Stat(loggPreference.logFilePath)
	if err != nil {
		fmt.Printf("fail to stat log file : %s\n", err)
		loggPreference.logFilePtr = nil
		return
	}

	// close current log file ptr
	if loggPreference.logFilePtr != nil {
		loggPreference.logFilePtr.Close()
		loggPreference.logFilePtr = nil
	}

	// move current file to backup
	backupFilePath := fmt.Sprintf("%s%c%s.%s.log",
		loggPreference.logFolder,
		filepath.Separator,
		loggPreference.ProcessName, stat.ModTime().Format("2006-01-02"))
	os.Rename(loggPreference.logFilePath, backupFilePath)

	// open for new log file
	loggPreference.logFilePtr, err = os.OpenFile(loggPreference.logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("fail to open for new log file : %s\n", err)
	}

	stat, _ = loggPreference.logFilePtr.Stat()
	loggPreference.currentLogFileTime = stat.ModTime()

	go func() {
		keepingFileDaysChanged()
	}()
}

func writeLogEventToFile(s string) (n int, err error) {
	if loggPreference.logFilePtr == nil {
		return 0, nil
	}
	return loggPreference.logFilePtr.WriteString(s)
}

func keepingFileDaysChanged() {
	if loggPreference.KeepingFileDays < 1 {
		return
	}

	// find files in log path
	files, err := ioutil.ReadDir(loggPreference.logFolder)
	if err != nil {
		return
	}

	express := fmt.Sprintf("%s\\.[0-9]+-[0-9]+-[0-9]+\\.log", loggPreference.ProcessName)
	var validLogFileId = regexp.MustCompile(express)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "log") {
			continue
		}

		if !validLogFileId.MatchString(file.Name()) {
			continue
		}

		dotIndex := strings.Index(file.Name(), ".")
		if dotIndex < 1 {
			continue
		}

		lastDotIndex := strings.LastIndex(file.Name(), ".")
		if lastDotIndex <= dotIndex {
			continue
		}

		createdDateExpression := file.Name()[dotIndex+1:lastDotIndex]
		TIME_YYYYMMDD := "2006-01-02"
		createdDate, err := time.Parse(TIME_YYYYMMDD, createdDateExpression)
		if err != nil {
			continue
		}

		diff := time.Duration(24 * loggPreference.KeepingFileDays) * time.Hour
		deadline := time.Now().Add(-diff)
		if createdDate.Before(deadline) {
			os.Remove(filepath.Join(loggPreference.logFolder, file.Name()))
		}
	}
}