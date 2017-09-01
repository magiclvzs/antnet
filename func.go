package antnet

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func Print(a ...interface{}) (int, error) {
	return fmt.Print(a...)
}
func Println(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}
func Printf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(format, a...)
}
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
func Stop() {
	if !atomic.CompareAndSwapInt32(&stop, 0, 1) {
		return
	}

	for _, v := range msgqueMap {
		v.Stop()
	}

	stopMapLock.Lock()
	for k, v := range stopMap {
		close(v)
		delete(stopMap, k)
	}
	stopMapLock.Unlock()

	stopChan <- nil

	for _, v := range redisManagers {
		v.close()
	}

	LogInfo("Server Stop")
	waitAll.Wait()

	if !atomic.CompareAndSwapInt32(&stopForLog, 0, 1) {
		return
	}

	stopMapForLogLock.Lock()
	for k, v := range stopMapForLog {
		close(v)
		delete(stopMapForLog, k)
	}
	stopMapForLogLock.Unlock()
	waitAllForLog.Wait()
}

func IsStop() bool {
	return stop == 1
}

func IsRuning() bool {
	return stop == 0
}

func Now() time.Time {
	return time.Now()
}

func CmdAct(cmd, act uint8) int {
	return int(cmd)<<8 + int(act)
}

func Tag(cmd, act uint8, index uint16) int {
	return int(cmd)<<16 + int(act)<<8 + int(index)
}

func MD5Str(s string) string {
	return MD5Bytes([]byte(s))
}

func MD5Bytes(s []byte) string {
	md5Ctx := md5.New()
	md5Ctx.Write(s)
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func Go(fn func()) {
	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		_, file, line, _ := runtime.Caller(1)
		i := strings.LastIndex(file, "/") + 1
		i = strings.LastIndex((string)(([]byte(file))[:i-1]), "/") + 1
		debugStr = Sprintf("%s:%d", (string)(([]byte(file))[i:]), line)
		LogDebug("goroutine start id:%d count:%d from:%s", id, id, debugStr)
	}
	go func() {
		Try(fn, nil)
		waitAll.Done()
		c = atomic.AddInt32(&gocount, ^int32(0))

		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func GoArgs(fn func(...interface{}), args ...interface{}) {
	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		_, file, line, _ := runtime.Caller(1)
		i := strings.LastIndex(file, "/") + 1
		i = strings.LastIndex((string)(([]byte(file))[:i-1]), "/") + 1
		debugStr = Sprintf("%s:%d", (string)(([]byte(file))[i:]), line)
		LogDebug("goroutine start id:%d count:%d from:%s", id, id, debugStr)
	}

	go func() {
		Try(func() { fn(args...) }, nil)

		waitAll.Done()
		c = atomic.AddInt32(&gocount, ^int32(0))
		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func Go2(fn func(cstop chan struct{})) bool {
	if IsStop() {
		return false
	}
	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		_, file, line, _ := runtime.Caller(1)
		i := strings.LastIndex(file, "/") + 1
		i = strings.LastIndex((string)(([]byte(file))[:i-1]), "/") + 1
		debugStr = Sprintf("%s:%d", (string)(([]byte(file))[i:]), line)
		LogDebug("goroutine start id:%d count:%d from:%s", id, id, debugStr)
	}

	go func() {
		id := atomic.AddUint64(&goId, 1)
		cstop := make(chan struct{})
		stopMapLock.Lock()
		stopMap[id] = cstop
		stopMapLock.Unlock()
		Try(func() { fn(cstop) }, nil)

		stopMapLock.Lock()
		if _, ok := stopMap[id]; ok {
			close(cstop)
			delete(stopMap, id)
		}
		stopMapLock.Unlock()

		waitAll.Done()
		c = atomic.AddInt32(&gocount, ^int32(0))
		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
	return true
}

func goForLog(fn func(cstop chan struct{})) bool {
	if IsStop() {
		return false
	}
	waitAllForLog.Add(1)

	go func() {
		id := atomic.AddUint64(&goId, 1)
		cstop := make(chan struct{})
		stopMapForLogLock.Lock()
		stopMapForLog[id] = cstop
		stopMapForLogLock.Unlock()
		fn(cstop)

		stopMapForLogLock.Lock()
		if _, ok := stopMapForLog[id]; ok {
			close(cstop)
			delete(stopMapForLog, id)
		}
		stopMapForLogLock.Unlock()

		waitAllForLog.Done()
	}()
	return true
}

func WaitForSystemExit() {
	statis.StartTime = time.Now()
	stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	for stop == 0 {
		select {
		case <-stopChan:
			Stop()
		}
	}
	Stop()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func Daemon(skip []string) {
	if os.Getppid() != 1 {
		filePath, _ := filepath.Abs(os.Args[0])
		newCmd := []string{}
		for _, v := range os.Args {
			add := true
			for _, s := range skip {
				if strings.Contains(v, s) {
					add = false
					break
				}
			}
			if add {
				newCmd = append(newCmd, v)
			}
		}
		cmd := exec.Command(filePath)
		cmd.Args = newCmd
		cmd.Start()
	}
}

func LogStack() {
	buf := make([]byte, 1<<12)
	LogError(string(buf[:runtime.Stack(buf, false)]))
}

func GetStatis() *Statis {
	statis.GoCount = int(gocount)
	statis.MsgqueCount = len(msgqueMap)
	return &statis
}

func Atoi(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return i
}

func Itoa(num interface{}) string {
	switch n := num.(type) {
	case int8:
		return strconv.FormatInt(int64(n), 10)
	case int16:
		return strconv.FormatInt(int64(n), 10)
	case int32:
		return strconv.FormatInt(int64(n), 10)
	case int:
		return strconv.FormatInt(int64(n), 10)
	case int64:
		return strconv.FormatInt(int64(n), 10)
	case uint8:
		return strconv.FormatUint(uint64(n), 10)
	case uint16:
		return strconv.FormatUint(uint64(n), 10)
	case uint32:
		return strconv.FormatUint(uint64(n), 10)
	case uint:
		return strconv.FormatUint(uint64(n), 10)
	case uint64:
		return strconv.FormatUint(uint64(n), 10)
	}
	return ""
}

func JoinStr(a []string, sep string) string {
	return strings.Join(a, sep)
}

var allIp []string

func GetSelfIp(ifnames ...string) []string {
	if allIp != nil {
		return allIp
	}
	inters, _ := net.Interfaces()
	if len(ifnames) == 0 {
		ifnames = []string{"eth", "lo", "无线网络连接", "本地连接"}
	}

	filterFunc := func(name string) bool {
		for _, v := range ifnames {
			if strings.Index(name, v) != -1 {
				return true
			}
		}
		return false
	}

	for _, inter := range inters {
		if !filterFunc(inter.Name) {
			continue
		}
		addrs, _ := inter.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					allIp = append(allIp, ipnet.IP.String())
				}
			}
		}
	}
	return allIp
}

func GetSelfIntraIp(ifnames ...string) (ips []string) {
	all := GetSelfIp(ifnames...)
	for _, v := range all {
		ipA := strings.Split(v, ".")[0]
		if ipA == "10" || ipA == "172" || ipA == "192" || v == "127.0.0.1" {
			ips = append(ips, v)
		}
	}

	return
}

func GetSelfExtraIp(ifnames ...string) (ips []string) {
	all := GetSelfIp(ifnames...)
	for _, v := range all {
		ipA := strings.Split(v, ".")[0]
		if ipA == "10" || ipA == "172" || ipA == "192" || v == "127.0.0.1" {
			continue
		}
		ips = append(ips, v)
	}

	return
}

func ReadFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, ErrFileRead
	}
	return data, nil
}

func SplitStr(s string, sep string) []string {
	return strings.Split(s, sep)
}

func SplitStrN(s string, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

func StrFind(s string, f string) int {
	return strings.Index(s, f)
}

func ReplaceStr(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func TrimStr(s string) string {
	return strings.TrimSpace(s)
}

func Try(fun func(), handler func(interface{})) {
	defer func() {
		if err := recover(); err != nil {
			if handler == nil {
				LogStack()
				LogError("error catch:%v", err)
			} else {
				handler(err)
			}
		}
	}()
	fun()
}

func ParseBaseKind(kind reflect.Kind, data string) (interface{}, error) {
	switch kind {
	case reflect.String:
		return data, nil
	case reflect.Bool:
		v := data == "1" || data == "true"
		return v, nil
	case reflect.Int:
		x, err := strconv.ParseInt(data, 0, 64)
		return int(x), err
	case reflect.Int8:
		x, err := strconv.ParseInt(data, 0, 8)
		return int8(x), err
	case reflect.Int16:
		x, err := strconv.ParseInt(data, 0, 16)
		return int16(x), err
	case reflect.Int32:
		x, err := strconv.ParseInt(data, 0, 32)
		return int32(x), err
	case reflect.Int64:
		x, err := strconv.ParseInt(data, 0, 64)
		return int64(x), err
	case reflect.Float32:
		x, err := strconv.ParseFloat(data, 32)
		return float32(x), err
	case reflect.Float64:
		x, err := strconv.ParseFloat(data, 64)
		return float64(x), err
	case reflect.Uint:
		x, err := strconv.ParseUint(data, 10, 64)
		return uint(x), err
	case reflect.Uint8:
		x, err := strconv.ParseUint(data, 10, 8)
		return uint8(x), err
	case reflect.Uint16:
		x, err := strconv.ParseUint(data, 10, 16)
		return uint16(x), err
	case reflect.Uint32:
		x, err := strconv.ParseUint(data, 10, 32)
		return uint32(x), err
	case reflect.Uint64:
		x, err := strconv.ParseUint(data, 10, 64)
		return uint64(x), err
	default:
		LogError("parse failed type not found type:%v data:%v", kind, data)
		return nil, errors.New("type not found")
	}
}
