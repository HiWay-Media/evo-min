package dot

import (
	"errors"
	"github.com/getevo/evo-min/lib/reflections"
	"reflect"
	"strings"
)

func Get(obj interface{}, prop string) (interface{}, error) {
	// fmt.Println("getting property")
	// fmt.Println(args)

	// Get the array access
	arr := strings.Split(prop, ".")

	// fmt.Println(arr)
	var err error
	// last, arr := arr[len(arr)-1], arr[:len(arr)-1]
	for _, key := range arr {
		obj, err = getProperty(obj, key)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			return nil, nil
		}
	}
	return obj, nil
}

// Loop through this to get properties via dot notation
func getProperty(obj interface{}, prop string) (interface{}, error) {
	if reflect.TypeOf(obj).Kind() == reflect.Map {

		val := reflect.ValueOf(obj)

		valueOf := val.MapIndex(reflect.ValueOf(prop))

		if valueOf == reflect.Zero(reflect.ValueOf(prop).Type()) {
			return nil, nil
		}

		idx := val.MapIndex(reflect.ValueOf(prop))

		if !idx.IsValid() {
			return nil, nil
		}
		return idx.Interface(), nil
	}

	prop = strings.Title(prop)
	return reflections.GetField(obj, prop)
}

func Set(input interface{}, prop string, value interface{}) error {
	// Get the array access
	arr := strings.Split(prop, ".")
	var val = reflect.ValueOf(input)
	var obj reflect.Value
	if val.Kind() == reflect.Ptr {
		obj = val.Elem()
	} else {
		obj = val
	}
	// fmt.Println(arr)

	last, arr := arr[len(arr)-1], arr[:len(arr)-1]

	for _, key := range arr {
		if obj.Kind() == reflect.Map {
			v := obj.MapIndex(reflect.ValueOf(key))
			if v.IsValid() {
				obj = v
			} else {
				var m = map[string]interface{}{}
				obj.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(m))
				obj = obj.MapIndex(reflect.ValueOf(key))
			}

		} else {
			var ref, err = getProperty(obj.Interface(), key)
			if err != nil {
				return err
			}
			obj = reflect.ValueOf(ref)

		}

	}

	return setProperty(obj.Interface(), last, value)

}

func setProperty(obj interface{}, prop string, val interface{}) error {
	var ref = reflect.ValueOf(obj)
	if ref.Kind() == reflect.Map {
		ref.SetMapIndex(reflect.ValueOf(prop), reflect.ValueOf(val))
		return nil
	}

	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return errors.New("object must be a pointer to a struct")
	}
	prop = strings.Title(prop)

	return reflections.SetField(obj, prop, val)
}
