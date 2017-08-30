package antnet

import (
	"encoding/csv"
	"os"
	"proto/pb"
	"reflect"
	"strings"
)

type GenConfigObject func() interface{}

func setValue(fieldv reflect.Value, data, path string) error {
	if fieldv.Kind() == reflect.Struct {
		if data == "0" {
			return nil
		} else {
			stringArry := strings.Split(data, "_")
			if len(stringArry) == (fieldv.NumField() - 1) { //NumField - 1是由于生成pb结构多了一个XXX_unrecognized成员
				for i := 0; i < (fieldv.NumField() - 1); i++ {
					if fieldv.Field(i).Kind() == reflect.Ptr {
						v, err := ParseBaseKind(fieldv.Field(i).Elem().Kind(), stringArry[i])
						if err != nil {
							return err
						}
						fieldv.Field(i).Elem().Set(reflect.ValueOf(v))
					} else {
						v, err := ParseBaseKind(fieldv.Field(i).Kind(), stringArry[i])
						if err != nil {
							return err
						}
						fieldv.Field(i).Set(reflect.ValueOf(v))
					}
				}
			} else {
				LogInfo("path:%v split string is err, stringArry len:%v fieldv.NumField:%v\n", path, len(stringArry), fieldv.NumField())
			}
		}
	} else if fieldv.Kind() == reflect.Slice {
		Try(func() {
			s := []int32{}
			isitem := false
			item := []*pb.ItemConfig{}
			for _, v := range strings.Split(data, "*") {
				if strings.Contains(v, "_") {
					arr := strings.Split(v, "_")
					isitem = true
					item = append(item, &pb.ItemConfig{
						Id:     pb.Int32(int32(Atoi(arr[0]))),
						Number: pb.Int32(int32(Atoi(arr[1])))})
				} else {
					s = append(s, int32(Atoi(v)))
				}
			}
			if isitem {
				fieldv.Set(reflect.ValueOf(item))
			} else {
				fieldv.Set(reflect.ValueOf(s))
			}
		}, func(err interface{}) {
			s := []string{}
			for _, v := range strings.Split(data, "*") {
				s = append(s, v)
			}
			fieldv.Set(reflect.ValueOf(s))
		})
	} else {
		v, err := ParseBaseKind(fieldv.Kind(), data)
		if err != nil {
			return err
		}
		fieldv.Set(reflect.ValueOf(v))
	}

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

	typ := reflect.ValueOf(f()).Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.FieldByIndex([]int{i}).Name
		if v, ok := csv_nimap[name]; ok {
			nimap[name] = v
		} else if name != "XXX_unrecognized" { //由于生成pb结构多了一个XXX_unrecognized成员,此成员不参与异常字段判断
			LogInfo("config index not found file:%s name:%s\n", path, name)
		}
	}

	for i := dataBegin - 1; i < len(csvdata); i++ {
		obj := f()
		objv := reflect.ValueOf(obj)
		obje := objv.Elem()
		for k, v := range nimap {
			switch obje.FieldByName(k).Kind() {
			case reflect.Ptr:
				setValue(obje.FieldByName(k).Elem(), strings.TrimSpace(csvdata[i][v]), path)
			default:
				setValue(obje.FieldByName(k), strings.TrimSpace(csvdata[i][v]), path)
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
func ReadConfigFromCSVLie(path string, keyIndex int, valueIndex int, dataBegin int, f GenConfigObject) (error, interface{}) {
	fi, err := os.Open(path)
	if err != nil {
		return err, nil
	}

	csvdata, err := csv.NewReader(fi).ReadAll()
	if err != nil {
		return err, nil
	}

	obj := f()
	robj := reflect.Indirect(reflect.ValueOf(obj))
	for i := dataBegin - 1; i < len(csvdata); i++ {
		name := csvdata[i][keyIndex-1]
		bname := []byte(name)
		bname[0] = byte(int(bname[0]) & ^32)
		setValue(robj.FieldByName(string(bname)), strings.TrimSpace(csvdata[i][valueIndex-1]), path)
	}

	return nil, obj
}
