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

func setValue(fieldv reflect.Value, item, data, path string, f *GenConfigObj) error {
	pm := csvParseMap
	if f.ParseObjFun != nil {
		pm = f.ParseObjFun
	}

	if fun, ok := pm[fieldv.Kind()]; ok {
		return fun(fieldv, data, path)
	} else {
		v, err := ParseBaseKind(fieldv.Kind(), data)
		if err != nil {
			LogError("csv read error path:%v err:%v field:%v", path, err, item)
			return err
		}
		fieldv.Set(reflect.ValueOf(v))
	}

	return nil
}

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
		stringTemp := ""
		stringArry := strings.Split(name, "_")
		for _, v := range stringArry {
			bname := []byte(v)
			if (47 < bname[0]) && (bname[0] < 58) { //避開"_+數字"的配置字段
				stringTemp += "_"
			} else {
				bname[0] = byte(int(bname[0]) & ^32)
			}
			stringTemp += string(bname)
		}
		csv_nimap[stringTemp] = index
	}

	typ := reflect.ValueOf(f.GenObjFun()).Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.FieldByIndex([]int{i}).Name
		if v, ok := csv_nimap[name]; ok {
			nimap[name] = v
		} else if name != "XXX_unrecognized" { //由于生成pb结构多了一个XXX_unrecognized成员,此成员不参与异常字段判断
			LogInfo("config index not found file:%s name:%s\n", path, name)
		}
	}

	for i := dataBegin - 1; i < len(csvdata); i++ {
		obj := f.GenObjFun()
		objv := reflect.ValueOf(obj)
		obje := objv.Elem()
		for k, v := range nimap {
			switch obje.FieldByName(k).Kind() {
			case reflect.Ptr:
				setValue(obje.FieldByName(k).Elem(), k, strings.TrimSpace(csvdata[i][v]), path, f)
			default:
				setValue(obje.FieldByName(k), k, strings.TrimSpace(csvdata[i][v]), path, f)
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
		setValue(robj.FieldByName(string(bname)), string(bname), strings.TrimSpace(csvdata[i][valueIndex-1]), path, f)
	}

	return nil, obj
}
