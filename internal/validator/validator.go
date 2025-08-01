package validator

import (
	"fmt"
	"reflect"
)

func Validate(name string, deps ...any) error {
	for _, dep := range deps {
		v := reflect.ValueOf(dep)

		if !v.IsValid() {
			return fmt.Errorf("missing required deps for component: %s, %v", name, v.Type())
		}

		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface ||
			v.Kind() == reflect.Slice || v.Kind() == reflect.Map ||
			v.Kind() == reflect.Chan) && v.IsNil() {
			return fmt.Errorf("missing required deps for component: %s, %v", name, v.Type())
		}

		if v.IsZero() {
			return fmt.Errorf("missing required deps for component: %s, %v", name, v.Type())
		}
	}

	return nil
}
