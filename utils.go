package errors

import "reflect"

func isNil(err error) bool {
	if err == nil {
		return true
	}
	switch reflect.TypeOf(err).Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if reflect.ValueOf(err).IsNil() {
			return true
		}
	default:
	}
	return false
}
