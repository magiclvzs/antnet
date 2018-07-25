package antnet

import (
	"encoding/csv"
	"os"
	"reflect"
	"strings"
)

type GenConfigObj struct {
	GenObjFun   func() interface{}
	ParseObjFun map[reflect.Kind]func(fieldv reflect.Value, data, path string) error
}

var csvParseMap = map[reflect.Kind]func(fieldv reflect.Value, data, path string) error{}

func SetCSVParseFunc(kind reflect.Kind, fun func(fieldv reflect.Value, data, path string) error) {
	csvParseMap[kind] = fun
}

func GetCSVParseFunc(kind reflect.Kind) func(fieldv reflect.Value, data, path string) error {
	return csvParseMap[kind]
}

func setValue(fieldv reflect.Value, item, data, path string, line int, f *GenConfigObj) error {
	pm := csvParseMap
	if f.ParseObjFun != nil {
		pm = f.ParseObjFun
	}

	if fun, ok := pm[fieldv.Kind()]; ok {
		err := fun(fieldv, data, path)
		if err != nil {
			LogError("csv read error path:%v line:%v err:%v field:%v", path, line, err, item)
		}
		return err
	} else {
		v, err := ParseBaseKind(fieldv.Kind(), data)
		if err != nil {
			LogError("csv read error path:%v line:%v err:%v field:%v", path, line, err, item)
			return err
		}
		fieldv.Set(reflect.ValueOf(v))
	}

	return nil
}

/*
	path 文件路径
	nindex key值行号，从1开始
	dataBegin 数据开始行号，从1开始
	f 对象产生器 json:"-" tag字段会被跳过
*/
func ReadConfigFromCSV(path string, nindex int, dataBegin int, f *GenConfigObj) (error, []interface{}) {
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

	typ := reflect.ValueOf(f.GenObjFun()).Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		fieldt := typ.FieldByIndex([]int{i})
		name := fieldt.Name
		if v, ok := csv_nimap[name]; ok {
			nimap[name] = v
		} else if fieldt.Tag.Get("json") != "-" {
			LogError("config index not found path:%s name:%s", path, name)
			return ErrCSVParse, nil
		}
	}

	for i := dataBegin - 1; i < len(csvdata); i++ {
		obj := f.GenObjFun()
		objv := reflect.ValueOf(obj)
		obje := objv.Elem()
		for k, v := range nimap {
			switch obje.FieldByName(k).Kind() {
			case reflect.Ptr:
				err = setValue(obje.FieldByName(k).Elem(), k, strings.TrimSpace(csvdata[i][v]), path, i+1, f)
				if err != nil {
					return err, nil
				}
			default:
				err = setValue(obje.FieldByName(k), k, strings.TrimSpace(csvdata[i][v]), path, i+1, f)
				if err != nil {
					return err, nil
				}
			}
		}
		dataObj = append(dataObj, obj)
	}

	return nil, dataObj
}

/*读取csv字段+值，竖着处理
  [in] path       文件路径
  [in] keyIndex   需要读取的字段列号
  [in] valueIndex 需要读取的字段数据列号
  [in] dataBegin  从哪一行开始输出
*/
func ReadConfigFromCSVLie(path string, keyIndex int, valueIndex int, dataBegin int, f *GenConfigObj) (error, interface{}) {
	fi, err := os.Open(path)
	if err != nil {
		return err, nil
	}

	csvdata, err := csv.NewReader(fi).ReadAll()
	if err != nil {
		return err, nil
	}

	obj := f.GenObjFun()
	robj := reflect.Indirect(reflect.ValueOf(obj))
	for i := dataBegin - 1; i < len(csvdata); i++ {
		name := csvdata[i][keyIndex-1]
		bname := []byte(name)
		bname[0] = byte(int(bname[0]) & ^32)
		err = setValue(robj.FieldByName(string(bname)), string(bname), strings.TrimSpace(csvdata[i][valueIndex-1]), path, i+1, f)
		if err != nil {
			return err, nil
		}
	}

	return nil, obj
}
