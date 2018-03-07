package antnet

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
)

func PathBase(p string) string {
	return path.Base(p)
}

func PathAbs(p string) string {
	f, err := filepath.Abs(p)
	if err != nil {
		LogError("get abs path failed path:%v err:%v", p, err)
		return ""
	}
	return f
}

func GetEXEDir() string {
	return PathDir(GetEXEPath())
}

func GetEXEPath() string {
	return PathAbs(os.Args[0])
}

func GetExeName() string {
	return PathBase(GetEXEPath())
}

func GetExeDir() string {
	return PathDir(GetEXEPath())
}

func GetExePath() string {
	return PathAbs(os.Args[0])
}

func GetEXEName() string {
	return PathBase(GetEXEPath())
}

func PathDir(p string) string {
	return path.Dir(p)
}

func PathExt(p string) string {
	return path.Ext(p)
}

func PathClean(p string) string {
	return path.Clean(p)
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

func NewDir(path string) error {
	return os.MkdirAll(path, 0777)
}

func ReadFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		LogError("read file filed path:%v err:%v", path, err)
		return nil, ErrFileRead
	}
	return data, nil
}

func WriteFile(path string, data []byte) {
	dir := PathDir(path)
	if !PathExists(dir) {
		NewDir(dir)
	}
	ioutil.WriteFile(path, data, 0777)
}

func GetFiles(path string) []string {
	files := []string{}
	filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

func DelFile(path string) {
	os.Remove(path)
}

func DelDir(path string) {
	os.RemoveAll(path)
}

func CreateFile(path string) (*os.File, error) {
	dir := PathDir(path)
	if !PathExists(dir) {
		NewDir(dir)
	}
	return os.Create(path)
}

func CopyFile(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)
}

func walkDirTrue(dir string, wg *sync.WaitGroup, fun func(dir string, info os.FileInfo)) {
	wg.Add(1)
	defer wg.Done()
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		LogError("walk dir failed dir:%v err:%v", dir, err)
		return
	}
	for _, info := range infos {
		if info.IsDir() {
			fun(dir, info)
			subDir := filepath.Join(dir, info.Name())
			go walkDirTrue(subDir, wg, fun)
		} else {
			fun(dir, info)
		}
	}
}

func WalkDir(dir string, fun func(dir string, info os.FileInfo)) {
	if fun == nil {
		return
	}
	wg := &sync.WaitGroup{}
	walkDirTrue(dir, wg, fun)
	wg.Wait()
}

func FileCount(dir string) int32 {
	var count int32 = 0
	WalkDir(dir, func(dir string, info os.FileInfo) {
		if !info.IsDir() {
			atomic.AddInt32(&count, 1)
		}
	})
	return count
}

func DirCount(dir string) int32 {
	var count int32 = 0
	WalkDir(dir, func(dir string, info os.FileInfo) {
		if info.IsDir() {
			atomic.AddInt32(&count, 1)
		}
	})
	return count
}

func DirSize(dir string) int64 {
	var size int64 = 0
	WalkDir(dir, func(dir string, info os.FileInfo) {
		if !info.IsDir() {
			atomic.AddInt64(&size, info.Size())
		}
	})
	return size
}
