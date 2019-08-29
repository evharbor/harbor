package convert

import (
	"errors"
	"reflect"
)

// ToFloat try convert interface to float64.
func ToFloat(value interface{}) (val float64, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New("can not convert to as type float")
			return
		}
	}()

	val = reflect.ValueOf(value).Float()
	return
}

// ToInt try convert interface to int
func ToInt(value interface{}) (val int64, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New("can not convert to as type int")
			return
		}
	}()

	val = reflect.ValueOf(value).Int()
	return
}

// ToUint try convert interface to uint
func ToUint(value interface{}) (val uint64, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New("can not convert to as type uint")
			return
		}
	}()

	val = reflect.ValueOf(value).Uint()
	return
}

// MaxInt return the largest argument
func MaxInt(a, b int64) int64 {

	if a > b {
		return a
	}
	return b
}

// MaxUint return the largest argument
func MaxUint(a, b uint64) uint64 {

	if a > b {
		return a
	}
	return b
}

// MinInt return the smallest argument
func MinInt(a, b int64) int64 {

	if a < b {
		return a
	}
	return b
}

// MinUint return the smallest argument
func MinUint(a, b uint64) uint64 {

	if a < b {
		return a
	}
	return b
}
