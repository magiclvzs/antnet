package antnet

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ZlibCompress(data []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(data)
	w.Close()
	return in.Bytes()
}

func ZlibUnCompress(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	r, _ := zlib.NewReader(b)
	defer r.Close()
	undatas, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return undatas, nil
}

func GZipCompress(data []byte) []byte {
	var in bytes.Buffer
	w := gzip.NewWriter(&in)
	w.Write(data)
	w.Close()
	return in.Bytes()
}

func GZipUnCompress(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	r, _ := gzip.NewReader(b)
	defer r.Close()
	undatas, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return undatas, nil
}

func Zip(srcFile string, destZip string) error {
	zipfile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(srcFile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}


		header.Name = strings.TrimPrefix(path, filepath.Dir(srcFile) + "/")
		// header.Name = path
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if ! info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})

	return err
}

func Unzip(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer inFile.Close()

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}