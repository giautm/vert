//go:build js && wasm

package vert

import (
	"reflect"
	"strings"
	"syscall/js"
)

var (
	null   = js.ValueOf(nil)
	object = js.Global().Get("Object")
	array  = js.Global().Get("Array")
)

// ValueOf returns the Go value as a new value.
func ValueOf(i any) js.Value {
	return valueOf(reflect.ValueOf(i))
}

var (
	kindToType = map[reflect.Kind]reflect.Type{
		reflect.String:     reflect.TypeOf(""),
		reflect.Int:        reflect.TypeOf(int(0)),
		reflect.Int8:       reflect.TypeOf(int8(0)),
		reflect.Int16:      reflect.TypeOf(int16(0)),
		reflect.Int32:      reflect.TypeOf(int32(0)),
		reflect.Int64:      reflect.TypeOf(int64(0)),
		reflect.Uint:       reflect.TypeOf(uint(0)),
		reflect.Uint8:      reflect.TypeOf(uint8(0)),
		reflect.Uint16:     reflect.TypeOf(uint16(0)),
		reflect.Uint32:     reflect.TypeOf(uint32(0)),
		reflect.Uint64:     reflect.TypeOf(uint64(0)),
		reflect.Bool:       reflect.TypeOf(false),
		reflect.Float32:    reflect.TypeOf(float32(0)),
		reflect.Float64:    reflect.TypeOf(float64(0)),
		reflect.Complex64:  reflect.TypeOf(complex64(0)),
		reflect.Complex128: reflect.TypeOf(complex128(0)),
	}
)

// valueOf recursively returns a new value.
func valueOf(v reflect.Value) js.Value {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return valueOfPointerOrInterface(v)
	case reflect.Slice, reflect.Array:
		return valueOfSliceOrArray(v)
	case reflect.Map:
		return valueOfMap(v)
	case reflect.Struct:
		if v.Type() == jsValue {
			return v.Interface().(js.Value)
		}
		return valueOfStruct(v)
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return js.ValueOf(v.Convert(kindToType[v.Kind()]).Interface())
	default:
		if v.IsValid() {
			return js.ValueOf(v.Interface())
		}
		return null
	}
}

// valueOfPointerOrInterface returns a new value.
func valueOfPointerOrInterface(v reflect.Value) js.Value {
	if v.IsNil() {
		return null
	}
	return valueOf(v.Elem())
}

// valueOfSliceOrArray returns a new array object value.
func valueOfSliceOrArray(v reflect.Value) js.Value {
	if v.IsNil() {
		return null
	}
	a := array.New()
	for i := 0; i < v.Len(); i++ {
		e := v.Index(i)
		a.SetIndex(i, valueOf(e))
	}
	return a
}

// valueOfMap returns a new object value.
// Map keys must be of type string.
func valueOfMap(v reflect.Value) js.Value {
	if v.IsNil() {
		return null
	}
	m := object.New()
	for it := v.MapRange(); it.Next(); {
		// Support named string key types by converting to plain string.
		k := it.Key().Convert(reflect.TypeOf("")).String()
		m.Set(k, valueOf(it.Value()))
	}
	return m
}

// valueOfStruct returns a new object value.
func valueOfStruct(v reflect.Value) js.Value {
	t, s := v.Type(), object.New()
	for i := 0; i < v.NumField(); i++ {
		if f := v.Field(i); f.CanInterface() {
			// Inline embedded struct or *struct fields by merging their properties,
			// unless a tag name is provided (then use that name as a nested property).
			sf := t.Field(i)
			k := nameOf(sf)
			if !sf.Anonymous {
				s.Set(k, valueOf(f))
				continue
			}
			ft := f.Type()
			if ft.Kind() == reflect.Pointer {
				if f.IsNil() {
					continue
				}
				ft = ft.Elem()
			}
			if ft.Kind() != reflect.Struct {
				continue
			}
			// If the field has a tag-provided name, use it as a nested property.
			if k != sf.Name {
				s.Set(k, valueOf(f))
				continue
			}
			// Otherwise, merge the embedded struct or *struct properties.
			if ov := valueOf(f); !ov.IsNull() && !ov.IsUndefined() {
				keys := object.Call("keys", ov)
				for j := 0; j < keys.Length(); j++ {
					k := keys.Index(j).String()
					s.Set(k, ov.Get(k))
				}
			}
		}
	}
	return s
}

// nameOf returns the JS tag name, otherwise the field name.
func nameOf(sf reflect.StructField) string {
	if n := sf.Tag.Get("js"); n != "" {
		return n
	}
	if n, _, _ := strings.Cut(sf.Tag.Get("json"), ","); n != "" {
		return n
	}
	return sf.Name
}
