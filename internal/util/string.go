package util

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"
)

func snakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}

	return strings.ToLower(string(data[:]))
}

func GenerateUpdateSql(s interface{}) (str string) {
	v := reflect.ValueOf(s) // 获取字段值
	t := reflect.TypeOf(s)  // 获取字段类型
	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		log.Println("Check type error not Struct")
		return
	}
	// 遍历 struct 中的字段
	for i := 0; i < t.NumField(); i++ {
		if !v.Field(i).IsZero() {
			str += snakeString(t.Field(i).Name) + " = "
			if v.Field(i).Type().String() == "time.Time" {
				t := v.Field(i).Interface().(time.Time)
				str += fmt.Sprintf("\"%v\",", t.Format("2006-01-02 15:04:05"))
			} else if v.Field(i).Kind() != reflect.Int && v.Field(i).Kind() != reflect.Bool {
				str += fmt.Sprintf("\"%v\",", v.Field(i))
			} else {
				str += fmt.Sprintf("%v,", v.Field(i))
			}
		}
	}
	str = str[:len(str)-1]
	fmt.Println(str)
	return
}
