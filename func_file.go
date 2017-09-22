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

func ReadFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, ErrFileRead
	}
	return data, nil
}

func WriteFile(path string, data []byte) {
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
