// Package jsonstream writes ordinary JSON documents a field/array entry at a
// time. It uses encoding/json for values, escaping, numbers and map ordering.
package jsonstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
)

type ValueWriter func(io.Writer, bool, string) error

var indentBuffers = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func put(w io.Writer, s string) error {
	n, err := io.WriteString(w, s)
	if err == nil && n != len(s) {
		err = io.ErrShortWrite
	}
	return err
}

// Value's prefix has the same meaning as json.MarshalIndent's prefix.
func Value(w io.Writer, value any, pretty bool, prefix string) error {
	var data []byte
	var err error
	data, err = json.Marshal(value)
	if err != nil {
		return err
	}
	if pretty {
		buffer := indentBuffers.Get().(*bytes.Buffer)
		buffer.Reset()
		defer func() {
			if buffer.Cap() <= 256*1024 {
				indentBuffers.Put(buffer)
			}
		}()
		if err := json.Indent(buffer, data, prefix, "  "); err != nil {
			return err
		}
		data = buffer.Bytes()
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

func Array(w io.Writer, count int, nilSlice bool, pretty bool, prefix string, item func(int) any) error {
	if nilSlice {
		return put(w, "null")
	}
	if err := put(w, "["); err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if i > 0 {
			if err := put(w, ","); err != nil {
				return err
			}
		}
		if pretty {
			if err := put(w, "\n"+prefix+"  "); err != nil {
				return err
			}
		}
		if err := Value(w, item(i), pretty, prefix+"  "); err != nil {
			return err
		}
	}
	if pretty && count > 0 {
		if err := put(w, "\n"+prefix); err != nil {
			return err
		}
	}
	return put(w, "]")
}

func empty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Interface, reflect.Pointer:
		return v.IsZero()
	}
	return false
}

// Object supports the exported, nonembedded fields used by canonical Brain and
// checkpoint envelopes. Overrides replace a field's value, never its presence
// or order. Neither Object nor Array retains the entire encoded document.
func Object(w io.Writer, value any, pretty bool, prefix string, overrides map[string]ValueWriter) error {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return fmt.Errorf("jsonstream: expected struct")
	}
	if err := put(w, "{"); err != nil {
		return err
	}
	count := 0
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			return fmt.Errorf("jsonstream: embedded field %s is unsupported", field.Name)
		}
		tag := strings.Split(field.Tag.Get("json"), ",")
		name := tag[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		omit := false
		for _, option := range tag[1:] {
			if option == "omitempty" {
				omit = true
			} else if option != "" {
				return fmt.Errorf("jsonstream: unsupported tag %q", option)
			}
		}
		item := v.Field(i)
		if omit && empty(item) {
			continue
		}
		if count > 0 {
			if err := put(w, ","); err != nil {
				return err
			}
		}
		base := ""
		if pretty {
			base = prefix + "  "
			if err := put(w, "\n"+base); err != nil {
				return err
			}
		}
		key, _ := json.Marshal(name)
		colon := ":"
		if pretty {
			colon = ": "
		}
		if err := put(w, string(key)+colon); err != nil {
			return err
		}
		if override := overrides[name]; override != nil {
			if err := override(w, pretty, base); err != nil {
				return err
			}
		} else if item.Kind() == reflect.Slice && item.Type() != reflect.TypeOf(json.RawMessage{}) && item.Type().Elem().Kind() != reflect.Uint8 {
			if err := Array(w, item.Len(), item.IsNil(), pretty, base, func(index int) any { return item.Index(index).Interface() }); err != nil {
				return err
			}
		} else if err := Value(w, item.Interface(), pretty, base); err != nil {
			return err
		}
		count++
	}
	if pretty && count > 0 {
		if err := put(w, "\n"+prefix); err != nil {
			return err
		}
	}
	return put(w, "}")
}
