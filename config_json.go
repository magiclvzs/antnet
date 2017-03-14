package antnet

import (
	"encoding/json"
	"io/ioutil"
)

func ReadConfigFromJson(path string, v interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return ErrFileRead
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return ErrJsonUnPack
	}
	return nil
}
