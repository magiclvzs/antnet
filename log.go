package antnet

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type ILogger interface {
	Write(str string)
}

type ConsoleLogger struct {
	Ln bool
}

func (r *ConsoleLogger) Write(str string) {
	if r.Ln {
		fmt.Println(str)
	} else {
		fmt.Print(str)
	}
}

type FileLoggerFull func(path string)

type FileLogger struct {
	Path    string
	Ln      bool
	Timeout int //0表示不设置
	MaxSize int //0表示不限制，最大大小
	OnFull  FileLoggerFull

	size     int
	file     *os.File
	filename string
	extname  string
	dirname  string
}

func (r *FileLogger) Write(str string) {
	if r.file == nil {
		return
	}

	newsize := r.size
	if r.Ln {
		newsize += len(str) + 1
	} else {
		newsize += len(str)
	}

	if r.MaxSize > 0 && newsize >= r.MaxSize {
		r.file.Close()
		r.file = nil
		newpath := r.dirname + "/" + r.filename + fmt.Sprintf("_%d", time.Now().Unix()) + r.extname
		os.Rename(r.Path, newpath)
		file, err := os.OpenFile(r.Path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
		if err == nil {
			r.file = file
		}
		r.size = 0
		if r.OnFull != nil {
			r.OnFull(newpath)
		}
	}

	if r.file == nil {
		return
	}

	if r.Ln {
		r.file.WriteString(str)
		r.file.WriteString("\n")
		r.size += len(str) + 1
	} else {
		r.file.WriteString(str)
		r.size += len(str)
	}
}

type LogLevel int

const (
	LogLevelAllOn  LogLevel = iota //开放说有日志
	LogLevelDebug                  //调试信息
	LogLevelInfo                   //资讯讯息
	LogLevelWarn                   //警告状况发生
	LogLevelError                  //一般错误，可能导致功能不正常
	LogLevelFatal                  //严重错误，会导致进程退出
	LogLevelAllOff                 //关闭所有日志
)

type Log struct {
	logger  []ILogger
	cwrite  chan string
	clogger chan ILogger
	bufsize int
	stop    int32
	level   LogLevel
}

func (r *Log) initFileLogger(f *FileLogger) *FileLogger {
	if f.file == nil {
		f.Path, _ = filepath.Abs(f.Path)
		f.dirname = path.Dir(f.Path)
		f.extname = path.Ext(f.Path)
		f.filename = filepath.Base(f.Path[0 : len(f.Path)-len(f.extname)])
		os.MkdirAll(f.dirname, 0777)
		file, err := os.OpenFile(f.Path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
		if err == nil {
			f.file = file
			info, err := f.file.Stat()
			if err != nil {
				return nil
			}

			f.size = int(info.Size())
			return f
		}
	}
	return nil
}

func (r *Log) start() {
	goForLog(func(cstop chan struct{}) {
		for !r.IsStop() {
			select {
			case l, ok := <-r.clogger:
				if ok {
					r.logger = append(r.logger, l)
				}
			case s, ok := <-r.cwrite:
				if ok {
					for _, writer := range r.logger {
						writer.Write(s)
					}
				}
			case <-cstop:
			}
		}

		for s := range r.cwrite {
			for _, writer := range r.logger {
				writer.Write(s)
			}
		}
	})
}

func (r *Log) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		close(r.clogger)
		close(r.cwrite)
	}
}

func (r *Log) SetLogger(logger ILogger, imme bool) {
	defer func() { recover() }()
	if f, ok := logger.(*FileLogger); ok {
		if f := r.initFileLogger(f); f != nil {
			if imme {
				r.logger = append(r.logger, f)
			} else {
				r.clogger <- f
			}
		}
	} else {
		if imme {
			r.logger = append(r.logger, logger)
		} else {
			r.clogger <- logger
		}
	}
}
func (r *Log) Level() LogLevel {
	return r.level
}
func (r *Log) SetLevel(level LogLevel) {
	r.level = level
}

func isLogStop() bool {
	return stopForLog == 1
}

func (r *Log) IsStop() bool {
	if r.stop == 0 {
		if isLogStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *Log) write(levstr string, v ...interface{}) {
	defer func() { recover() }()
	if r.IsStop() {
		return
	}
	prefix := levstr
	_, file, line, ok := runtime.Caller(3)
	if ok {
		i := strings.LastIndex(file, "/") + 1
		prefix = fmt.Sprintf("[%s][%s][%s:%d]:", levstr, time.Now().Format("2006-01-02 15:04:05"), (string)(([]byte(file))[i:]), line)
	}
	if len(v) > 1 {
		r.cwrite <- prefix + fmt.Sprintf(v[0].(string), v[1:]...)
	} else {
		r.cwrite <- prefix + fmt.Sprint(v[0])
	}
}

func (r *Log) Debug(v ...interface{}) {
	if r.level <= LogLevelDebug {
		r.write("DEBUG", v...)
	}
}

func (r *Log) Info(v ...interface{}) {
	if r.level <= LogLevelInfo {
		r.write("INFO", v...)
	}
}

func (r *Log) Warn(v ...interface{}) {
	if r.level <= LogLevelWarn {
		r.write("WARN", v...)
	}
}

func (r *Log) Error(v ...interface{}) {
	if r.level <= LogLevelError {
		r.write("ERROR", v...)
	}
}

func (r *Log) Fatal(v ...interface{}) {
	if r.level <= LogLevelFatal {
		r.write("FATAL", v...)
	}
}

func NewLog(bufsize int, logger ...ILogger) *Log {
	log := &Log{
		bufsize: bufsize,
		cwrite:  make(chan string, bufsize),
		clogger: make(chan ILogger, 32),
		level:   LogLevelAllOn,
	}
	if len(logger) > 0 {
		for _, l := range logger {
			if f, ok := l.(*FileLogger); ok {
				if f := log.initFileLogger(f); f != nil {
					log.logger = append(log.logger, l)
				}
			} else {
				log.logger = append(log.logger, l)
			}
		}
	}
	log.start()
	return log
}

func LogInfo(v ...interface{}) {
	DefLog.Info(v...)
}

func LogDebug(v ...interface{}) {
	DefLog.Debug(v...)
}

func LogError(v ...interface{}) {
	DefLog.Error(v...)
}

func LogFatal(v ...interface{}) {
	DefLog.Fatal(v...)
}

func LogWarn(v ...interface{}) {
	DefLog.Warn(v...)
}
