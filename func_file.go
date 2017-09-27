package antnet

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func PathBase(p string) string {
	return path.Base(p)
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
