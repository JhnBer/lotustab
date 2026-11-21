package driver

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// NormalizeValue converts a driver-specific cell into a JSON-friendly value.
func NormalizeValue(v any) any {
	if v == nil {
		return nil
	}

	switch t := v.(type) {
	case bool, string:
		return t
	case int:
		return int64(t)
	case int8:
		return int64(t)
	case int16:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case uint:
		return uint64(t)
	case uint8:
		return uint64(t)
	case uint16:
		return uint64(t)
	case uint32:
		return uint64(t)
	case uint64:
		return t
	case float32:
		return float64(t)
	case float64:
		return t
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339Nano)
	case json.Number:
		return t.String()
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		return NormalizeValue(rv.Elem().Interface())
	}

	if valuer, ok := v.(driver.Valuer); ok {
		dv, err := valuer.Value()
		if err == nil {
			switch t := dv.(type) {
			case nil:
				return nil
			case string:
				return t
			case []byte:
				return string(t)
			}
		}
	}

	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	if rv.IsValid() && rv.Kind() == reflect.Slice {
		out := make([]any, rv.Len())
		for i := range rv.Len() {
			out[i] = NormalizeValue(rv.Index(i).Interface())
		}
		return out
	}

	// json/jsonb objects arrive as maps. Keep the map so the wire format is
	// a JSON object; fmt.Sprint would print map[k:v].
	if rv.IsValid() && rv.Kind() == reflect.Map {
		if _, err := json.Marshal(v); err == nil {
			return v
		}
	}
	return fmt.Sprint(v)
}
