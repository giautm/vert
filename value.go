//go:build js && wasm

package vert

import (
	"reflect"
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
	for i := v.MapRange(); i.Next(); {
		k := i.Key().Interface().(string)
		m.Set(k, valueOf(i.Value()))
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
	if n := sf.Tag.Get("json"); n != "" {
		return n
	}
	return sf.Name
}
