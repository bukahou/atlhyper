package contract

import (
	"fmt"
	"reflect"
	"time"
)

// FixedTime 填充时间字段用的固定时刻
var FixedTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// Fill 把 ptr 指向的值里每个可导出字段都填上非零值，让响应里每个属性都出现、可被校验：
// 字符串 "x"、数值 1、布尔 true、时间取 FixedTime、切片与映射各放一个元素、指针分配。
// 具名字符串类型（枚举）用 samples 里同类型的值；缺样例时 panic，提示补上。
func Fill(ptr any, samples ...any) {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic("contract.Fill 需要非 nil 指针")
	}
	bySample := map[reflect.Type]reflect.Value{}
	for _, s := range samples {
		bySample[reflect.TypeOf(s)] = reflect.ValueOf(s)
	}
	fill(v.Elem(), bySample, 0)
}

const maxFillDepth = 12

func fill(v reflect.Value, samples map[reflect.Type]reflect.Value, depth int) {
	if depth > maxFillDepth || !v.CanSet() {
		return
	}
	if s, ok := samples[v.Type()]; ok {
		v.Set(s)
		return
	}
	if v.Type() == reflect.TypeOf(time.Time{}) {
		v.Set(reflect.ValueOf(FixedTime))
		return
	}
	switch v.Kind() {
	case reflect.String:
		if v.Type().PkgPath() != "" {
			panic(fmt.Sprintf("contract.Fill: 具名字符串类型 %s 需要一个样例值（多半是枚举）", v.Type()))
		}
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1.5)
	case reflect.Ptr:
		elem := reflect.New(v.Type().Elem())
		fill(elem.Elem(), samples, depth+1)
		v.Set(elem)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fill(s.Index(0), samples, depth+1)
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMapWithSize(v.Type(), 1)
		key := reflect.New(v.Type().Key()).Elem()
		fill(key, samples, depth+1)
		val := reflect.New(v.Type().Elem()).Elem()
		fill(val, samples, depth+1)
		m.SetMapIndex(key, val)
		v.Set(m)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fill(v.Field(i), samples, depth+1)
			}
		}
	case reflect.Interface:
		v.Set(reflect.ValueOf("x"))
	}
}
