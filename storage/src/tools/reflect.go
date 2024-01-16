package tools

import (
	"encoding/json"
	"fmt"
	"github.com/finishy1995/go-library/storage/core"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

const (
	VersionMark  = "Version"
	TagHashMark  = "hash"
	TagRangeMark = "range"
)

func GetSliceStructName(value interface{}) string {
	tp := reflect.TypeOf(value)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	if tp.Kind() != reflect.Slice {
		return ""
	}
	tp = tp.Elem()
	if tp.Kind() != reflect.Struct {
		return ""
	}
	return tp.Name()
}

func GetStructName(value interface{}) string {
	tp := reflect.TypeOf(value)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	if tp.Kind() != reflect.Struct {
		return ""
	}
	return tp.Name()
}

func GetStructOnlyName(value interface{}) string {
	if reflect.TypeOf(value).Kind() != reflect.Struct {
		return ""
	}
	return reflect.TypeOf(value).Name()
}

func GetHashAndRangeKey(value interface{}, useTag bool) (hashKey string, rangeKey string) {
	// 类型检查
	tp := reflect.TypeOf(value)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	k := tp.Kind()
	if k == reflect.Slice {
		tp = tp.Elem()
		k = tp.Kind()
	}
	if k != reflect.Struct {
		return
	}

	for i := 0; i < tp.NumField(); i++ {
		fieldType := tp.Field(i)
		tag := fieldType.Tag.Get("dynamo")
		tagArr := strings.Split(tag, ",")
		name := LowerAllChar(fieldType.Name)
		if useTag {
			if len(tagArr) > 0 && tagArr[0] != "" {
				name = tagArr[0]
			}
		}
		for j := 1; j < len(tagArr); j++ {
			if tagArr[j] == TagHashMark {
				hashKey = name
				continue
			}
			if tagArr[j] == TagRangeMark {
				rangeKey = name
			}
		}
	}
	return
}

// GetFieldInfo 获取结构体中除了特定标签外的所有字段的名称和值
func GetFieldInfo(value interface{}) map[string]interface{} {
	fieldsInfo := make(map[string]interface{})

	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	tp := val.Type()
	if val.Kind() != reflect.Struct {
		fmt.Println("Provided value is not a struct or a pointer to struct")
		return fieldsInfo
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := tp.Field(i)

		tag := fieldType.Tag.Get("dynamo")
		if tag == "" || (!strings.Contains(tag, TagHashMark) && !strings.Contains(tag, TagRangeMark)) {
			fieldsInfo[LowerAllChar(fieldType.Name)] = field.Interface()
		}
	}

	return fieldsInfo
}

func GetHashAndRangeValue(value interface{}) (hashValue interface{}, rangeValue interface{}) {
	val := reflect.ValueOf(value)
	// 确保我们在处理指向结构体的指针时正确地解引用
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}
	// 类型检查
	tp := reflect.TypeOf(value)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	k := tp.Kind()
	if k == reflect.Slice {
		tp = tp.Elem()
		k = tp.Kind()
	}
	if k != reflect.Struct {
		return
	}

	for i := 0; i < tp.NumField(); i++ {
		fieldType := tp.Field(i)
		tag := fieldType.Tag.Get("dynamo")
		tagArr := strings.Split(tag, ",")
		for j := 1; j < len(tagArr); j++ {
			if tagArr[j] == TagHashMark {
				hashValue = val.Field(i).Interface()
				continue
			}
			if tagArr[j] == TagRangeMark {
				rangeValue = val.Field(i).Interface()
			}
		}
	}
	return
}

// lowerFirstChar 将字符串的首字母转换为小写
func lowerFirstChar(s string) string {
	if s == "" {
		return s
	}

	// 将字符串转换为rune切片以支持多字节字符
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])

	return string(runes)
}

func LowerAllChar(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s)
}

func GetPointer(value interface{}) interface{} {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		// 如果不是指针，则创建一个指向它的指针
		valPtr := reflect.New(reflect.TypeOf(value))
		valPtr.Elem().Set(val)
		return valPtr.Interface()
	}
	return value // 已经是指针
}

func TrySetStructDefaultValue(value interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return core.ErrUnsupportedValueType
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() || !isZero(field) {
			continue
		}

		tag := val.Type().Field(i).Tag.Get("dynamo")
		if defaultVal := parseDefaultTag(tag); defaultVal != "" {
			if err := setDefaultValue(field, defaultVal); err != nil {
				return err
			}
		}
	}
	return nil
}

// isNonZero checks if the field is non-zero (has been set to a value other than the zero value for its type).
func isZero(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface:
		return field.IsNil()

	default:
		return field.IsZero()
	}
}

func parseDefaultTag(tag string) string {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		if strings.HasPrefix(part, "default=") {
			return strings.TrimPrefix(part, "default=")
		}
	}
	return ""
}

func setDefaultValue(field reflect.Value, defaultVal string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(defaultVal, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(defaultVal, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(defaultVal, 64)
		if err != nil {
			return err
		}
		field.SetFloat(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(defaultVal)
		if err != nil {
			return err
		}
		field.SetBool(val)
	case reflect.String:
		field.SetString(defaultVal)
	default:
		return fmt.Errorf("unsupported default set field, type: %s", field.Kind())
	}
	return nil
}

func TrySetStructVersion(value interface{}) (uint64, error) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	} else {
		return 0, core.ErrUnsupportedValueType
	}
	if val.Kind() != reflect.Struct {
		return 0, core.ErrUnsupportedValueType
	}

	field := val.FieldByName(VersionMark)
	if field.Kind() != reflect.Invalid && field.CanInterface() {
		fieldInterface := field.Interface()
		if fieldInterface != nil {
			if version, ok := fieldInterface.(uint64); ok {
				field.SetUint(version + 1)
				return version, nil
			}
		}
	}
	return 0, core.ErrUnsupportedValueType
}

func GetFieldValueByName(value interface{}, name string) interface{} {
	tp := reflect.ValueOf(value)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	if tp.Kind() != reflect.Struct {
		return nil
	}

	field := tp.FieldByName(name)
	if field.Kind() != reflect.Invalid && field.CanInterface() {
		return field.Interface()
	}
	return nil
}

func GetFieldValueByRealName(value interface{}, name string) interface{} {
	tp := reflect.ValueOf(value)
	kp := reflect.TypeOf(value)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
		kp = kp.Elem()
	}
	if tp.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < kp.NumField(); i++ {
		fieldType := kp.Field(i)
		tag := fieldType.Tag.Get("dynamo")
		tagArr := strings.Split(tag, ",")
		realName := fieldType.Name
		if len(tagArr) > 0 && tagArr[0] != "" {
			realName = tagArr[0]
		}
		if realName == name {
			field := tp.Field(i)
			if field.Kind() != reflect.Invalid && field.CanInterface() {
				return field.Interface()
			}
		}
	}

	return nil
}

func DeepCopy(source interface{}, target interface{}) error {
	b, err := json.Marshal(source)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, target)
	return err
}

func GetSliceFromInterfacePtr(value interface{}) []interface{} {
	tp := reflect.TypeOf(value).Elem().Kind()
	if tp == reflect.Slice {
		s := reflect.ValueOf(value).Elem()
		temp := make([]interface{}, 0, 0)
		for i := 0; i < s.Len(); i++ {
			temp = append(temp, s.Index(i).Interface())
		}
		return temp
	}
	return nil
}

func GetInterfacePtr(value interface{}) interface{} {
	vp := reflect.ValueOf(value)
	if vp.Kind() == reflect.Ptr {
		return value
	} else {
		vNew := reflect.New(vp.Type())
		vNew.Elem().Set(vp)
		return vNew.Interface()
	}
}

// GenerateKeyMapFromStructSlice
// sl为StorageModel数组 []Storage
func GenerateKeyMapFromStructSlice(sl interface{}) map[interface{}]uint8 {
	var res = make(map[interface{}]uint8, 0)
	realVal := reflect.ValueOf(sl)
	valLen := realVal.Len()
	hashName := GetHashName(realVal.Interface())
	for i := 0; i < valLen; i++ {
		key := realVal.Index(i).FieldByName(hashName).Interface()
		res[key] = 0
	}
	return res
}

// GetHashName 获取主键在结构体中的名字
func GetHashName(model interface{}) string {
	vStruct := reflect.TypeOf(model).Elem()
	filedNum := vStruct.NumField()
	for i := 0; i < filedNum; i++ {
		filedTag := vStruct.Field(i).Tag.Get("dynamo")
		tagSlice := strings.Split(filedTag, ",")
		for _, v := range tagSlice {
			if v == "hash" {
				return vStruct.Field(i).Name
			}
		}
	}
	return ""
}

// GetStructVersionFromOriginData 获取版本
func GetStructVersionFromOriginData(value interface{}) (uint64, error) {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Struct {
		return 0, core.ErrUnsupportedValueType
	}

	field := val.FieldByName(VersionMark)
	if field.Kind() != reflect.Invalid && field.CanInterface() {
		fieldInterface := field.Interface()
		if fieldInterface != nil {
			if version, ok := fieldInterface.(uint64); ok {
				return version, nil
			}
		}
	}
	return 0, core.ErrUnsupportedValueType
}
