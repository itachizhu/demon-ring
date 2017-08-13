package lang

import (
	"reflect"
	"errors"
	"strconv"
)

const INDEX_NOT_FOUND = -1

func Copy(source []interface{}) []interface{} {
	if IsEmpty(source) {
		return nil
	}
	dst := make([]interface{}, len(source))
	copy(dst, source)
	return dst
}

func Add(source []interface{}, elem ...interface{}) []interface{} {
	m := len(source)
	n := len(elem)
	if m + n == 0 {
		return nil
	}
	dst := []interface{}(nil)
	if m > 0 {
		dst = append(dst, source...)
	}
	if n > 0 {
		dst = append(dst, elem...)
	}
	return dst
}

func IsEmpty(source []interface{}) bool {
	return len(source) == 0
}

func IsNotEmpty(source []interface{}) bool {
	return !IsEmpty(source)
}

func IndexOf(source []interface{}, elem interface{}, startIndex int) int {
	if IsEmpty(source) {
		return INDEX_NOT_FOUND
	}
	if startIndex < 0 {
		startIndex = 0
	}
	for i := startIndex; i < len(source); i++ {
		if reflect.DeepEqual(source[i], elem) {
			return i
		}
	}
	return INDEX_NOT_FOUND
}

func LastIndexOf(source []interface{}, elem interface{}, endIndex int) int {
	if IsEmpty(source) {
		return INDEX_NOT_FOUND
	}
	if endIndex < 0 {
		return INDEX_NOT_FOUND
	} else if endIndex >= len(source) {
		endIndex = len(source) - 1
	}
	for i := endIndex; i >= 0; i-- {
		if reflect.DeepEqual(source[i], elem) {
			return i
		}
	}
	return INDEX_NOT_FOUND
}

func Contains(source []interface{}, elem interface{}) bool {
	return IndexOf(source, elem, 0) > INDEX_NOT_FOUND
}

func Insert(index int, source []interface{}, elem ...interface{}) []interface{} {
	if IsEmpty(source) {
		return nil
	}
	if len(elem) == 0 {
		return Copy(source)
	}
	if index < 0 || index >= len(source) {
		panic(errors.New("index out of bounds! Index: " + strconv.Itoa(index) + ", Length: " + strconv.Itoa(len(source))))
	}
	dst := append([]interface{}(nil), source[:index])
	dst = append(dst, append([]interface{}{elem}, source[index:]...)...)
	return dst
}

func Remove(source []interface{}, index int) interface{} {
	if IsEmpty(source) {
		return nil
	}
	if index < 0 || index >= len(source) {
		panic(errors.New("index out of bounds! Index: " + strconv.Itoa(index) + ", Length: " + strconv.Itoa(len(source))))
	}
	return append(source[:index],source[index+1:]...)
}