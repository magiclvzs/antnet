package antnet

import (
	"encoding/csv"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type GenConfigObject func() interface{}

func setValue(fieldv reflect.Value, data string) error {
	var v interface{}

	switch fieldv.Kind() {
	case reflect.Slice:
		v = []byte(data)
	case reflect.String:
		v = data
	case reflect.Bool:
		v = data == "1" || data == "true"
	case reflect.Int:
		x, err := strconv.Atoi(data)
		if err != nil {
			return err
		}
		v = int(x)
	case reflect.Int8:
		x, err := strconv.Atoi(data)
		if err != nil {
			return err
		}
		v = int8(x)
	case reflect.Int16:
		x, err := strconv.Atoi(data)
		if err != nil {
			return err
		}
		v = int16(x)
	case reflect.Int32:
		x, err := strconv.Atoi(data)
		if err != nil {
			return err
		}
		v = int32(x)
	case reflect.Int64:
		x, err := strconv.ParseInt(data, 10, 64)
		if err != nil {
			return err
		}
		v = int64(x)
	case reflect.Float32:
		x, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return err
		}
		v = float32(x)
	case reflect.Float64:
		x, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return err
		}
		v = float64(x)
	case reflect.Uint:
		x, err := strconv.ParseUint(data, 10, 64)
		if err != nil {
			return err
		}
		v = uint(x)
	case reflect.Uint8:
		x, err := strconv.ParseUint(data, 10, 64)
		if err != nil {
			return err
		}
		v = uint8(x)
	case reflect.Uint16:
		x, err := strconv.ParseUint(data, 10, 64)
		if err != nil {
			return err
		}
		v = uint16(x)
	case reflect.Uint32:
		x, err := strconv.ParseUint(data, 10, 64)
		if err != nil {
			return err
		}
		v = uint32(x)
	case reflect.Uint64:
		x, err := strconv.ParseUint(data, 10, 64)
		if err != nil {
			return err
		}
		v = uint64(x)
	default:
		return errors.New("type not found")
	}

	fieldv.Set(reflect.ValueOf(v))
	return nil
}

func ReadConfigFromCSV(path string, nindex int, dataBegin int, f GenConfigObject) (error, []interface{}) {
	csv_nimap := map[string]int{}
	nimap := map[string]int{}
	var dataObj []interface{}

	fi, err := os.Open(path)
	if err != nil {
		return err, nil
	}

	csvdata, err := csv.NewReader(fi).ReadAll()
	if err != nil {
		return err, nil
	}

	dataCount := len(csvdata) - dataBegin + 1
	dataObj = make([]interface{}, 0, dataCount)
	for index, name := range csvdata[nindex-1] {
		if name == "" {
			continue
		}
		bname := []byte(name)
		bname[0] = byte(int(bname[0]) & ^32)
		csv_nimap[string(bname)] = index
	}

	typ := reflect.ValueOf(f()).Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.FieldByIndex([]int{i}).Name
		if v, ok := csv_nimap[name]; ok {
			nimap[name] = v
		} else {
			LogInfo("config index not found file:%s name:%s\n", path, name)
		}
	}

	for i := dataBegin - 1; i < len(csvdata); i++ {
		obj := f()
		robj := reflect.Indirect(reflect.ValueOf(obj))
		for k, v := range nimap {
			setValue(robj.FieldByName(k), strings.TrimSpace(csvdata[i][v]))
		}

		dataObj = append(dataObj, obj)
	}

	return nil, dataObj
}
