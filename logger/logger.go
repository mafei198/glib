/*
The MIT License (MIT)

Copyright (c) 2018 SavinMax. All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package logger

import (
	"fmt"
	"log"
	"sync/atomic"
)

var logLevel int32 = LL_DEBUG
var logDriver int32 = LD_DEFAULT

const (
	LD_DEFAULT = iota
	LD_ZAP
)

const (
	LL_DEBUG = iota
	LL_INFO
	LL_WARNING
	LL_ERROR
)

func SetLogger(ld int32) {
	logDriver = ld
	switch ld {
	case LD_DEFAULT:
		Start()
	case LD_ZAP:
		StartZap()
	}
}

func SetLevel(level string) {
	switch level {
	case "DEBUG":
		logLevel = LL_DEBUG
	case "INFO":
		logLevel = LL_INFO
	case "WARNING":
		logLevel = LL_WARNING
	case "ERROR":
		logLevel = LL_ERROR
	default:
		println("invalid log level: ", level)
	}
}

func ERR(v ...interface{}) {
	/*
		event := sentry.NewEvent()
		event.Level = sentry.LevelError
		event.Message = strings.TrimRight(fmt.Sprintln(v...), "\n")
		sentry.CaptureEvent(event)
	*/
	switch logDriver {
	case LD_ZAP:
		sugarLogger.Error(v...)
	default:
		add(formatErr(v...))
	}
}

func ERRDirect(v ...interface{}) {
	add(formatErr(v...))
}

func WARN(v ...interface{}) {
	if logLevel > LL_WARNING {
		return
	}
	switch logDriver {
	case LD_ZAP:
		sugarLogger.Warn(v...)
	default:
		add(formatWarn(v...))
	}
}

func INFO(v ...interface{}) {
	if logLevel > LL_INFO {
		return
	}
	switch logDriver {
	case LD_ZAP:
		sugarLogger.Info(v...)
	default:
		add(formatInfo(v...))
	}
}

func DEBUG(v ...interface{}) {
	if logLevel > LL_DEBUG {
		return
	}
	switch logDriver {
	case LD_ZAP:
		sugarLogger.Debug(v...)
	default:
		add(formatDebug(v...))
	}
}

var msgQueue chan string
var dropped int32 = 0

const bufferLen = 1024

func Start() {
	msgQueue = make(chan string, bufferLen)
	go worker()
}

func Stop() {
	if logDriver == LD_ZAP {
		_ = sugarLogger.Sync()
	}
}

func add(msg string) {
	if msgQueue == nil {
		log.Print(msg)
		return
	}
	if len(msgQueue) >= bufferLen {
		atomic.AddInt32(&dropped, 1)
		return
	}
	if dropped > 0 {
		println("dropped: ", dropped)
		dropped = 0
	}
	msgQueue <- msg
}

func worker() {
	for msg := range msgQueue {
		log.Print(msg)
	}
}

const (
	debugFormator = "\033[1;35m[DEBUG] %v \033[0m\n"
	infoFormator  = "\033[32m[INFO] %v \033[0m\n"
	warnFormator  = "\033[1;33m[WARN] %v \033[0m\n"
	errorFormator = "\033[1;4;31m[ERROR] %v \033[0m\n"
)

func formatDebug(v ...interface{}) string {
	return fmt.Sprintf(debugFormator, fmt.Sprint(v...))
}

func formatInfo(v ...interface{}) string {
	return fmt.Sprintf(infoFormator, fmt.Sprint(v...))
}

func formatWarn(v ...interface{}) string {
	return fmt.Sprintf(warnFormator, fmt.Sprint(v...))
}

func formatErr(v ...interface{}) string {
	return fmt.Sprintf(errorFormator, fmt.Sprintln(v...))
}
