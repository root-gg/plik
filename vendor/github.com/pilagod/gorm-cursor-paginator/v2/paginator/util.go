package paginator

import "reflect"

func reverse(elems reflect.Value) reflect.Value {
	result := reflect.MakeSlice(elems.Type(), 0, elems.Cap())
	for i := elems.Len() - 1; i >= 0; i-- {
		result = reflect.Append(result, elems.Index(i))
	}
	return result
}
